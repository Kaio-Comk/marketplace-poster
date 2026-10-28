package main

import (
	"testing"
	"time"
)

func TestInicioComJitter(t *testing.T) {
	base := time.Date(2026, 10, 27, 19, 0, 0, 0, time.Local)
	abre := time.Date(2026, 10, 28, 8, 0, 0, 0, time.Local)
	vistos := map[time.Duration]bool{}
	for range 200 {
		d := inicioComJitter(base, 8).Sub(abre)
		if d < 0 || d >= jitterMaxInicio {
			t.Fatalf("fora de [0, %v): %v", jitterMaxInicio, d)
		}
		vistos[d.Round(time.Minute)] = true
	}
	if len(vistos) < 20 {
		t.Errorf("horários pouco variados: %d valores distintos em 200 sorteios", len(vistos))
	}
}

func TestIntervaloAleatorio(t *testing.T) {
	agora := time.Date(2026, 10, 27, 9, 0, 0, 0, time.Local) // faltam 9 h até as 18
	piso := 25 * time.Minute
	teto := 9 * time.Hour / 3 * 3 / 2 // 3 anúncios restantes: 1,5x a média de 3 h
	for range 500 {
		d := intervaloAleatorio(agora, 8, 18, 3, 25)
		if d < piso || d > teto {
			t.Fatalf("fora de [%v, %v]: %v", piso, teto, d)
		}
	}
	if got := intervaloAleatorio(agora, 8, 18, 0, 25); got != piso {
		t.Errorf("sem anúncios restantes deve dar o piso: %v", got)
	}
	tarde := time.Date(2026, 10, 27, 17, 59, 0, 0, time.Local)
	if got := intervaloAleatorio(tarde, 8, 18, 5, 25); got != piso {
		t.Errorf("quase no fim do expediente deve dar o piso: %v", got)
	}
	// janela de 24 h conta até a meia-noite e nunca passa de 1,5x a média
	if got := intervaloAleatorio(agora, 8, 8, 1, 25); got > 15*time.Hour*3/2 {
		t.Errorf("janela 24h: %v", got)
	}
}
