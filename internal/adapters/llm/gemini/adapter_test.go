package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/llm/gemini"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

type fakeTransport struct {
	jsonResponses []string
	jsonCalls     int
	promptsSeen   []string
	embedCalls    int
}

func (f *fakeTransport) GenerateJSON(_ context.Context, _ string, prompt string) (string, error) {
	f.promptsSeen = append(f.promptsSeen, prompt)
	idx := f.jsonCalls
	f.jsonCalls++
	if idx >= len(f.jsonResponses) {
		return f.jsonResponses[len(f.jsonResponses)-1], nil
	}
	return f.jsonResponses[idx], nil
}

func (f *fakeTransport) EmbedBatch(_ context.Context, _ string, texts []string) ([][]float32, error) {
	f.embedCalls++
	out := make([][]float32, len(texts))
	for i, txt := range texts {
		out[i] = []float32{float32(len(txt)), float32(i)}
	}
	return out, nil
}

func TestValidationGuards(t *testing.T) {
	if err := gemini.ValidateDrillRequest(touchpoint.DrillRequest{}); !errors.Is(err, gemini.ErrEmptyCorpus) {
		t.Fatalf("expected ErrEmptyCorpus, got %v", err)
	}

	tooLong := strings.Repeat("word ", 215)
	if err := gemini.ValidateDrillResponse(domain.TouchpointTypeMorningCatalyst, &touchpoint.DrillResponse{PromptText: tooLong}); !errors.Is(err, gemini.ErrPromptTooLong) {
		t.Fatalf("expected ErrPromptTooLong for 215-word morning drill, got %v", err)
	}

	eveningOK := strings.Repeat("word ", 45)
	if err := gemini.ValidateDrillResponse(domain.TouchpointTypeEveningStreakSaver, &touchpoint.DrillResponse{PromptText: eveningOK}); err != nil {
		t.Fatalf("expected 45-word evening drill to pass, got %v", err)
	}

	if err := gemini.ValidateEvaluateResponse(&touchpoint.EvaluateResponse{Score: 1.25, Feedback: "ok"}); !errors.Is(err, gemini.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for 1.25, got %v", err)
	}

	validEval := &touchpoint.EvaluateResponse{Score: 0.75, Feedback: "Solid recall."}
	if err := gemini.ValidateEvaluateResponse(validEval); err != nil {
		t.Fatalf("expected valid eval to pass, got %v", err)
	}
	if !validEval.Passed {
		t.Fatalf("expected Passed=true for score 0.75")
	}
}

func TestNewAdapter_MissingAPIKeyFailsFast(t *testing.T) {
	_, err := gemini.NewAdapter(context.Background(), gemini.Config{})
	if !errors.Is(err, gemini.ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestEmbedChunks_BatchingAndOrder(t *testing.T) {
	tr := &fakeTransport{}
	adapter, err := gemini.NewAdapterWithTransport(gemini.Config{EmbedBatchSize: 50}, tr)
	if err != nil {
		t.Fatalf("NewAdapterWithTransport: %v", err)
	}

	empty, err := adapter.EmbedChunks(context.Background(), nil)
	if err != nil || len(empty) != 0 || tr.embedCalls != 0 {
		t.Fatalf("expected empty embeddings with 0 transport calls, got len=%d calls=%d err=%v", len(empty), tr.embedCalls, err)
	}

	chunks := make([]string, 125)
	for i := range chunks {
		chunks[i] = strings.Repeat("a", (i%10)+1)
	}
	vecs, err := adapter.EmbedChunks(context.Background(), chunks)
	if err != nil {
		t.Fatalf("EmbedChunks: %v", err)
	}
	if tr.embedCalls != 3 {
		t.Fatalf("expected 3 batch calls for 125 chunks (batch=50), got %d", tr.embedCalls)
	}
	if len(vecs) != 125 {
		t.Fatalf("expected 125 embeddings, got %d", len(vecs))
	}
	if vecs[0][0] != float32(len(chunks[0])) || vecs[124][0] != float32(len(chunks[124])) {
		t.Fatalf("embedding order mismatch")
	}
}

func TestGenerateMicroDrill_RetriesOversizedResponseOnce(t *testing.T) {
	longJSON, _ := json.Marshal(map[string]string{"prompt_text": strings.Repeat("word ", 250)})
	shortJSON, _ := json.Marshal(map[string]string{"prompt_text": strings.Repeat("word ", 80)})

	tr := &fakeTransport{jsonResponses: []string{string(longJSON), string(shortJSON)}}
	adapter, err := gemini.NewAdapterWithTransport(gemini.Config{MaxRetries: 1}, tr)
	if err != nil {
		t.Fatalf("NewAdapterWithTransport: %v", err)
	}

	resp, err := adapter.GenerateMicroDrill(context.Background(), touchpoint.DrillRequest{
		ConceptKey: "stereotypes",
		Corpus:     []string{"An Information Holder knows and provides information."},
		Type:       domain.TouchpointTypeMorningCatalyst,
	})
	if err != nil {
		t.Fatalf("GenerateMicroDrill: %v", err)
	}
	if tr.jsonCalls != 2 {
		t.Fatalf("expected 2 attempts, got %d", tr.jsonCalls)
	}
	if len(strings.Fields(resp.PromptText)) != 80 {
		t.Fatalf("expected 80-word prompt, got %d words", len(strings.Fields(resp.PromptText)))
	}
}

func TestEvaluateAnswer_MalformedJSONFailsAfterRetriesAndValidPasses(t *testing.T) {
	badTr := &fakeTransport{jsonResponses: []string{"{not-json"}}
	badAdapter, _ := gemini.NewAdapterWithTransport(gemini.Config{MaxRetries: 1}, badTr)

	_, err := badAdapter.EvaluateAnswer(context.Background(), touchpoint.EvaluateRequest{
		Prompt:     "Define Information Holder",
		UserAnswer: "Stores state",
		Corpus:     []string{"Information Holder knows and provides information."},
	})
	if !errors.Is(err, gemini.ErrMalformedJSON) {
		t.Fatalf("expected ErrMalformedJSON, got %v", err)
	}
	if badTr.jsonCalls != 2 {
		t.Fatalf("expected 2 attempts (1 initial + 1 retry), got %d", badTr.jsonCalls)
	}

	goodTr := &fakeTransport{jsonResponses: []string{`{"score": 0.85, "feedback": "Accurate and concise.", "passed": true}`}}
	goodAdapter, _ := gemini.NewAdapterWithTransport(gemini.Config{MaxRetries: 1}, goodTr)
	evalResp, err := goodAdapter.EvaluateAnswer(context.Background(), touchpoint.EvaluateRequest{
		Prompt:     "Define Information Holder",
		UserAnswer: "Stores and provides state",
		Corpus:     []string{"Information Holder knows and provides information."},
	})
	if err != nil {
		t.Fatalf("EvaluateAnswer valid: %v", err)
	}
	if evalResp.Score != 0.85 || !evalResp.Passed || evalResp.Feedback == "" {
		t.Fatalf("unexpected EvaluateResponse: %+v", evalResp)
	}
}
