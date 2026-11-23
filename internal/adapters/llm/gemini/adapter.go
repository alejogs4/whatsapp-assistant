package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

var (
	_ touchpoint.LLMPort     = (*Adapter)(nil)
	_ learning.EmbeddingPort = (*Adapter)(nil)
)

// Transport is the package-internal interface wrapping concrete Gemini SDK calls
// so unit and contract tests execute deterministically offline.
type Transport interface {
	GenerateJSON(ctx context.Context, model string, prompt string) (string, error)
	EmbedBatch(ctx context.Context, model string, texts []string) ([][]float32, error)
}

// Adapter implements touchpoint.LLMPort and learning.EmbeddingPort via Gemini.
type Adapter struct {
	cfg       Config
	transport Transport
}

// NewAdapterWithTransport builds an Adapter with an injected Transport (used in tests and custom wiring).
func NewAdapterWithTransport(cfg Config, transport Transport) (*Adapter, error) {
	if transport == nil {
		return nil, fmt.Errorf("gemini: transport must not be nil")
	}
	return &Adapter{
		cfg:       cfg.withDefaults(),
		transport: transport,
	}, nil
}

// EmbedChunks generates vector embeddings in bounded batches preserving exact input order.
func (a *Adapter) EmbedChunks(ctx context.Context, chunks []string) ([][]float32, error) {
	if len(chunks) == 0 {
		return [][]float32{}, nil
	}

	batchSize := a.cfg.EmbedBatchSize
	result := make([][]float32, 0, len(chunks))

	for start := 0; start < len(chunks); start += batchSize {
		end := start + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[start:end]
		vecs, err := a.transport.EmbedBatch(ctx, a.cfg.EmbeddingModel, batch)
		if err != nil {
			return nil, fmt.Errorf("gemini: embed batch [%d:%d]: %w", start, end, err)
		}
		if len(vecs) != len(batch) {
			return nil, fmt.Errorf("%w: batch [%d:%d] expected %d, got %d", ErrEmbeddingCount, start, end, len(batch), len(vecs))
		}
		result = append(result, vecs...)
	}

	return result, nil
}

type drillPayload struct {
	PromptText string `json:"prompt_text"`
}

func (a *Adapter) GenerateMicroDrill(ctx context.Context, req touchpoint.DrillRequest) (*touchpoint.DrillResponse, error) {
	if err := ValidateDrillRequest(req); err != nil {
		return nil, err
	}

	var lastErr error
	var correction string
	attempts := a.cfg.MaxRetries + 1

	for i := 0; i < attempts; i++ {
		prompt := buildMicroDrillPrompt(req, correction)
		raw, err := a.transport.GenerateJSON(ctx, a.cfg.TextModel, prompt)
		if err != nil {
			return nil, fmt.Errorf("gemini: generate micro-drill: %w", err)
		}

		var payload drillPayload
		if err := json.Unmarshal([]byte(stripMarkdownFences(raw)), &payload); err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrMalformedJSON, err)
			correction = "Response was not valid JSON with field prompt_text"
			continue
		}

		resp := &touchpoint.DrillResponse{PromptText: strings.TrimSpace(payload.PromptText)}
		if err := ValidateDrillResponse(req.Type, resp); err != nil {
			lastErr = err
			correction = err.Error()
			continue
		}
		return resp, nil
	}

	return nil, lastErr
}

type evaluatePayload struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
	Passed   bool    `json:"passed"`
}

func (a *Adapter) EvaluateAnswer(ctx context.Context, req touchpoint.EvaluateRequest) (*touchpoint.EvaluateResponse, error) {
	if err := ValidateEvaluateRequest(req); err != nil {
		return nil, err
	}

	var lastErr error
	var correction string
	attempts := a.cfg.MaxRetries + 1

	for i := 0; i < attempts; i++ {
		prompt := buildEvaluatePrompt(req, correction)
		raw, err := a.transport.GenerateJSON(ctx, a.cfg.TextModel, prompt)
		if err != nil {
			return nil, fmt.Errorf("gemini: evaluate answer: %w", err)
		}

		var payload evaluatePayload
		if err := json.Unmarshal([]byte(stripMarkdownFences(raw)), &payload); err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrMalformedJSON, err)
			correction = "Response was not valid JSON with fields score, feedback, passed"
			continue
		}

		resp := &touchpoint.EvaluateResponse{
			Score:    payload.Score,
			Feedback: strings.TrimSpace(payload.Feedback),
			Passed:   payload.Passed,
		}
		if err := ValidateEvaluateResponse(resp); err != nil {
			lastErr = err
			correction = err.Error()
			continue
		}
		return resp, nil
	}

	return nil, lastErr
}

func stripMarkdownFences(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "```") {
		if idx := strings.Index(trimmed, "\n"); idx != -1 {
			trimmed = trimmed[idx+1:]
		}
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}
	return trimmed
}
