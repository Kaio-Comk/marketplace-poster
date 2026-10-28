package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func LerPlanilha(path string) (*Planilha, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("planilha em uso ou ilegível: %w", err)
	}
	defer f.Close()

	p := &Planilha{}

	cfg := map[string]string{}
	rows, err := f.GetRows("Config", excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("aba Config não encontrada: %w", err)
	}
	for i, r := range rows {
		if i == 0 || len(r) == 0 {
			continue
		}
		key := strings.TrimSpace(r[0])
		val := ""
		if len(r) > 1 {
			val = strings.TrimSpace(r[1])
		}
		cfg[key] = val
	}

	get := func(k string) (string, error) {
		v, ok := cfg[k]
		if !ok {
			return "", fmt.Errorf("item de Config não encontrado: %q", k)
		}
		return v, nil
	}

	precoStr, err := get("Preço (R$)")
	if err != nil {
		return nil, err
	}
	if precoStr != "" {
		precoStr = strings.ReplaceAll(precoStr, ",", ".")
		p.Preco, _ = strconv.ParseFloat(precoStr, 64)
	}

	if p.Categoria, err = get("Categoria do anúncio"); err != nil {
		return nil, err
	}
	if p.Estado, err = get("Estado do produto"); err != nil {
		return nil, err
	}
	if p.Etiquetas, err = get("Etiquetas (separadas por vírgula)"); err != nil {
		return nil, err
	}
	if p.PastaFotos, err = get("Pasta das fotos"); err != nil {
		return nil, err
	}

	inicioStr, err := get("Data de início da publicação")
	if err != nil {
		return nil, err
	}
	limiteStr, err := get("Limite diário inicial (anúncios por dia)")
	if err != nil {
		return nil, err
	}
	limiteInicial := 0
	if limiteStr != "" {
		limiteInicial, _ = strconv.Atoi(strings.TrimSpace(limiteStr))
	}

	var inicio time.Time
	if inicioStr != "" {
		if t, e := time.Parse("02/01/2006", inicioStr); e == nil {
			inicio = t
		} else if serial, e2 := strconv.ParseFloat(inicioStr, 64); e2 == nil {
			if t, e3 := excelize.ExcelDateToTime(serial, false); e3 == nil {
				inicio = t
			}
		}
	}

	if inicio.IsZero() {
		p.LimiteDia = limiteInicial
	} else {
		dias := int(time.Since(inicio).Hours() / 24)
		switch {
		case dias < 7:
			p.LimiteDia = 3
		case dias < 14:
			p.LimiteDia = 5
		case dias < 21:
			p.LimiteDia = 8
		default:
			p.LimiteDia = 12
		}
	}
	if p.LimiteDia > 15 {
		p.LimiteDia = 15
	}

	lerVariantes := func(aba, prefixo string) ([]Variante, error) {
		var out []Variante
		rows, err := f.GetRows(aba, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("aba %s não encontrada: %w", aba, err)
		}
		for i := 1; i < len(rows); i++ {
			r := rows[i]
			texto := ""
			if len(r) > 1 {
				texto = strings.TrimSpace(r[1])
			}
			if texto == "" {
				break
			}
			ativa := ""
			if len(r) > 2 {
				ativa = strings.TrimSpace(r[2])
			}
			if !strings.EqualFold(ativa, "Sim") {
				continue
			}
			id := ""
			if len(r) > 0 {
				id = strings.TrimSpace(r[0])
			}
			if id == "" {
				id = fmt.Sprintf("%s%d", prefixo, i)
			}
			out = append(out, Variante{ID: id, Texto: texto})
		}
		return out, nil
	}

	if p.Fotos, err = lerVariantes("Fotos", "I"); err != nil {
		return nil, err
	}
	if p.Titulos, err = lerVariantes("Títulos", "T"); err != nil {
		return nil, err
	}
	if p.Descricoes, err = lerVariantes("Descrições", "D"); err != nil {
		return nil, err
	}

	cidRows, err := f.GetRows("Cidades", excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("aba Cidades não encontrada: %w", err)
	}
	for i := 1; i < len(cidRows); i++ {
		r := cidRows[i]
		nome := ""
		if len(r) > 1 {
			nome = strings.TrimSpace(r[1])
		}
		if nome == "" {
			continue
		}
		uf := ""
		if len(r) > 2 {
			uf = strings.TrimSpace(r[2])
		}
		pop := 0
		if len(r) > 3 {
			pop, _ = strconv.Atoi(strings.TrimSpace(r[3]))
		}
		ativa := false
		if len(r) > 4 {
			ativa = strings.EqualFold(strings.TrimSpace(r[4]), "Sim")
		}
		p.Cidades = append(p.Cidades, Cidade{Nome: nome, UF: uf, Populacao: pop, Ativa: ativa})
	}

	pubRows, err := f.GetRows("Anúncios publicados", excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("aba Anúncios publicados não encontrada: %w", err)
	}
	for i := 1; i < len(pubRows); i++ {
		r := pubRows[i]
		id := ""
		if len(r) > 0 {
			id = strings.TrimSpace(r[0])
		}
		if id == "" {
			break
		}
		cell := func(n int) string {
			if len(r) > n {
				return strings.TrimSpace(r[n])
			}
			return ""
		}
		status := cell(8)
		if strings.EqualFold(status, "Exemplo") {
			continue
		}
		pub := Publicado{
			ID:        id,
			Foto:      cell(2),
			Titulo:    cell(3),
			Descricao: cell(4),
			Cidade:    cell(5),
			UF:        cell(6),
			Link:      cell(7),
			Status:    status,
		}
		dataStr := cell(1)
		if dataStr != "" {
			if t, e := time.Parse("02/01/2006", dataStr); e == nil {
				pub.Data = t
			} else if serial, e2 := strconv.ParseFloat(dataStr, 64); e2 == nil {
				if t, e3 := excelize.ExcelDateToTime(serial, false); e3 == nil {
					pub.Data = t
				}
			}
		}
		p.Publicados = append(p.Publicados, pub)
	}

	return p, nil
}

func AnexarPublicado(path string, p Publicado) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("planilha em uso ou ilegível: %w", err)
	}
	defer f.Close()

	const aba = "Anúncios publicados"
	rows, err := f.GetRows(aba, excelize.Options{RawCellValue: true})
	if err != nil {
		return fmt.Errorf("aba %s não encontrada: %w", aba, err)
	}

	linha := len(rows) + 1
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if len(r) == 0 || strings.TrimSpace(r[0]) == "" {
			linha = i + 1
			break
		}
	}

	style, _ := f.NewStyle(&excelize.Style{NumFmt: 14})

	vals := []interface{}{p.ID, p.Data, p.Foto, p.Titulo, p.Descricao, p.Cidade, p.UF, p.Link, p.Status}
	for i, v := range vals {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s%d", col, linha)
		if i == 1 {
			f.SetCellValue(aba, cell, v)
			f.SetCellStyle(aba, cell, cell, style)
		} else {
			f.SetCellValue(aba, cell, v)
		}
	}

	if err := f.Save(); err != nil {
		return fmt.Errorf("planilha em uso ou ilegível: %w", err)
	}
	return nil
}
