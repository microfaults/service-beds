package llm

import "context"

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) GenerateContent(ctx context.Context, prompt string, images []string) (string, error) {
	// Stubbed response
	return "This looks like a modern living room with a beige sofa and a wooden coffee table. I recommend adding a geometric rug and some potted plants to enhance the natural vibe.", nil
}

func (m *MockClient) EmbedContent(ctx context.Context, text string) ([]float32, error) {
	// Return a zero vector of dimension 768 to match typical embedding size
	return make([]float32, 768), nil
}
