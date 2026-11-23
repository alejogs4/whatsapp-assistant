package gemini

import (
	"fmt"
	"strings"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

func buildMicroDrillPrompt(req touchpoint.DrillRequest, correctionNote string) string {
	var b strings.Builder
	b.WriteString("You are an adaptive learning coach. Generate ONE bite-sized active-recall drill strictly grounded in the provided corpus.\n")
	b.WriteString("NEVER invent facts outside the provided corpus.\n")
	if req.Type == domain.TouchpointTypeEveningStreakSaver {
		fmt.Fprintf(&b, "Format: 15-second zero-friction recall card (STRICTLY <= %d words).\n", MaxEveningDrillWords)
	} else {
		fmt.Fprintf(&b, "Format: 2-minute focused micro-drill (STRICTLY < %d words).\n", MaxMorningDrillWords+1)
	}
	fmt.Fprintf(&b, "Target concept: %s\n", req.ConceptKey)
	b.WriteString("Respond ONLY with JSON matching {\"prompt_text\": \"...\"}.\n\n")
	b.WriteString("Grounded Corpus:\n")
	for i, chunk := range req.Corpus {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, chunk)
	}
	if correctionNote != "" {
		fmt.Fprintf(&b, "\nPREVIOUS OUTPUT FAILED VALIDATION: %s. Fix this strictly.\n", correctionNote)
	}
	return b.String()
}

func buildEvaluatePrompt(req touchpoint.EvaluateRequest, correctionNote string) string {
	var b strings.Builder
	b.WriteString("You are an adaptive learning evaluator. Grade the learner's answer strictly against the provided grounded corpus.\n")
	b.WriteString("Respond ONLY with JSON matching {\"score\": 0.0-1.0, \"feedback\": \"...\", \"passed\": true|false}.\n")
	fmt.Fprintf(&b, "Passing threshold is score >= %.1f.\n\n", PassingScoreFloor)
	fmt.Fprintf(&b, "Drill Prompt: %s\n", req.Prompt)
	fmt.Fprintf(&b, "Learner Answer: %s\n\n", req.UserAnswer)
	b.WriteString("Grounded Corpus:\n")
	for i, chunk := range req.Corpus {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, chunk)
	}
	if correctionNote != "" {
		fmt.Fprintf(&b, "\nPREVIOUS OUTPUT FAILED VALIDATION: %s. Fix this strictly.\n", correctionNote)
	}
	return b.String()
}
