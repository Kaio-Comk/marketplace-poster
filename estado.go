package main

import (
	"encoding/json"
	"os"
	"time"
)

// Estado guarda o que precisa sobreviver a um reinício do processo/PC.
// ProximoEm e PausadoAte são persistidos para que a pausa entre anúncios
// e o backoff de falhas não se percam quando o serviço cai.
type Estado struct {
	ProximoEm       time.Time `json:"proximo_em"`
	UltimoPublicado time.Time `json:"ultimo_publicado"`
	FalhasSeguidas  int       `json:"falhas_seguidas"`
	PausadoAte      time.Time `json:"pausado_ate"`
	Motivo          string    `json:"motivo"`
}

// CarregarEstado lê o estado do disco. Arquivo ausente ou corrompido
// devolve Estado zero sem panic: o daemon deve sempre poder iniciar.
func CarregarEstado(path string) Estado {
	var e Estado
	b, err := os.ReadFile(path)
	if err != nil {
		return Estado{}
	}
	if err := json.Unmarshal(b, &e); err != nil {
		// Corrompido: melhor recomeçar do zero do que travar o serviço.
		return Estado{}
	}
	return e
}

// SalvarEstado grava de forma atômica: escreve em path+".tmp" no mesmo
// diretório, faz fsync e renomeia sobre o destino. os.Rename substitui
// o arquivo existente também no Windows, evitando estado parcialmente
// escrito se o processo morrer no meio.
func SalvarEstado(path string, e Estado) error {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
