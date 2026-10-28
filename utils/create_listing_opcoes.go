package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// fmPrimeiraLinha devolve a primeira linha não vazia de s, já com TrimSpace.
func fmPrimeiraLinha(s string) string {
	for _, linha := range strings.Split(s, "\n") {
		linha = strings.TrimSpace(linha)
		if linha != "" {
			return linha
		}
	}
	return ""
}

// fmEscolherCategoria abre o combobox de categoria e clica no item cuja
// primeira linha dobrada bate com alguma grafia aceita para valor.
func fmEscolherCategoria(page *rod.Page, valor string) {
	label := fmAchaLabel(page, "categoria", "category")
	if label == nil {
		panic("fmEscolherCategoria: label de categoria nao encontrado")
	}
	if err := label.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmEscolherCategoria: falha ao clicar no label: %v", err))
	}
	fmPausa()

	alvos := optionCandidates(valor)
	var vistos []string
	var escolhido *rod.Element

	fmEsperar(func() bool {
		itens, err := page.ElementsByJS(rod.Eval(`() => Array.from(document.querySelectorAll('div[role="button"]')).filter(e => e.offsetParent !== null)`))
		if err != nil {
			return false
		}
		for _, it := range itens {
			txt, err := it.Text()
			if err != nil {
				continue
			}
			primeira := fmPrimeiraLinha(txt)
			if primeira == "" {
				continue
			}
			dobrada := foldText(primeira)
			vistos = append(vistos, primeira)
			if foldedContainsAny(dobrada, alvos...) {
				escolhido = it
				return true
			}
		}
		return false
	}, 5*time.Second)

	if escolhido == nil {
		panic(fmt.Sprintf("fmEscolherCategoria: nenhuma categoria casou com %q; vistas: %v", valor, vistos))
	}

	if err := escolhido.ScrollIntoView(); err != nil {
		Logger.Warn("fmEscolherCategoria: falha no ScrollIntoView", "erro", err)
	}
	fmPausa()
	if err := escolhido.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmEscolherCategoria: falha ao clicar no item: %v", err))
	}
	fmPausa()

	confirmado := fmEsperar(func() bool {
		txt, err := label.Text()
		if err != nil {
			return false
		}
		linhas := strings.Split(strings.TrimSpace(txt), "\n")
		return len(linhas) >= 2 && strings.TrimSpace(linhas[1]) != ""
	}, 5*time.Second)
	if !confirmado {
		Logger.Warn("fmEscolherCategoria: combobox nao exibiu o valor escolhido", "valor", valor)
	}
}

// fmEscolherEstado abre o listbox de estado e clica na opcao cuja primeira
// linha dobrada bate com alguma grafia aceita para valor.
func fmEscolherEstado(page *rod.Page, valor string) {
	label := fmAchaLabel(page, "estado", "condition", "condição")
	if label == nil {
		panic("fmEscolherEstado: label de estado nao encontrado")
	}
	if err := label.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmEscolherEstado: falha ao clicar no label: %v", err))
	}
	fmPausa()

	alvos := optionCandidates(valor)
	var vistos []string
	var escolhido *rod.Element

	fmEsperar(func() bool {
		opcoes, err := page.ElementsByJS(rod.Eval(`() => Array.from(document.querySelectorAll('[role="listbox"] [role="option"]')).filter(e => e.offsetParent !== null)`))
		if err != nil {
			return false
		}
		for _, op := range opcoes {
			txt, err := op.Text()
			if err != nil {
				continue
			}
			primeira := fmPrimeiraLinha(txt)
			if primeira == "" {
				continue
			}
			dobrada := foldText(primeira)
			vistos = append(vistos, primeira)
			if foldedContainsAny(dobrada, alvos...) {
				escolhido = op
				return true
			}
		}
		return false
	}, 5*time.Second)

	if escolhido == nil {
		panic(fmt.Sprintf("fmEscolherEstado: nenhum estado casou com %q; vistas: %v", valor, vistos))
	}

	if err := escolhido.ScrollIntoView(); err != nil {
		Logger.Warn("fmEscolherEstado: falha no ScrollIntoView", "erro", err)
	}
	fmPausa()
	if err := escolhido.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmEscolherEstado: falha ao clicar na opcao: %v", err))
	}
	fmPausa()

	confirmado := fmEsperar(func() bool {
		txt, err := label.Text()
		if err != nil {
			return false
		}
		linhas := strings.Split(strings.TrimSpace(txt), "\n")
		return len(linhas) >= 2 && strings.TrimSpace(linhas[1]) != ""
	}, 5*time.Second)
	if !confirmado {
		Logger.Warn("fmEscolherEstado: combobox nao exibiu o valor escolhido", "valor", valor)
	}
}
