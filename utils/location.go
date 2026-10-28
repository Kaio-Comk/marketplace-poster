package utils

import (
	"strings"
	"time"

	"github.com/go-rod/rod"
)

func fillLocation(page *rod.Page, label *rod.Element, city string) {
	if strings.TrimSpace(city) == "" {
		return
	}

	cityName := city
	if idx := strings.Index(city, ","); idx != -1 {
		cityName = city[:idx]
	}
	targetCity := foldText(strings.TrimSpace(cityName))

	// Primeiro tenta o texto completo ("Cidade, UF"); se o Facebook não sugerir nada,
	// tenta só o nome da cidade. As sugestões chegam em tempo variável: espera por polling.
	in := label.MustElement("input")
	var options rod.Elements
	for _, texto := range []string{city, strings.TrimSpace(cityName)} {
		in.MustClick().MustSelectAllText().MustInput(texto)
		options = esperarSugestoes(page, 8*time.Second)
		if len(options) > 0 {
			break
		}
		Logger.Warn("LOCATION SUGGESTIONS NOT FOUND, RETRYING", "TYPED", texto)
	}
	if len(options) == 0 {
		Logger.Warn("LOCATION SUGGESTIONS NOT FOUND", "CITY", city)
		return
	}

	type suggestion struct {
		element   *rod.Element
		rawText   string
		line1Norm string
		line2Norm string
	}

	suggestions := make([]suggestion, 0, len(options))
	for _, opt := range options {
		raw := opt.MustText()
		rawLines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
		var lines []string
		for _, l := range rawLines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}

		var l1, l2 string
		if len(lines) > 0 {
			l1 = foldText(lines[0])
		}
		if len(lines) > 1 {
			l2 = foldText(lines[1])
		}

		suggestions = append(suggestions, suggestion{
			element:   opt,
			rawText:   raw,
			line1Norm: l1,
			line2Norm: l2,
		})
	}

	var chosen *suggestion

	// (a) a primeira cuja 1ª linha, normalizada, seja igual ao nome da cidade
	// E cuja 2ª linha seja "Cidade" ou "City" (ignore maiúsculas e acentos)
	for i := range suggestions {
		if suggestions[i].line1Norm == targetCity && (suggestions[i].line2Norm == "cidade" || suggestions[i].line2Norm == "city") {
			chosen = &suggestions[i]
			break
		}
	}

	// (b) senão a primeira cuja 1ª linha seja igual ao nome
	if chosen == nil {
		for i := range suggestions {
			if suggestions[i].line1Norm == targetCity {
				chosen = &suggestions[i]
				break
			}
		}
	}

	// (c) senão a primeira cuja 1ª linha comece com o nome
	if chosen == nil && targetCity != "" {
		for i := range suggestions {
			if strings.HasPrefix(suggestions[i].line1Norm, targetCity) {
				chosen = &suggestions[i]
				break
			}
		}
	}

	// (d) senão a primeira sugestão
	if chosen == nil {
		chosen = &suggestions[0]
	}

	chosenText := chosen.rawText
	chosen.element.MustClick()
	time.Sleep(1 * time.Second)

	Logger.Info("LOCATION SELECTED", "CITY", city, "OPTION", chosenText)
}

// esperarSugestoes consulta a página a cada 300 ms até aparecer alguma opção da lista
// de sugestões ou o limite estourar.
func esperarSugestoes(page *rod.Page, limite time.Duration) rod.Elements {
	fim := time.Now().Add(limite)
	for time.Now().Before(fim) {
		if opcoes, err := page.Elements(`[role="option"]`); err == nil && len(opcoes) > 0 {
			return opcoes
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}
