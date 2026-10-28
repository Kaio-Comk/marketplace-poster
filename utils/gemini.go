package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

const geminiRewriteModel = "gemini-2.5-flash"

// geminiDescriptionRewriter encapsula o cliente do Gemini usado para
// reescrever descrições de anúncios do Marketplace.
type geminiDescriptionRewriter struct {
	client *genai.Client
	model  string
	phone  string
}

// NewGeminiDescriptionRewriter monta o reescritor a partir da configuração.
// Retorna erro explícito quando a chave da API não foi informada.
func NewGeminiDescriptionRewriter(ctx context.Context, config Config) (*geminiDescriptionRewriter, error) {
	if strings.TrimSpace(config.GEMINI_API_KEY) == "" {
		return nil, fmt.Errorf("gemini: GEMINI_API_KEY não configurada")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  strings.TrimSpace(config.GEMINI_API_KEY),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: falha ao criar cliente: %w", err)
	}

	return &geminiDescriptionRewriter{
		client: client,
		model:  geminiRewriteModel,
		phone:  strings.TrimSpace(config.PHONE_NUMBER),
	}, nil
}

// Rewrite reescreve a descrição mantendo o idioma e os fatos originais.
func (r *geminiDescriptionRewriter) Rewrite(ctx context.Context, description string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	prompt := r.buildPrompt(description)

	result, err := r.client.Models.GenerateContent(ctx, r.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini: falha ao gerar conteúdo: %w", err)
	}

	text := strings.TrimSpace(result.Text())
	if text == "" {
		return "", fmt.Errorf("gemini: resposta vazia do modelo")
	}

	return text, nil
}

// buildPrompt monta as instruções enviadas ao modelo, incluindo ou não a
// linha de contato conforme o telefone esteja configurado.
func (r *geminiDescriptionRewriter) buildPrompt(description string) string {
	var b strings.Builder

	b.WriteString("Você é um vendedor experiente do Marketplace. Reescreva a descrição de anúncio abaixo.\n")
	b.WriteString("Regras obrigatórias:\n")
	b.WriteString("- Escreva no MESMO idioma do texto original.\n")
	b.WriteString("- Mantenha todos os fatos apresentados; não invente nem omita informações.\n")
	b.WriteString("- Use texto simples, sem markdown, sem aspas e sem listas.\n")
	b.WriteString("- Adote um tom natural de vendedor, claro e direto.\n")

	if r.phone != "" {
		b.WriteString("- Termine o texto com uma linha de contato exibindo o telefone ")
		b.WriteString(phoneToEmoji(r.phone))
		b.WriteString(" (cada dígito como emoji).\n")
	} else {
		b.WriteString("- Não inclua nenhum telefone, número de contato ou dado semelhante no texto.\n")
	}

	b.WriteString("\nDescrição original:\n")
	b.WriteString(description)

	return b.String()
}

// phoneToEmoji converte apenas os dígitos de phone em emojis keycap,
// descartando qualquer outro caractere.
func phoneToEmoji(phone string) string {
	keycaps := map[rune]string{
		'0': "0️⃣",
		'1': "1️⃣",
		'2': "2️⃣",
		'3': "3️⃣",
		'4': "4️⃣",
		'5': "5️⃣",
		'6': "6️⃣",
		'7': "7️⃣",
		'8': "8️⃣",
		'9': "9️⃣",
	}

	var b strings.Builder
	for _, ch := range phone {
		if emoji, ok := keycaps[ch]; ok {
			b.WriteString(emoji)
		}
	}
	return b.String()
}
