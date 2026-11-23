package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type sdkTransport struct {
	client *genai.Client
}

// NewAdapter constructs a Gemini Adapter backed by the official google.golang.org/genai SDK.
func NewAdapter(ctx context.Context, cfg Config) (*Adapter, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, ErrMissingAPIKey
	}
	cfg = cfg.withDefaults()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: create client: %w", err)
	}

	return NewAdapterWithTransport(cfg, &sdkTransport{client: client})
}

func (s *sdkTransport) GenerateJSON(ctx context.Context, model string, prompt string) (string, error) {
	resp, err := s.client.Models.GenerateContent(
		ctx,
		model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
		},
	)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", ErrEmptyOutput
	}
	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", ErrEmptyOutput
	}
	return text, nil
}

func (s *sdkTransport) EmbedBatch(ctx context.Context, model string, texts []string) ([][]float32, error) {
	contents := make([]*genai.Content, len(texts))
	for i, txt := range texts {
		contents[i] = genai.NewContentFromText(txt, genai.RoleUser)
	}

	resp, err := s.client.Models.EmbedContent(ctx, model, contents, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Embeddings) != len(texts) {
		got := 0
		if resp != nil {
			got = len(resp.Embeddings)
		}
		return nil, fmt.Errorf("%w: expected %d embeddings, got %d", ErrEmbeddingCount, len(texts), got)
	}

	out := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		if emb == nil {
			out[i] = []float32{}
			continue
		}
		out[i] = emb.Values
	}
	return out, nil
}
