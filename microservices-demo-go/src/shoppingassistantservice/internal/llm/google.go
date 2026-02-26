package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GoogleClient struct {
	client *genai.Client
}

func NewGoogleClient(ctx context.Context, projectID, location string) (*GoogleClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, err
	}
	return &GoogleClient{client: client}, nil
}

func (g *GoogleClient) GenerateContent(ctx context.Context, prompt string, images []string) (string, error) {
	// Not implemented fully yet as handler logic is complex with images, but basic structure:
	// This method needs to handle the images which are passed as base64 or URL.
	// The current handler logic parses images inside the handler.
	// Ideally, the interface should take `[]Part` or similar, but we want to abstract that.
	// For now, let's keep it simple and assume the caller handles part construction or we pass raw data?
	// The interface says `images []string`. Let's assume they are URLs or paths.

	// BUT wait, the current handler does some complex logic including downloading images.
	// To match the interface `GenerateContent(ctx, prompt, images)`, we should move that logic here or keep it in handler?
	// The user wants "preserve the usual flow".
	// The flow is: Handler -> LLM.
	// So Handler should do HTTP/DB stuff, LLM should do LLM stuff.
	// Let's implement basic text generation here.

	// Refactoring note: The current handler logic creates parts.
	// We should probably move the `Part` creation logic here if we want a clean abstraction,
	// OR just pass `[]genai.Part` but that leaks implementation details.
	// Let's stick to `prompt string, images []string` (assuming URLs) and handle download inside if needed,
	// OR better, let the interface take `[]byte` for images?
	// Given the constraints and "preserve flow", let's make the interface flexible.
	// But for `mock`, we just return strings.
	// Let's assume `images` are URLs for now as per current handler.

	return "", fmt.Errorf("not implemented for GoogleClient in this refactor step, logic remains in handler for now or needs migration")
}

func (g *GoogleClient) EmbedContent(ctx context.Context, text string) ([]float32, error) {
	content := genai.NewContentFromText(text, "user")
	resp, err := g.client.Models.EmbedContent(ctx, "text-embedding-004", []*genai.Content{content}, nil)
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return resp.Embeddings[0].Values, nil
}
