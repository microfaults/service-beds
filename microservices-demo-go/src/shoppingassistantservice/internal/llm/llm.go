package llm

import "context"

type Client interface {
	GenerateContent(ctx context.Context, prompt string, images []string) (string, error)
	EmbedContent(ctx context.Context, text string) ([]float32, error)
}
