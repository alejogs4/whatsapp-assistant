package gemini_test

import (
	"context"
	"testing"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/llm/gemini"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

func TestGeminiRoutingPorts_ClassifyAndSynthesize(t *testing.T) {
	tr := &fakeTransport{
		jsonResponses: []string{
			`{"matched_paths": ["p-rdd", "unknown-id"], "intent": "PATH_BOUND"}`,
			`{"answer": "A Coordinator delegates work to collaborators."}`,
			`{"proposed_path": "Design Patterns", "reasoning": "Builds directly on RDD stereotypes."}`,
		},
	}
	adapter, err := gemini.NewAdapterWithTransport(gemini.Config{}, tr)
	if err != nil {
		t.Fatalf("NewAdapterWithTransport: %v", err)
	}

	aff, err := adapter.ClassifyPathAffinity(context.Background(), "What is a Coordinator?", []touchpoint.PathSummary{
		{ID: "p-rdd", Title: "RDD"},
	})
	if err != nil {
		t.Fatalf("ClassifyPathAffinity: %v", err)
	}
	if len(aff.MatchedPaths) != 1 || aff.MatchedPaths[0] != "p-rdd" || aff.Intent != touchpoint.IntentPathBound {
		t.Fatalf("unexpected affinity: %+v", aff)
	}

	ans, err := adapter.AnswerGrounded(context.Background(), "What is a Coordinator?", touchpoint.PathCorpusContext{
		PathID:    "p-rdd",
		PathTitle: "RDD",
		Chunks:    []string{"Coordinators delegate work."},
	})
	if err != nil || ans == "" {
		t.Fatalf("AnswerGrounded failed: ans=%q err=%v", ans, err)
	}

	rec, err := adapter.SynthesizeMilestone(context.Background(), touchpoint.MilestoneRequest{
		UserID:            "u-1",
		CompletedPathName: "RDD",
	})
	if err != nil || rec.ProposedPath != "Design Patterns" {
		t.Fatalf("SynthesizeMilestone failed: rec=%+v err=%v", rec, err)
	}
}
