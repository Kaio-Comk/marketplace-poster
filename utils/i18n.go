package utils

import "strings"

// Facebook renders Marketplace in the account language, and the values written in
// description.txt may be in English or Portuguese. Every group below lists names for the
// same thing (English, European and Brazilian Portuguese); any of them selects the option.
var optionAliases = [][]string{
	// condition
	{"new", "novo"},
	{"used - like new", "usado - como novo", "como novo", "like new", "seminovo"},
	{"used - good", "usado - em bom estado", "em bom estado", "bom estado", "good", "bom", "used", "usado"},
	{"used - fair", "usado - aceitavel", "aceitavel", "fair"},

	// category
	{"tools", "ferramentas"},
	{"furniture", "mobiliario", "moveis"},
	{"household", "home goods", "artigos domesticos", "utilidades domesticas"},
	{"garden", "jardim"},
	{"appliances", "eletrodomesticos"},
	{"video games", "videojogos", "videogames", "jogos"},
	{"books, movies & music", "livros, filmes e musicas", "livros, filmes e musica"},
	{"bags & luggage", "malas e bagagem", "bolsas e malas"},
	{"women's clothing & shoes", "vestuario e calcado para mulher", "roupas e calcados femininos"},
	{"men's clothing & shoes", "vestuario e calcado para homem", "roupas e calcados masculinos"},
	{"jewelry & accessories", "joias e acessorios"},
	{"health & beauty", "saude e beleza"},
	{"pet supplies", "produtos para animais de estimacao"},
	{"baby & kids", "bebe e crianca"},
	{"toys & games", "brinquedos e jogos"},
	{"electronics & computers", "electronics", "equipamento eletronico e computadores", "eletronicos e computadores", "eletronicos", "computadores"},
	{"cell phones", "phones", "telemoveis", "celulares", "celular", "telefones"},
	{"bicycles", "bicicletas"},
	{"arts & crafts", "artes e artesanato"},
	{"sports & outdoors", "desportos e ar livre", "esportes e ar livre"},
	{"auto parts", "pecas para automoveis", "pecas de carro", "pecas automotivas"},
	{"musical instruments", "instrumentos musicais"},
	{"antiques & collectibles", "artigos antigos e colecionaveis", "antiguidades e colecionaveis"},
	{"garage sale", "venda de garagem"},
	{"miscellaneous", "diversos"},
	{"vehicles", "veiculos"},
}

// description.txt keys, folded (lowercase, no accents), by canonical name.
var fieldKeyAliases = map[string]string{
	"title":          "title",
	"titulo":         "title",
	"price":          "price",
	"preco":          "price",
	"category":       "category",
	"categoria":      "category",
	"condition":      "condition",
	"condicao":       "condition",
	"estado":         "condition",
	"description":    "description",
	"descricao":      "description",
	"tags":           "tags",
	"etiquetas":      "tags",
	"identificacoes": "tags",
	"city":           "location",
	"cidade":         "location",
	"location":       "location",
	"localizacao":    "location",
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
	"–", "-", "—", "-", "’", "'",
)

// foldText lowercases, drops accents, unifies dashes and collapses spaces.
func foldText(s string) string {
	s = accentReplacer.Replace(strings.ToLower(strings.TrimSpace(s)))
	return strings.Join(strings.Fields(s), " ")
}

// optionCandidates returns every accepted spelling (folded) for a condition or category value.
func optionCandidates(value string) []string {
	folded := foldText(value)
	for _, group := range optionAliases {
		for _, name := range group {
			if folded == name {
				return group
			}
		}
	}
	return []string{folded}
}

func foldedMatchAny(text string, names ...string) bool {
	text = foldText(text)
	for _, name := range names {
		if text == foldText(name) {
			return true
		}
	}
	return false
}

func foldedContainsAny(text string, parts ...string) bool {
	text = foldText(text)
	for _, part := range parts {
		if strings.Contains(text, foldText(part)) {
			return true
		}
	}
	return false
}

// splitField splits "Título: x" into the canonical key ("title") and its value.
func splitField(line string) (key, value string, ok bool) {
	name, rest, found := strings.Cut(line, ":")
	if !found {
		return "", "", false
	}

	key, ok = fieldKeyAliases[foldText(name)]
	if !ok {
		return "", "", false
	}

	return key, strings.TrimSpace(rest), true
}
