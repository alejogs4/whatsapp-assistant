package gemini

import (
	"fmt"
	"strings"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

func hasNonEmptyChunk(corpus []string) bool {
	for _, c := range corpus {
		if strings.TrimSpace(c) != "" {
			return true
		}
	}
	return false
}

// ValidateDrillRequest enforces grounded corpus presence before calling the LLM.
func ValidateDrillRequest(req touchpoint.DrillRequest) error {
	if !hasNonEmptyChunk(req.Corpus) {
		return ErrEmptyCorpus
	}
	return nil
}

// ValidateDrillResponse enforces non-empty output and bite-sized word limits per TouchpointType.
func ValidateDrillResponse(tpType domain.TouchpointType, resp *touchpoint.DrillResponse) error {
	if resp == nil || strings.TrimSpace(resp.PromptText) == "" {
		return ErrEmptyOutput
	}
	words := len(strings.Fields(resp.PromptText))
	maxWords := MaxMorningDrillWords
	if tpType == domain.TouchpointTypeEveningStreakSaver {
		maxWords = MaxEveningDrillWords
	}
	if words > maxWords {
		return fmt.Errorf("%w: got %d words, max %d", ErrPromptTooLong, words, maxWords)
	}
	return nil
}

// ValidateEvaluateRequest enforces grounded corpus and non-empty answer/prompt.
func ValidateEvaluateRequest(req touchpoint.EvaluateRequest) error {
	if !hasNonEmptyChunk(req.Corpus) {
		return ErrEmptyCorpus
	}
	if strings.TrimSpace(req.Prompt) == "" || strings.TrimSpace(req.UserAnswer) == "" {
		return ErrEmptyOutput
	}
	return nil
}

// ValidateEvaluateResponse enforces score bounds [0.0, 1.0], non-empty feedback, and Passed consistency.
func ValidateEvaluateResponse(resp *touchpoint.EvaluateResponse) error {
	if resp == nil || strings.TrimSpace(resp.Feedback) == "" {
		return ErrEmptyOutput
	}
	if resp.Score < 0.0 || resp.Score > 1.0 {
		return fmt.Errorf("%w: got %f", ErrInvalidScore, resp.Score)
	}
	resp.Passed = resp.Score >= PassingScoreFloor
	return nil
}
