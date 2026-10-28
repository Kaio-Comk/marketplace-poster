package utils

import (
	"math/rand/v2"
	"strings"
)

type Plano struct {
	Cidade    Cidade
	Foto      Variante
	Titulo    Variante
	Descricao Variante
}

func ProximosAnuncios(p *Planilha, n int, rng *rand.Rand) []Plano {
	if len(p.Fotos) == 0 || len(p.Titulos) == 0 || len(p.Descricoes) == 0 {
		return nil
	}

	candidatas := make([]Cidade, 0, n)
	for _, c := range p.Cidades {
		if !c.Ativa {
			continue
		}
		if cidadePublicada(p.Publicados, c) {
			continue
		}
		candidatas = append(candidatas, c)
		if len(candidatas) == n {
			break
		}
	}
	if len(candidatas) == 0 {
		return nil
	}

	usoFoto := contarUso(p.Publicados, func(pub Publicado) string { return pub.Foto })
	usoTitulo := contarUso(p.Publicados, func(pub Publicado) string { return pub.Titulo })
	usoDescricao := contarUso(p.Publicados, func(pub Publicado) string { return pub.Descricao })

	triplas := make(map[string]bool)
	for _, pub := range p.Publicados {
		triplas[chaveTripla(pub.Foto, pub.Titulo, pub.Descricao)] = true
	}

	planos := make([]Plano, 0, len(candidatas))
	for _, c := range candidatas {
		foto := escolher(p.Fotos, usoFoto, rng)
		titulo := escolher(p.Titulos, usoTitulo, rng)
		descricao := escolher(p.Descricoes, usoDescricao, rng)

		for i := 0; i < 30; i++ {
			if !triplas[chaveTripla(foto.ID, titulo.ID, descricao.ID)] {
				break
			}
			foto = escolherNivel(p.Fotos, usoFoto, rng)
			titulo = escolherNivel(p.Titulos, usoTitulo, rng)
			descricao = escolherNivel(p.Descricoes, usoDescricao, rng)
		}

		usoFoto[foto.ID]++
		usoTitulo[titulo.ID]++
		usoDescricao[descricao.ID]++
		triplas[chaveTripla(foto.ID, titulo.ID, descricao.ID)] = true

		planos = append(planos, Plano{Cidade: c, Foto: foto, Titulo: titulo, Descricao: descricao})
	}
	return planos
}

func cidadePublicada(pubs []Publicado, c Cidade) bool {
	for _, pub := range pubs {
		if strings.EqualFold(pub.Cidade, c.Nome) && strings.EqualFold(pub.UF, c.UF) {
			return true
		}
	}
	return false
}

func contarUso(pubs []Publicado, campo func(Publicado) string) map[string]int {
	m := make(map[string]int)
	for _, pub := range pubs {
		m[campo(pub)]++
	}
	return m
}

func chaveTripla(foto, titulo, descricao string) string {
	return foto + "\x00" + titulo + "\x00" + descricao
}

func escolher(vs []Variante, uso map[string]int, rng *rand.Rand) Variante {
	min := -1
	for _, v := range vs {
		c := uso[v.ID]
		if min == -1 || c < min {
			min = c
		}
	}
	cands := make([]Variante, 0, len(vs))
	for _, v := range vs {
		if uso[v.ID] == min {
			cands = append(cands, v)
		}
	}
	return cands[rng.IntN(len(cands))]
}

func escolherNivel(vs []Variante, uso map[string]int, rng *rand.Rand) Variante {
	min := -1
	for _, v := range vs {
		c := uso[v.ID]
		if min == -1 || c < min {
			min = c
		}
	}
	limite := min + 1
	cands := make([]Variante, 0, len(vs))
	for _, v := range vs {
		if uso[v.ID] <= limite {
			cands = append(cands, v)
		}
	}
	return cands[rng.IntN(len(cands))]
}
