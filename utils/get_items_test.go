package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func escreve(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGetItemsProdutoEPastaDeCidade(t *testing.T) {
	raiz := t.TempDir()

	// Produto direto na raiz, com BOM e CRLF como o Bloco de Notas salva.
	escreve(t, filepath.Join(raiz, "A", "description.txt"),
		"\xef\xbb\xbftítulo: Camiseta\r\npreço: 10\r\ncategoria: Roupas\r\nestado: Novo\r\netiquetas: moda, verao\r\ndescrição: Linha um\r\nLinha dois\r\n")
	escreve(t, filepath.Join(raiz, "A", "foto1.JPG"), "x")
	escreve(t, filepath.Join(raiz, "A", "Thumbs.db"), "x")

	// Produto dentro de uma pasta de cidade, sem cidade própria.
	escreve(t, filepath.Join(raiz, "Fortaleza", "B", "description.txt"), "title: Mesa\nprice: 50\ndescription: Mesa boa\n")
	escreve(t, filepath.Join(raiz, "Fortaleza", "B", "m.png"), "x")

	// Produto sem foto e pasta sem description.txt: ignorados.
	escreve(t, filepath.Join(raiz, "C", "description.txt"), "title: Sem foto\nprice: 1\ndescription: x\n")

	itens, err := GetItems(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(itens) != 2 {
		t.Fatalf("esperava 2 itens, veio %d: %+v", len(itens), itens)
	}

	a, b := itens[0], itens[1]
	if a.Title != "CAMISETA" || a.Price != "10" || a.Category != "roupas" || a.Condition != "novo" {
		t.Errorf("campos do item A errados: %+v", a)
	}
	if len(a.Tags) != 2 || a.Tags[0] != "moda" || a.Tags[1] != "verao" {
		t.Errorf("tags erradas: %v", a.Tags)
	}
	if a.Description != "Linha um\nLinha dois" {
		t.Errorf("descrição multilinha errada: %q", a.Description)
	}
	if len(a.Images) != 1 {
		t.Errorf("deveria ter só a foto real (sem Thumbs.db): %v", a.Images)
	}
	if b.Title != "MESA" || b.Location != "Fortaleza" {
		t.Errorf("cidade deveria vir da pasta-pai: %+v", b)
	}
}
