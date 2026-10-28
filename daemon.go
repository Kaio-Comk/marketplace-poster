package main

import (
	"context"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-rod/rod"

	"marketplace-poster/utils"
)

// Arquivo onde o estado persiste entre execucoes (sobrevive a reinicios).
const estadoArquivo = "estado.json"

// Pausa padrao quando algo externo falha (planilha, navegador, sem anuncios).
const pausaErro = 10 * time.Minute

// runDaemon e o loop principal: publica um anuncio por vez, respeitando
// janela de horario, limite diario, intervalo aleatorio e backoff de falhas.
func runDaemon(planilhaPath, chromePath string, minMin, janelaIni, janelaFim int) {
	// Ctrl+C ou SIGTERM cancelam o contexto e encerram o loop com limpeza.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7))

	var browser *rod.Browser
	var page *rod.Page
	// Closure para fechar o navegador que estiver aberto no momento da saida,
	// e nao o valor que a variavel tinha quando o defer foi registrado.
	defer func() {
		if browser != nil {
			fecharNavegador(browser)
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		// O estado e relido a cada volta: o arquivo e a fonte da verdade.
		est := CarregarEstado(estadoArquivo)

		// Respeita o que for mais tardio: proximo horario ou pausa de backoff.
		espera := est.ProximoEm
		if est.PausadoAte.After(espera) {
			espera = est.PausadoAte
		}
		if agora := time.Now(); agora.Before(espera) {
			if !dormirFatias(ctx, time.Until(espera)) {
				return
			}
			continue
		}

		// Fora da janela permitida, dorme ate a abertura da proxima.
		agora := time.Now()
		if !dentroJanela(agora, janelaIni, janelaFim) {
			if !dormirFatias(ctx, time.Until(inicioComJitter(agora, janelaIni))) {
				return
			}
			continue
		}

		// Envia ao disco o que ficou pendente de execucoes anteriores antes de ler.
		flushPendentes(planilhaPath)

		p, err := utils.LerPlanilha(planilhaPath)
		if err != nil {
			utils.Logger.Warn("falha ao ler planilha", "erro", err)
			if !dormirFatias(ctx, pausaErro) {
				return
			}
			continue
		}
		// Soma os pendentes para o limite diario nao ser estourado por atraso de gravacao.
		p.Publicados = append(p.Publicados, lerPendentes()...)

		if p.LimiteDia-publicadosHoje(p) <= 0 {
			utils.Logger.Info("limite diario atingido, aguardando proximo dia")
			if !dormirFatias(ctx, time.Until(inicioComJitter(agora, janelaIni))) {
				return
			}
			continue
		}

		planos := utils.ProximosAnuncios(p, 1, rng)
		if len(planos) == 0 {
			utils.Logger.Warn("nenhum anuncio disponivel para publicar")
			if !dormirFatias(ctx, pausaErro) {
				return
			}
			continue
		}

		// O navegador so abre quando ha o que publicar e e reaberto apos falha.
		if page == nil {
			browser, page = abrirNavegador(chromePath)
			if page == nil {
				// Quase sempre e login expirado (ninguem para entrar): backoff em vez de
				// reabrir o Chrome a cada poucos minutos.
				browser = nil
				est.FalhasSeguidas++
				est.PausadoAte = time.Now().Add(backoffFalhas(est.FalhasSeguidas))
				est.Motivo = "nao abriu o navegador ou login do Facebook pendente"
				if err := SalvarEstado(estadoArquivo, est); err != nil {
					utils.Logger.Error("falha ao salvar estado", "erro", err)
				}
				utils.Logger.Warn("nao foi possivel abrir o navegador",
					"falhas", est.FalhasSeguidas, "pausadoAte", est.PausadoAte)
				continue
			}
		}

		ok := publishOne(page, montarItem(p, planos[0]))
		if !ok {
			// Sessao possivelmente comprometida: fecha tudo e aplica backoff.
			fecharNavegador(browser)
			browser, page = nil, nil

			est.FalhasSeguidas++
			est.PausadoAte = time.Now().Add(backoffFalhas(est.FalhasSeguidas))
			est.Motivo = "publicacao recusada"
			if err := SalvarEstado(estadoArquivo, est); err != nil {
				utils.Logger.Error("falha ao salvar estado", "erro", err)
			}
			utils.Logger.Warn("publicacao recusada, aplicando backoff",
				"falhas", est.FalhasSeguidas, "pausadoAte", est.PausadoAte)
			continue
		}

		// Horarios irregulares: os anuncios que faltam hoje se espalham ate o fim do expediente.
		agora = time.Now()
		restantesDepois := p.LimiteDia - publicadosHoje(p) - 1
		intervalo := intervaloAleatorio(agora, janelaIni, janelaFim, restantesDepois, minMin)
		est.FalhasSeguidas = 0
		est.Motivo = ""
		est.UltimoPublicado = agora
		est.ProximoEm = agora.Add(intervalo)

		// Salva ANTES de registrar: buscar o link leva segundos, e um corte de energia
		// nesse intervalo faria o anuncio sair de novo depois do reinicio.
		if err := SalvarEstado(estadoArquivo, est); err != nil {
			utils.Logger.Error("falha ao salvar estado", "erro", err)
		}
		registrarPublicado(page, planilhaPath, planos[0])
		utils.Logger.Info("anuncio publicado",
			"proximoEm", est.ProximoEm, "intervaloMin", int(intervalo.Minutes()))
	}
}
