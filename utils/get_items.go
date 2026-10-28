package utils

import (
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Item representa um produto coletado a partir da árvore de pastas.
type Item struct {
	Images      []string
	Title       string
	Price       string
	Category    string
	Condition   string
	Description string
	Tags        []string
	Location    string
}

// extensões de imagem aceitas (comparadas em minúsculas)
var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".heic": true,
	".heif": true,
}

// arquivos que nunca devem ser tratados como foto
var ignoredFiles = map[string]bool{
	"thumbs.db":   true,
	"desktop.ini": true,
}

// GetItems varre root e devolve todos os produtos encontrados.
//
// Estrutura esperada:
//   - subpasta com description.txt  -> produto direto
//   - subpasta sem description.txt  -> cidade; cada subpasta dela com
//     description.txt é um produto, herdando o nome da cidade como
//     Location caso o próprio produto não defina uma.
func GetItems(root string) ([]Item, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	// ordena as subpastas de root alfabeticamente
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)

	var items []Item

	for _, name := range dirs {
		dirPath := filepath.Join(root, name)

		if hasDescription(dirPath) {
			// produto direto na raiz
			if it, ok := buildItem(dirPath, ""); ok {
				items = append(items, it)
			}
			continue
		}

		// tratado como cidade: varre suas subpastas
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			Logger.Warn("não foi possível ler a pasta da cidade", "pasta", dirPath, "erro", err)
			continue
		}

		subDirs := make([]string, 0, len(subEntries))
		for _, se := range subEntries {
			if se.IsDir() {
				subDirs = append(subDirs, se.Name())
			}
		}
		sort.Strings(subDirs)

		for _, sub := range subDirs {
			prodPath := filepath.Join(dirPath, sub)
			if !hasDescription(prodPath) {
				Logger.Warn("pasta de produto sem description.txt, ignorando", "pasta", prodPath)
				continue
			}
			if it, ok := buildItem(prodPath, name); ok {
				items = append(items, it)
			}
		}
	}

	return items, nil
}

// hasDescription informa se o diretório contém um description.txt.
func hasDescription(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "description.txt"))
	return err == nil && !info.IsDir()
}

// buildItem monta um Item a partir de uma pasta de produto.
// city é o nome da pasta-cidade (vazio quando o produto está na raiz).
// Devolve ok=false quando o produto deve ser ignorado.
func buildItem(dir, city string) (Item, bool) {
	var it Item

	raw, err := os.ReadFile(filepath.Join(dir, "description.txt"))
	if err != nil {
		Logger.Warn("falha ao ler description.txt, ignorando", "pasta", dir, "erro", err)
		return Item{}, false
	}

	parseDescription(string(raw), &it)

	if it.Location == "" {
		it.Location = city
	}

	it.Images = collectImages(dir)

	if it.Title == "" {
		Logger.Warn("produto sem título, ignorando", "pasta", dir)
		return Item{}, false
	}
	if len(it.Images) == 0 {
		Logger.Warn("produto sem fotos, ignorando", "pasta", dir)
		return Item{}, false
	}

	return it, true
}

// parseDescription interpreta o conteúdo do description.txt e preenche o Item.
func parseDescription(content string, it *Item) {
	// remove BOM UTF-8 eventual
	content = strings.TrimPrefix(content, "\ufeff")

	// normaliza CRLF -> LF
	content = strings.ReplaceAll(content, "\r\n", "\n")

	lines := strings.Split(content, "\n")

	seen := map[string]bool{}
	var descLines []string
	inDesc := false

	for _, line := range lines {
		key, value, ok := splitField(line)

		if ok && !seen[key] {
			// campo novo: encerra a descrição se estávamos nela
			inDesc = false
			seen[key] = true

			switch key {
			case "title":
				it.Title = strings.ToUpper(strings.TrimSpace(value))
			case "price":
				it.Price = strings.TrimSpace(value)
			case "category":
				it.Category = strings.ToLower(strings.TrimSpace(value))
			case "condition":
				it.Condition = strings.ToLower(strings.TrimSpace(value))
			case "description":
				inDesc = true
				if v := strings.TrimSpace(value); v != "" {
					descLines = append(descLines, v)
				}
			case "tags":
				it.Tags = parseTags(value)
			case "location":
				it.Location = strings.TrimSpace(value)
			}
			continue
		}

		if inDesc {
			descLines = append(descLines, line)
		}
	}

	it.Description = strings.TrimSpace(strings.Join(descLines, "\n"))
}

// parseTags divide a string de tags por vírgula, aplicando TrimSpace
// e descartando itens vazios.
func parseTags(value string) []string {
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// collectImages lista as fotos válidas da pasta, embaralha e limita a 6.
func collectImages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		Logger.Warn("falha ao listar fotos", "pasta", dir, "erro", err)
		return nil
	}

	var imgs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if ignoredFiles[strings.ToLower(name)] {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if imageExts[ext] {
			imgs = append(imgs, filepath.Join(dir, name))
		}
	}

	rand.Shuffle(len(imgs), func(i, j int) {
		imgs[i], imgs[j] = imgs[j], imgs[i]
	})

	if len(imgs) > 6 {
		imgs = imgs[:6]
	}
	return imgs
}
