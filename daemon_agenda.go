package main

import (
	"math/rand/v2"
	"time"
)

// jitterMaxInicio e o atraso aleatorio maximo na abertura da janela: o primeiro
// anuncio do dia nunca sai sempre no mesmo horario.
const jitterMaxInicio = 90 * time.Minute

// inicioComJitter e o proximo comeco de expediente somado a um atraso aleatorio.
func inicioComJitter(t time.Time, ini int) time.Time {
	return proximoInicio(t, ini).Add(time.Duration(rand.Int64N(int64(jitterMaxInicio))))
}

// intervaloAleatorio espalha os anuncios que faltam ate o fim do expediente de forma
// irregular: sorteia entre o piso (minMin) e 1,5x a media do que sobra do dia.
// Janela de 24 h (fim <= ini) conta ate a meia-noite.
func intervaloAleatorio(agora time.Time, ini, fim, restantes, minMin int) time.Duration {
	piso := time.Duration(minMin) * time.Minute
	if restantes <= 0 {
		return piso
	}
	y, m, d := agora.Date()
	limite := time.Date(y, m, d, fim, 0, 0, 0, agora.Location())
	if fim <= ini {
		limite = time.Date(y, m, d+1, 0, 0, 0, 0, agora.Location())
	}
	if !limite.After(agora) {
		return piso
	}
	teto := limite.Sub(agora) / time.Duration(restantes) * 3 / 2
	if teto <= piso {
		return piso
	}
	return piso + time.Duration(rand.Int64N(int64(teto-piso)+1))
}
