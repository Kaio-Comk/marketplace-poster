package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDentroJanela(t *testing.T) {
	h := func(hora int) time.Time { return time.Date(2026, 10, 27, hora, 30, 0, 0, time.Local) }
	cases := []struct {
		hora, ini, fim int
		want           bool
	}{
		{7, 8, 22, false}, {8, 8, 22, true}, {21, 8, 22, true}, {22, 8, 22, false},
		{3, 8, 8, true},   // fim <= ini: janela de 24 h
		{23, 20, 6, true}, // atravessa a meia-noite não é suportado: tratado como 24 h
	}
	for _, c := range cases {
		if got := dentroJanela(h(c.hora), c.ini, c.fim); got != c.want {
			t.Errorf("hora=%d janela=[%d,%d) got %v want %v", c.hora, c.ini, c.fim, got, c.want)
		}
	}
}

func TestProximoInicio(t *testing.T) {
	antes := time.Date(2026, 10, 27, 5, 0, 0, 0, time.Local)
	depois := time.Date(2026, 10, 27, 15, 0, 0, 0, time.Local)
	if got := proximoInicio(antes, 8); !got.Equal(time.Date(2026, 10, 27, 8, 0, 0, 0, time.Local)) {
		t.Errorf("antes da abertura: %v", got)
	}
	if got := proximoInicio(depois, 8); !got.Equal(time.Date(2026, 10, 28, 8, 0, 0, 0, time.Local)) {
		t.Errorf("depois da abertura: %v", got)
	}
	exato := time.Date(2026, 10, 27, 8, 0, 0, 0, time.Local)
	if got := proximoInicio(exato, 8); !got.After(exato) {
		t.Errorf("deve ser estritamente depois: %v", got)
	}
}

func TestBackoffFalhas(t *testing.T) {
	want := map[int]time.Duration{1: 30 * time.Minute, 2: time.Hour, 3: 2 * time.Hour, 4: 4 * time.Hour, 5: 12 * time.Hour, 9: 12 * time.Hour}
	for n, d := range want {
		if got := backoffFalhas(n); got != d {
			t.Errorf("n=%d got %v want %v", n, got, d)
		}
	}
}

func TestEstadoSobreviveReinicio(t *testing.T) {
	path := filepath.Join(t.TempDir(), "estado.json")
	if e := CarregarEstado(path); !e.ProximoEm.IsZero() {
		t.Fatalf("arquivo ausente deve dar estado zero: %+v", e)
	}
	in := Estado{ProximoEm: time.Now().Add(30 * time.Minute).Round(time.Second), FalhasSeguidas: 2, Motivo: "x"}
	if err := SalvarEstado(path, in); err != nil {
		t.Fatal(err)
	}
	out := CarregarEstado(path)
	if !out.ProximoEm.Equal(in.ProximoEm) || out.FalhasSeguidas != 2 {
		t.Errorf("estado não voltou igual: %+v", out)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("sobrou arquivo .tmp")
	}
	// arquivo corrompido não pode derrubar o serviço
	_ = os.WriteFile(path, []byte("{lixo"), 0o644)
	if e := CarregarEstado(path); !e.ProximoEm.IsZero() {
		t.Errorf("corrompido deve dar estado zero: %+v", e)
	}
}
