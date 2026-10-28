package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"marketplace-poster/utils"
)

// dormirFatias dorme em fatias de no máximo 30s, respeitando o contexto.
func dormirFatias(ctx context.Context, d time.Duration) bool {
	const fatia = 30 * time.Second
	restante := d
	for restante > 0 {
		passo := restante
		if passo > fatia {
			passo = fatia
		}
		t := time.NewTimer(passo)
		select {
		case <-ctx.Done():
			t.Stop()
			return false
		case <-t.C:
		}
		restante -= passo
	}
	return true
}

// dentroJanela verifica se a hora local de t está em [ini, fim).
// Se fim <= ini, a janela cobre 24h (sempre true).
func dentroJanela(t time.Time, ini, fim int) bool {
	if fim <= ini {
		return true
	}
	h := t.Hour()
	return h >= ini && h < fim
}

// proximoInicio devolve o próximo instante local com hora == ini:00 após t.
func proximoInicio(t time.Time, ini int) time.Time {
	loc := t.Location()
	cand := time.Date(t.Year(), t.Month(), t.Day(), ini, 0, 0, 0, loc)
	if !cand.After(t) {
		cand = cand.Add(24 * time.Hour)
	}
	return cand
}

// backoffFalhas devolve o intervalo de espera conforme o número de falhas.
func backoffFalhas(n int) time.Duration {
	switch {
	case n <= 1:
		return 30 * time.Minute
	case n == 2:
		return 1 * time.Hour
	case n == 3:
		return 2 * time.Hour
	case n == 4:
		return 4 * time.Hour
	default:
		return 12 * time.Hour
	}
}

// montarItem converte planilha + plano em utils.Item.
func montarItem(p *utils.Planilha, plano utils.Plano) utils.Item {
	preco := strings.Replace(fmt.Sprintf("%.2f", p.Preco), ".", ",", 1)

	var tags []string
	for _, t := range strings.Split(p.Etiquetas, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	return utils.Item{
		Images:      []string{filepath.Join(p.PastaFotos, plano.Foto.Texto)},
		Title:       plano.Titulo.Texto,
		Price:       preco,
		Category:    strings.ToLower(p.Categoria),
		Condition:   strings.ToLower(p.Estado),
		Description: plano.Descricao.Texto,
		Tags:        tags,
		Location:    plano.Cidade.Nome + ", " + plano.Cidade.UF,
	}
}

// registrarPublicado obtém link/id da página e anexa o registro.
func registrarPublicado(page *rod.Page, planilhaPath string, plano utils.Plano) {
	link, id := utils.LatestListingLink(page)
	if id == "" {
		id = fmt.Sprintf("A%d", time.Now().Unix())
	}

	pub := utils.Publicado{
		ID:        id,
		Data:      time.Now(),
		Foto:      plano.Foto.ID,
		Titulo:    plano.Titulo.ID,
		Descricao: plano.Descricao.ID,
		Cidade:    plano.Cidade.Nome,
		UF:        plano.Cidade.UF,
		Link:      link,
		Status:    "Ativo",
	}

	if err := utils.AnexarPublicado(planilhaPath, pub); err != nil {
		utils.Logger.Warn("falha ao anexar publicado", "erro", err.Error())
		savePendente(pub)
	}
}

// abrirNavegador chama utils.Login protegendo contra panic.
func abrirNavegador(chromePath string) (b *rod.Browser, p *rod.Page) {
	defer func() {
		if r := recover(); r != nil {
			utils.Logger.Warn("panic ao abrir navegador", "erro", fmt.Sprintf("%v", r))
			b = nil
			p = nil
		}
	}()
	b, p = utils.Login(chromePath)
	return b, p
}

// fecharNavegador fecha o browser ignorando erros e panics.
func fecharNavegador(b *rod.Browser) {
	if b == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	_ = b.Close()
}
