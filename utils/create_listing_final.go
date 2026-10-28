package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// fmSeguinte localiza o botao "Seguinte"/"Next", espera ele habilitar
// (aria-disabled diferente de "true") por ate 30s e clica. Em seguida
// aguarda a segunda tela aparecer (botoes "Publicar"/"Publish") por ate 20s.
// Faz panic com mensagem clara em qualquer falha.
func fmSeguinte(page *rod.Page) {
	botao := fmAcharBotaoPorRotulo(page, "Seguinte", "Next")
	if botao == nil {
		panic("fmSeguinte: botao Seguinte/Next nao encontrado na primeira tela")
	}

	habilitou := fmEsperar(func() bool {
		des, err := botao.Attribute("aria-disabled")
		if err != nil || des == nil {
			return true
		}
		return strings.ToLower(strings.TrimSpace(*des)) != "true"
	}, 30*time.Second)

	if !habilitou {
		panic("fmSeguinte: botao Seguinte nao habilitou em 30s; provavelmente faltam campos obrigatorios")
	}

	if err := botao.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmSeguinte: falha ao clicar em Seguinte: %v", err))
	}

	apareceu := fmEsperar(func() bool {
		return fmAcharBotaoPorRotulo(page, "Publicar", "Publish") != nil
	}, 20*time.Second)

	if !apareceu {
		panic("fmSeguinte: segunda tela nao apareceu (botao Publicar/Publish ausente) em 20s")
	}
}

// fmSegundaTela trata a segunda etapa do fluxo de criacao de anuncio:
// marca todas as caixas div[role=checkbox] visiveis que nao estejam
// com aria-checked=true, respeitando DryRun e, quando aplicavel, clica
// em Publicar e espera a publicacao terminar por ate 90s.
func fmSegundaTela(page *rod.Page) {
	caixas, err := page.Elements(`div[role="checkbox"]`)
	if err != nil {
		Logger.Warn("fmSegundaTela: falha ao listar caixas", "erro", err)
	}

	marcadas := 0
	for _, cx := range caixas {
		estado, _ := cx.Attribute("aria-checked")
		if estado != nil && strings.ToLower(strings.TrimSpace(*estado)) == "true" {
			continue
		}
		if err := cx.Click(proto.InputMouseButtonLeft, 1); err != nil {
			Logger.Warn("fmSegundaTela: falha ao clicar em checkbox", "erro", err)
			continue
		}
		marcadas++
		fmPausa()
	}

	if marcadas > 0 {
		Logger.Info("fmSegundaTela: caixas marcadas", "quantidade", marcadas)
	} else {
		Logger.Info("fmSegundaTela: nenhuma caixa para marcar")
	}

	if DryRun {
		Logger.Info("ENSAIO: parando antes de publicar")
		return
	}

	publicar := fmAcharBotaoPorRotulo(page, "Publicar", "Publish")
	if publicar == nil {
		panic("fmSegundaTela: botao Publicar/Publish nao encontrado")
	}

	if err := publicar.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmSegundaTela: falha ao clicar em Publicar: %v", err))
	}

	concluiu := fmEsperar(func() bool {
		info, err := page.Info()
		if err == nil && info != nil && !strings.Contains(info.URL, "/marketplace/create") {
			return true
		}
		return fmAcharBotaoPorRotulo(page, "Publicar", "Publish") == nil
	}, 90*time.Second)

	if !concluiu {
		panic("fmSegundaTela: publicacao nao concluiu em 90s")
	}
}

// fmAcharBotaoPorRotulo procura um div[role=button] cujo aria-label
// corresponda (via foldText) a qualquer um dos rotulos informados.
// Retorna nil se nao encontrar.
func fmAcharBotaoPorRotulo(page *rod.Page, rotulos ...string) *rod.Element {
	alvos := make([]string, 0, len(rotulos))
	for _, r := range rotulos {
		alvos = append(alvos, foldText(r))
	}

	candidatos, err := page.Elements(`div[role="button"]`)
	if err != nil {
		return nil
	}
	for _, el := range candidatos {
		rotulo, err := el.Attribute("aria-label")
		if err != nil || rotulo == nil {
			continue
		}
		atual := foldText(*rotulo)
		for _, alvo := range alvos {
			if atual == alvo {
				return el
			}
		}
	}
	return nil
}
