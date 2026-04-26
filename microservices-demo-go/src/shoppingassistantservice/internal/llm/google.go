package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
	var parts []*genai.Part
	parts = append(parts, genai.NewPartFromText(prompt))

	for _, imgURL := range images {
		resp, err := http.Get(imgURL)
		if err != nil {
			return "", fmt.Errorf("failed to fetch image: %w", err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read image: %w", err)
		}
		mimeType := resp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		parts = append(parts, genai.NewPartFromBytes(data, mimeType))
	}

	result, err := g.client.Models.GenerateContent(ctx, "gemini-2.0-flash", []*genai.Content{genai.NewContentFromParts(parts, "user")}, nil)
	if err != nil {
		return "", err
	}
	return result.Text(), nil
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
