package utils

import (
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
)

// portaBase é o menor número de porta que aceitamos para o DevTools.
const portaBase = 9222

// portaMax é o maior número de porta que aceitamos para o DevTools.
const portaMax = 65535

// tempoEsperaDevTools é quanto tempo aguardamos o Chrome subir a porta de depuração.
const tempoEsperaDevTools = 30 * time.Second

// intervaloChecagem é o intervalo entre tentativas de conexão na porta.
const intervaloChecagem = 500 * time.Millisecond

// tempoLoginManual é quanto tempo damos ao usuário para fazer login na janela.
const tempoLoginManual = 3 * time.Minute

// portaDeterministica calcula uma porta estável a partir do caminho do diretório atual.
// Usa FNV-32a para que a mesma pasta sempre gere a mesma porta base.
func portaDeterministica(caminho string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(caminho))
	faixa := uint32(portaMax - portaBase + 1)
	return portaBase + int(h.Sum32()%faixa)
}

// portaOcupada informa se já existe algo escutando na porta informada.
func portaOcupada(porta int) bool {
	endereco := fmt.Sprintf("127.0.0.1:%d", porta)
	conn, err := net.DialTimeout("tcp", endereco, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// devToolsResponde verifica se a porta responde como endpoint do DevTools.
func devToolsResponde(porta int) bool {
	endereco := fmt.Sprintf("127.0.0.1:%d", porta)
	conn, err := net.DialTimeout("tcp", endereco, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// escolhePorta tenta reaproveitar a porta determinística se já houver um Chrome
// com DevTools da mesma pasta; caso contrário, procura a primeira porta livre.
func escolhePorta(inicial int) int {
	if devToolsResponde(inicial) {
		return inicial
	}
	for p := inicial; p <= portaMax; p++ {
		if !portaOcupada(p) {
			return p
		}
	}
	panic(fmt.Sprintf("utils.Login: nenhuma porta livre encontrada a partir de %d", inicial))
}

// garantePerfil cria o diretório do perfil do Chrome se ainda não existir.
func garantePerfil(caminho string) {
	if err := os.MkdirAll(caminho, 0o755); err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao criar perfil %q: %v", caminho, err))
	}
}

// montaLauncher configura o launcher do rod com as flags exigidas.
func montaLauncher(chromePath, perfil string, porta int) *launcher.Launcher {
	// Modo "usuário": flags de um Chrome comum, menos marcas de automação que launcher.New().
	l := launcher.NewUserMode()

	// Sem caminho informado, procura o Chrome instalado (o rod não faz isso sozinho aqui).
	if chromePath == "" {
		if achado, ok := launcher.LookPath(); ok {
			chromePath = achado
		}
	}
	if chromePath != "" {
		l = l.Bin(chromePath)
	}

	l = l.
		Set(flags.RemoteDebuggingPort, fmt.Sprintf("%d", porta)).
		Set(flags.UserDataDir, perfil).
		Set(flags.NoSandbox).
		Set("disable-notifications").
		Set("disable-features", "Translate").
		Delete("headless")

	// No Windows o helper Leakless é bloqueado por antivírus; desligamos.
	if runtime.GOOS == "windows" {
		l = l.Leakless(false)
	}

	return l
}

// iniciaChromeProcesso sobe o Chrome manualmente e espera a porta de depuração
// responder, já que no Windows o rod não consegue ler a linha de stderr.
func iniciaChromeProcesso(l *launcher.Launcher, porta int) *exec.Cmd {
	// Get(flags.Bin) já resolve o caminho informado ou localiza o navegador.
	bin := l.Get(flags.Bin)
	if bin == "" {
		panic("utils.Login: não foi possível localizar o Chrome")
	}

	args := l.FormatArgs()
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao iniciar o Chrome: %v", err))
	}

	limite := time.Now().Add(tempoEsperaDevTools)
	for time.Now().Before(limite) {
		if devToolsResponde(porta) {
			return cmd
		}
		time.Sleep(intervaloChecagem)
	}

	_ = cmd.Process.Kill()
	panic(fmt.Sprintf("utils.Login: Chrome não respondeu na porta %d em %s", porta, tempoEsperaDevTools))
}

// desligaEmulacao remove qualquer emulação de dispositivo ativa na página.
func desligaEmulacao(pagina *rod.Page) {
	_ = proto.EmulationClearDeviceMetricsOverride{}.Call(pagina)
}

// estaLogado verifica se existe elemento com aria-label contendo profile/perfil.
func estaLogado(pagina *rod.Page) bool {
	seletor := `[aria-label*="profile" i], [aria-label*="perfil" i]`
	el, err := pagina.Timeout(3 * time.Second).Element(seletor)
	if err != nil || el == nil {
		return false
	}
	return true
}

// esperaLoginManual aguarda o usuário entrar na conta, mostrando o tempo restante.
func esperaLoginManual(pagina *rod.Page) {
	fmt.Println("utils.Login: você tem 3 minutos para fazer login na janela do Chrome.")
	fim := time.Now().Add(tempoLoginManual)

	for time.Now().Before(fim) {
		if estaLogado(pagina) {
			fmt.Println("utils.Login: login detectado.")
			return
		}
		restante := time.Until(fim).Round(time.Second)
		fmt.Printf("utils.Login: aguardando login... tempo restante: %s\r", restante)
		time.Sleep(time.Second)
	}

	fmt.Println()
	panic("utils.Login: tempo esgotado aguardando login do usuário")
}

// Login abre o Chrome com perfil local, conecta o rod e garante o login no Facebook.
func Login(chromePath string) (*rod.Browser, *rod.Page) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("utils.Login: não foi possível obter o diretório atual: %v", err))
	}

	perfil := filepath.Join(cwd, "browser_profile")
	garantePerfil(perfil)

	porta := escolhePorta(portaDeterministica(cwd))

	// Um Chrome desta mesma pasta pode ter ficado aberto de uma execução anterior: reaproveita.
	var cmd *exec.Cmd
	if !devToolsResponde(porta) {
		cmd = iniciaChromeProcesso(montaLauncher(chromePath, perfil, porta), porta)
	}

	// Só encerra o Chrome se fomos nós que o abrimos e algo deu errado depois.
	sucesso := false
	defer func() {
		if !sucesso && cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// O rod precisa da URL do websocket, não do endereço http da porta.
	wsURL, err := launcher.ResolveURL(fmt.Sprintf("127.0.0.1:%d", porta))
	if err != nil {
		panic(fmt.Sprintf("utils.Login: não foi possível resolver o DevTools da porta %d: %v", porta, err))
	}
	navegador := rod.New().ControlURL(wsURL)
	if err := navegador.Connect(); err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao conectar no Chrome: %v", err))
	}

	pagina, err := navegador.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao abrir página: %v", err))
	}

	desligaEmulacao(pagina)

	if err := pagina.Navigate("https://web.facebook.com/"); err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao navegar para o Facebook: %v", err))
	}

	if err := pagina.WaitLoad(); err != nil {
		panic(fmt.Sprintf("utils.Login: falha ao aguardar carregamento: %v", err))
	}

	_ = pagina.SetWindow(&proto.BrowserBounds{WindowState: proto.BrowserWindowStateMaximized})

	if !estaLogado(pagina) {
		esperaLoginManual(pagina)
	}

	sucesso = true
	return navegador, pagina
}

// _ evita import não usado em builds que não exercitam strings diretamente.
var _ = strings.TrimSpace
