package utils

import "time"

// Variante is one photo (Texto = file name), title or description from the spreadsheet.
type Variante struct{ ID, Texto string }

type Cidade struct {
	Nome, UF  string
	Populacao int
	Ativa     bool
}

// Publicado is one row of the "Anúncios publicados" sheet.
type Publicado struct {
	ID, Foto, Titulo, Descricao, Cidade, UF, Link, Status string
	Data                                                  time.Time
}

type Planilha struct {
	Preco                                    float64
	Categoria, Estado, Etiquetas, PastaFotos string
	LimiteDia                                int
	Fotos, Titulos, Descricoes               []Variante
	Cidades                                  []Cidade
	Publicados                               []Publicado
}
