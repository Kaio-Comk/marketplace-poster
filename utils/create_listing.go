package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

var DryRun bool

// fmPausa dorme um intervalo aleatorio curto entre 300 e 900 ms.
func fmPausa() {
	d := time.Duration(300+rand.Intn(601)) * time.Millisecond
	time.Sleep(d)
}

// fmEsperar consulta cond a cada 300 ms ate limite ou sucesso.
func fmEsperar(cond func() bool, limite time.Duration) bool {
	fim := time.Now().Add(limite)
	for time.Now().Before(fim) {
		if cond() {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return cond()
}

// fmAchaLabel devolve o primeiro <label> visivel cuja primeira linha de texto
// (dobrada com foldText) seja igual a um dos nomes dobrados. Panic se nao achar.
func fmAchaLabel(page *rod.Page, nomes ...string) *rod.Element {
	alvos := make([]string, 0, len(nomes))
	for _, n := range nomes {
		alvos = append(alvos, foldText(n))
	}
	var achado *rod.Element
	ok := fmEsperar(func() bool {
		labels, err := page.Elements("label")
		if err != nil {
			return false
		}
		for _, l := range labels {
			txt, err := l.Text()
			if err != nil {
				continue
			}
			primeira := strings.TrimSpace(strings.SplitN(txt, "\n", 2)[0])
			if primeira == "" {
				continue
			}
			dobrada := foldText(primeira)
			for _, a := range alvos {
				if dobrada == a {
					achado = l
					return true
				}
			}
		}
		return false
	}, 10*time.Second)
	if !ok || achado == nil {
		panic(fmt.Sprintf("fmAchaLabel: label nao encontrado para %v", nomes))
	}
	return achado
}

// fmDigitar localiza input ou textarea dentro do label, clica, seleciona tudo e digita.
func fmDigitar(label *rod.Element, texto string) {
	campo, err := label.Element("input, textarea")
	if err != nil || campo == nil {
		panic("fmDigitar: campo de entrada nao encontrado dentro do label")
	}
	if err := campo.ScrollIntoView(); err != nil {
		Logger.Warn("fmDigitar: falha ao rolar ate o campo", "erro", err)
	}
	if err := campo.Click(proto.InputMouseButtonLeft, 1); err != nil {
		panic(fmt.Sprintf("fmDigitar: falha ao clicar no campo: %v", err))
	}
	if err := campo.SelectAllText(); err != nil {
		Logger.Warn("fmDigitar: falha ao selecionar texto", "erro", err)
	}
	if err := campo.Input(texto); err != nil {
		panic(fmt.Sprintf("fmDigitar: falha ao digitar: %v", err))
	}
}

// CreateListing preenche o formulario de criacao de anuncio do Marketplace.
func CreateListing(page *rod.Page, i Item, listingIndex int) {
	Logger.Info("iniciando criacao de anuncio", "indice", listingIndex)

	// (1) navegar e esperar o primeiro input[type=file]
	Logger.Info("navegando para a pagina de criacao")
	if err := page.Timeout(60 * time.Second).Navigate("https://www.facebook.com/marketplace/create/item"); err != nil {
		panic(fmt.Sprintf("CreateListing: falha ao navegar: %v", err))
	}
	page.WaitLoad()
	fmPausa()

	var arquivo *rod.Element
	ok := fmEsperar(func() bool {
		els, err := page.Elements("input[type=file]")
		if err != nil || len(els) == 0 {
			return false
		}
		arquivo = els[0]
		return true
	}, 30*time.Second)
	if !ok || arquivo == nil {
		panic("CreateListing: input[type=file] nao apareceu")
	}

	// (2) fotos
	if len(i.Images) > 0 {
		Logger.Info("enviando fotos", "quantidade", len(i.Images))
		limite := len(i.Images)
		if limite > 6 {
			limite = 6
		}
		if err := arquivo.SetFiles(i.Images[:limite]); err != nil {
			panic(fmt.Sprintf("CreateListing: falha ao definir arquivos: %v", err))
		}
		okFoto := fmEsperar(func() bool {
			els, err := page.Elements("div[role=button][aria-label]")
			if err != nil {
				return false
			}
			for _, e := range els {
				al, err := e.Attribute("aria-label")
				if err != nil || al == nil {
					continue
				}
				v := *al
				if strings.HasPrefix(v, "Remover foto") || strings.HasPrefix(v, "Remove photo") {
					return true
				}
			}
			return false
		}, 60*time.Second)
		if !okFoto {
			panic("CreateListing: botao de remover foto nao apareceu apos upload")
		}
	}
	fmPausa()

	// (3) Titulo e Preco
	Logger.Info("preenchendo titulo e preco")
	labelTitulo := fmAchaLabel(page, "título", "title")
	fmDigitar(labelTitulo, i.Title)
	fmPausa()

	labelPreco := fmAchaLabel(page, "preço", "price", "preco")
	fmDigitar(labelPreco, i.Price)
	fmPausa()

	// (4) Categoria e Estado
	Logger.Info("selecionando categoria e estado")
	fmEscolherCategoria(page, i.Category)
	fmPausa()
	fmEscolherEstado(page, i.Condition)
	fmPausa()

	// (5) garantir secao "Mais detalhes" aberta
	Logger.Info("verificando secao de mais detalhes")
	_, errDesc := page.Element("label")
	_ = errDesc
	temDescricao := false
	{
		labels, err := page.Elements("label")
		if err == nil {
			for _, l := range labels {
				txt, err := l.Text()
				if err != nil {
					continue
				}
				primeira := strings.TrimSpace(strings.SplitN(txt, "\n", 2)[0])
				d := foldText(primeira)
				if d == foldText("descrição") || d == foldText("description") || d == foldText("descricao") {
					temDescricao = true
					break
				}
			}
		}
	}
	if !temDescricao {
		Logger.Info("abrindo secao de mais detalhes")
		abriu := false
		els, err := page.Elements("div[role=button]")
		if err == nil {
			for _, e := range els {
				txt, err := e.Text()
				if err != nil {
					continue
				}
				t := strings.TrimSpace(txt)
				if strings.HasPrefix(t, "Mais detalhes") || strings.HasPrefix(t, "More details") {
					if err := e.Click(proto.InputMouseButtonLeft, 1); err != nil {
						panic(fmt.Sprintf("CreateListing: falha ao clicar em mais detalhes: %v", err))
					}
					abriu = true
					break
				}
			}
		}
		if !abriu {
			Logger.Warn("CreateListing: botao de mais detalhes nao encontrado")
		}
		fmPausa()
	}

	// (6) Descricao
	Logger.Info("preenchendo descricao")
	labelDesc := fmAchaLabel(page, "descrição", "description", "descricao")
	fmDigitar(labelDesc, i.Description)
	fmPausa()

	// (7) Etiquetas
	if len(i.Tags) > 0 {
		Logger.Info("preenchendo etiquetas", "quantidade", len(i.Tags))
		var labelTags *rod.Element
		func() {
			defer func() {
				if r := recover(); r != nil {
					labelTags = nil
				}
			}()
			labelTags = fmAchaLabel(page, "identificações de produtos", "product tags")
		}()
		if labelTags == nil {
			Logger.Warn("CreateListing: label de etiquetas nao encontrado, pulando")
		} else {
			area, err := labelTags.Element("textarea")
			if err != nil || area == nil {
				Logger.Warn("CreateListing: textarea de etiquetas nao encontrado, pulando")
			} else {
				for _, tag := range i.Tags {
					if err := area.Click(proto.InputMouseButtonLeft, 1); err != nil {
						panic(fmt.Sprintf("CreateListing: falha ao clicar no campo de etiquetas: %v", err))
					}
					if err := page.Keyboard.Type(input.Enter); err != nil {
						panic(fmt.Sprintf("CreateListing: falha ao digitar etiqueta: %v", err))
					}
					_ = tag
					fmPausa()
				}
			}
		}
	}

	// (8) Localizacao
	if strings.TrimSpace(i.Location) != "" {
		Logger.Info("preenchendo localizacao")
		labelLoc := fmAchaLabel(page, "localização", "localizacao", "location")
		fillLocation(page, labelLoc, i.Location)
		fmPausa()
	}

	// (9) avancar
	Logger.Info("avancando para a proxima tela")
	fmSeguinte(page)
	fmPausa()
	fmSegundaTela(page)
	fmPausa()

	Logger.Info("criacao de anuncio concluida", "indice", listingIndex)
}
