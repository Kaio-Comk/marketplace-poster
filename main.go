package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"

	"marketplace-poster/utils"
)

func main() {
	itemsFlag := flag.String("items", "", "pasta com as imagens dos itens (modo manual)")
	cidadeFlag := flag.String("cidade", "", "cidade padrao para os anuncios")
	autoFlag := flag.Bool("auto", false, "modo automatico a partir da planilha")
	planilhaFlag := flag.String("planilha", "", "caminho da planilha de anuncios")
	daemonFlag := flag.Bool("daemon", false, "modo daemon com janela de horario")
	janelaIni := flag.Int("janela-ini", 8, "hora inicial da janela do daemon")
	janelaFim := flag.Int("janela-fim", 18, "hora final da janela do daemon")
	pausaMin := flag.Int("pausa-min", 20, "pausa minima em minutos entre ciclos")
	pausaMax := flag.Int("pausa-max", 40, "pausa maxima em minutos entre ciclos")
	simularFlag := flag.Bool("simular", false, "simula o modo automatico sem abrir navegador")
	ensaioFlag := flag.Bool("ensaio", false, "preenche o formulario mas NAO clica em Publicar")
	flag.Parse()
	utils.DryRun = *ensaioFlag

	cfg, err := utils.LoadConfig(".")
	if err != nil {
		log.Fatal("falha ao carregar configuracao", "erro", err)
	}

	planilha := *planilhaFlag
	if planilha == "" {
		planilha = cfg.PLANILHA_PATH
	}

	if (*autoFlag || *daemonFlag) && planilha == "" {
		log.Fatal("informe -planilha ou defina PLANILHA_PATH no ambiente")
	}

	if *daemonFlag {
		runDaemon(planilha, cfg.CHROME_PATH, *pausaMin, *janelaIni, *janelaFim)
		return
	}

	if *autoFlag && *simularFlag {
		runAuto(nil, planilha, *pausaMin, *pausaMax, true)
		return
	}

	browser, page := utils.Login(cfg.CHROME_PATH)
	defer func() {
		page.MustClose()
		browser.MustClose()
	}()

	if *autoFlag {
		runAuto(page, planilha, *pausaMin, *pausaMax, false)
		return
	}

	// Modo manual
	pasta := *itemsFlag
	if pasta == "" {
		pasta = cfg.ITEMS_PATH
	}
	if pasta == "" {
		pasta = "./itens"
	}
	pastaAbs, err := filepath.Abs(pasta)
	if err != nil {
		log.Fatal("nao foi possivel resolver o caminho da pasta de itens", "erro", err)
	}

	itens, err := utils.GetItems(pastaAbs)
	if err != nil {
		log.Fatal("falha ao carregar itens", "erro", err)
	}

	cidade := *cidadeFlag
	if cidade == "" {
		cidade = cfg.DEFAULT_CITY
	}
	for i := range itens {
		if itens[i].Location == "" {
			itens[i].Location = cidade
		}
	}

	ctx := context.Background()
	reescritor, err := utils.NewGeminiDescriptionRewriter(ctx, cfg)
	if err != nil {
		log.Warn("reescritor Gemini indisponivel, seguindo sem reescrita", "erro", err)
		fmt.Print("Pressione Enter para continuar sem reescrita (ou Ctrl+C para sair)...")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}

	publicados := 0
	inicio := time.Now()

	defer func() {
		if r := recover(); r != nil {
			log.Error("panic durante a publicacao", "erro", r, "publicados", publicados)
			panic(r)
		}
	}()

	for i, item := range itens {
		log.Info("processando item", "indice", i+1, "total", len(itens), "titulo", item.Title)

		if reescritor != nil {
			nova, err := reescritor.Rewrite(ctx, item.Description)
			if err != nil {
				log.Warn("falha ao reescrever descricao, usando original", "erro", err)
			} else {
				item.Description = nova
			}
		}

		log.Info("publicando anuncio", "indice", i+1, "titulo", item.Title)
		utils.CreateListing(page, item, i)
		publicados++

		if i < len(itens)-1 {
			espera := time.Duration(5+rand.Intn(11)) * time.Second
			log.Info("aguardando antes do proximo item", "segundos", int(espera.Seconds()))
			time.Sleep(espera)
		}
	}

	minutos := time.Since(inicio).Minutes()
	log.Info("concluido", "anuncios", publicados, "minutos", fmt.Sprintf("%.1f", minutos))
}
