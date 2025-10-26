package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/sashabaranov/go-openai"
)

type GroqService struct {
	client *openai.Client
	model  string
}

func NewGroqService() *GroqService {
	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		panic("GROQ_API_KEY is not set; please add it to backend/.env or deployment env")
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = "https://api.groq.com/openai/v1"

	model := strings.TrimSpace(os.Getenv("GROQ_MODEL"))
	if model == "" {
		model = "llama-3.1-70b-versatile"
	}

	return &GroqService{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

func (g *GroqService) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       g.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		return "", fmt.Errorf("groq API error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}
	return resp.Choices[0].Message.Content, nil
}

func BuildChatMessages(history []models.Message, targetLang string) []openai.ChatCompletionMessage {
	msgs := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. Primary role: provide general medical information about symptoms, possible causes, and general advice. Do not provide diagnosis or treatment. Always include appropriate caution. You can also answer language-related questions (translations, grammar, usage, examples) when asked. Respond in " + targetLang + ".",
		},
	}
	for _, m := range history {
		role := openai.ChatMessageRoleUser
		if m.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}
	return msgs
}
