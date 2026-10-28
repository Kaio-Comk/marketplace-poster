package main

import (
	"encoding/csv"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"marketplace-poster/utils"

	"github.com/go-rod/rod"
)

const pendentesFile = "pendentes.csv"

// runAuto publishes today's quota from the spreadsheet: next cities in population order,
// balanced photo/title/description variants, a random pause between ads.
func runAuto(page *rod.Page, planilhaPath string, minMin, maxMin int, dryRun bool) {
	if !dryRun {
		flushPendentes(planilhaPath)
	}

	p, err := utils.LerPlanilha(planilhaPath)
	if err != nil {
		utils.Logger.Fatal("FAILED TO READ SPREADSHEET", "PATH", planilhaPath, "ERR", err)
	}

	// Ads waiting in pendentes.csv count as published, so a busy spreadsheet never causes repeats.
	p.Publicados = append(p.Publicados, lerPendentes()...)

	restante := p.LimiteDia - publicadosHoje(p)
	utils.Logger.Info("DAILY QUOTA", "LIMIT", p.LimiteDia, "REMAINING", restante)
	if restante <= 0 {
		utils.Logger.Info("QUOTA ALREADY REACHED TODAY, NOTHING TO DO")
		return
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7))
	planos := utils.ProximosAnuncios(p, restante, rng)
	if len(planos) == 0 {
		utils.Logger.Warn("NOTHING TO PUBLISH: CHECK ACTIVE PHOTOS, TITLES, DESCRIPTIONS AND CITIES")
		return
	}

	for i, plano := range planos {
		item := utils.Item{
			Images:      []string{filepath.Join(p.PastaFotos, plano.Foto.Texto)},
			Title:       plano.Titulo.Texto,
			Price:       strings.Replace(fmt.Sprintf("%.2f", p.Preco), ".", ",", 1),
			Category:    strings.ToLower(p.Categoria),
			Condition:   strings.ToLower(p.Estado),
			Description: plano.Descricao.Texto,
			Location:    plano.Cidade.Nome + ", " + plano.Cidade.UF,
		}
		for _, tag := range strings.Split(p.Etiquetas, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				item.Tags = append(item.Tags, tag)
			}
		}

		if dryRun {
			utils.Logger.Info("WOULD PUBLISH", "CITY", item.Location, "PHOTO", plano.Foto.ID, "TITLE", plano.Titulo.ID, "DESCRIPTION", plano.Descricao.ID, "PRICE", item.Price, "IMAGE", item.Images[0])
			continue
		}

		utils.Logger.Info("PUBLISHING", "CITY", item.Location, "PHOTO", plano.Foto.ID, "TITLE", plano.Titulo.ID, "DESCRIPTION", plano.Descricao.ID)
		if !publishOne(page, item) {
			utils.Logger.Warn("STOPPING FOR TODAY AFTER A FAILURE")
			return
		}

		link, id := utils.LatestListingLink(page)
		if id == "" {
			id = fmt.Sprintf("A%d", time.Now().Unix())
		}
		pub := utils.Publicado{
			ID: id, Data: time.Now(), Foto: plano.Foto.ID, Titulo: plano.Titulo.ID, Descricao: plano.Descricao.ID,
			Cidade: plano.Cidade.Nome, UF: plano.Cidade.UF, Link: link, Status: "Ativo",
		}
		if err := utils.AnexarPublicado(planilhaPath, pub); err != nil {
			utils.Logger.Warn("SPREADSHEET BUSY, SAVING TO pendentes.csv", "ERR", err)
			savePendente(pub)
		}

		if i == len(planos)-1 {
			break
		}
		pausa := time.Duration(minMin+rand.IntN(maxMin-minMin+1)) * time.Minute
		utils.Logger.Info("WAITING BEFORE NEXT AD", "MINUTES", pausa.Minutes())
		time.Sleep(pausa)
	}
	utils.Logger.Info("DAILY QUOTA DONE")
}

// publishOne reports false when Facebook refused (limit reached, errors): the run stops.
func publishOne(page *rod.Page, item utils.Item) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			utils.Logger.Error("PUBLISH FAILED", "ERR", r)
			ok = false
		}
	}()
	utils.CreateListing(page, item, 0)
	return true
}

func publicadosHoje(p *utils.Planilha) int {
	hoje := time.Now().Format("2006-01-02")
	n := 0
	for _, pub := range p.Publicados {
		if pub.Data.Local().Format("2006-01-02") == hoje {
			n++
		}
	}
	return n
}

func savePendente(pub utils.Publicado) {
	f, err := os.OpenFile(pendentesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		utils.Logger.Error("COULD NOT SAVE pendentes.csv", "ERR", err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{pub.ID, pub.Data.Format(time.RFC3339), pub.Foto, pub.Titulo, pub.Descricao, pub.Cidade, pub.UF, pub.Link, pub.Status})
	w.Flush()
}

func flushPendentes(planilhaPath string) {
	f, err := os.Open(pendentesFile)
	if err != nil {
		return
	}
	rows, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil || len(rows) == 0 {
		return
	}
	var left [][]string
	for _, r := range rows {
		if len(r) < 9 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, r[1])
		pub := utils.Publicado{ID: r[0], Data: t, Foto: r[2], Titulo: r[3], Descricao: r[4], Cidade: r[5], UF: r[6], Link: r[7], Status: r[8]}
		if utils.AnexarPublicado(planilhaPath, pub) != nil {
			left = append(left, r)
		}
	}
	if len(left) == 0 {
		_ = os.Remove(pendentesFile)
		return
	}
	out, err := os.Create(pendentesFile)
	if err != nil {
		return
	}
	defer out.Close()
	w := csv.NewWriter(out)
	_ = w.WriteAll(left)
}

func lerPendentes() []utils.Publicado {
	f, err := os.Open(pendentesFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil
	}

	var pubs []utils.Publicado
	for _, r := range rows {
		if len(r) < 9 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, r[1])
		pubs = append(pubs, utils.Publicado{ID: r[0], Data: t, Foto: r[2], Titulo: r[3], Descricao: r[4], Cidade: r[5], UF: r[6], Link: r[7], Status: r[8]})
	}
	return pubs
}
