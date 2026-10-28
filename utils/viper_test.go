package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigLeDotEnv(t *testing.T) {
	dir := t.TempDir()
	conteudo := "PLANILHA_PATH=C:/qualquer/Planilha.xlsx\nDEFAULT_CITY=Natal, RN\nPHONE_NUMBER=\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PLANILHA_PATH != "C:/qualquer/Planilha.xlsx" || cfg.DEFAULT_CITY != "Natal, RN" {
		t.Errorf(".env não foi lido: %+v", cfg)
	}
}

func TestLoadConfigSemArquivoNaoEhErro(t *testing.T) {
	t.Setenv("ITEMS_PATH", "D:/itens")
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("arquivo ausente não pode ser erro: %v", err)
	}
	if cfg.ITEMS_PATH != "D:/itens" {
		t.Errorf("variável de ambiente ignorada: %+v", cfg)
	}
}

func TestLoadConfigAmbienteSobrescreveArquivo(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("DEFAULT_CITY=Natal\n"), 0o644)
	t.Setenv("DEFAULT_CITY", "Fortaleza")
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DEFAULT_CITY != "Fortaleza" {
		t.Errorf("ambiente deveria vencer o arquivo: %+v", cfg)
	}
}
