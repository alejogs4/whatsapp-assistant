package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

var (
	_ touchpoint.PathAffinityPort = (*Adapter)(nil)
	_ touchpoint.SynthesisPort    = (*Adapter)(nil)
)

type affinityPayload struct {
	MatchedPaths []string `json:"matched_paths"`
	Intent       string   `json:"intent"`
}

func (a *Adapter) ClassifyPathAffinity(
	ctx context.Context,
	message string,
	activePaths []touchpoint.PathSummary,
) (*touchpoint.AffinityResult, error) {
	var b strings.Builder
	b.WriteString("Classify the user's message against their active Learning Paths.\n")
	b.WriteString("Respond ONLY with JSON: {\"matched_paths\": [\"path_id\", ...], \"intent\": \"PATH_BOUND|CURRICULUM|GENERAL\"}.\n")
	b.WriteString("Active Paths:\n")
	validIDs := make(map[shared.PathID]struct{}, len(activePaths))
	for _, p := range activePaths {
		validIDs[p.ID] = struct{}{}
		fmt.Fprintf(&b, "- id=%s title=%q\n", p.ID, p.Title)
	}
	fmt.Fprintf(&b, "\nMessage: %s\n", message)

	raw, err := a.transport.GenerateJSON(ctx, a.cfg.TextModel, b.String())
	if err != nil {
		return nil, fmt.Errorf("gemini: classify path affinity: %w", err)
	}

	var payload affinityPayload
	if err := json.Unmarshal([]byte(stripMarkdownFences(raw)), &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}

	matched := make([]shared.PathID, 0, len(payload.MatchedPaths))
	for _, id := range payload.MatchedPaths {
		pid := shared.PathID(id)
		if _, ok := validIDs[pid]; ok {
			matched = append(matched, pid)
		}
	}

	intent := touchpoint.IntentKind(strings.ToUpper(strings.TrimSpace(payload.Intent)))
	switch intent {
	case touchpoint.IntentPathBound, touchpoint.IntentCurriculum, touchpoint.IntentGeneral:
	default:
		if len(matched) > 0 {
			intent = touchpoint.IntentPathBound
		} else {
			intent = touchpoint.IntentGeneral
		}
	}

	return &touchpoint.AffinityResult{
		MatchedPaths: matched,
		Intent:       intent,
	}, nil
}

type answerPayload struct {
	Answer string `json:"answer"`
}

func (a *Adapter) AnswerGrounded(
	ctx context.Context,
	question string,
	pathCtx touchpoint.PathCorpusContext,
) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Answer the user's question strictly grounded in the Learning Path %q corpus.\n", pathCtx.PathTitle)
	b.WriteString("Respond ONLY with JSON: {\"answer\": \"...\"}.\n\n")
	for i, ch := range pathCtx.Chunks {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, ch)
	}
	fmt.Fprintf(&b, "\nQuestion: %s\n", question)

	return a.generateAnswerJSON(ctx, b.String())
}

func (a *Adapter) SynthesizeCrossPath(
	ctx context.Context,
	question string,
	contexts []touchpoint.PathCorpusContext,
) (string, error) {
	var b strings.Builder
	b.WriteString("Synthesize a unified multi-disciplinary answer connecting these isolated Learning Path corpora.\n")
	b.WriteString("Respond ONLY with JSON: {\"answer\": \"...\"}.\n\n")
	for _, pc := range contexts {
		fmt.Fprintf(&b, "=== Path: %s ===\n", pc.PathTitle)
		for i, ch := range pc.Chunks {
			fmt.Fprintf(&b, "[%d] %s\n", i+1, ch)
		}
	}
	fmt.Fprintf(&b, "\nQuestion: %s\n", question)

	return a.generateAnswerJSON(ctx, b.String())
}

type milestonePayload struct {
	ProposedPath string `json:"proposed_path"`
	Reasoning    string `json:"reasoning"`
}

func (a *Adapter) SynthesizeMilestone(
	ctx context.Context,
	req touchpoint.MilestoneRequest,
) (*touchpoint.MilestoneRecommendation, error) {
	var b strings.Builder
	b.WriteString("You are a Macro-Loop technical mentor. Recommend the next logical Learning Path milestone.\n")
	b.WriteString("Respond ONLY with JSON: {\"proposed_path\": \"...\", \"reasoning\": \"...\"}.\n")
	if req.CompletedPathName != "" {
		fmt.Fprintf(&b, "Completed Path: %s\n", req.CompletedPathName)
	}
	if len(req.ActivePathNames) > 0 {
		fmt.Fprintf(&b, "Active Paths: %s\n", strings.Join(req.ActivePathNames, ", "))
	}
	if req.UserQuestion != "" {
		fmt.Fprintf(&b, "User Question: %s\n", req.UserQuestion)
	}

	raw, err := a.transport.GenerateJSON(ctx, a.cfg.TextModel, b.String())
	if err != nil {
		return nil, fmt.Errorf("gemini: synthesize milestone: %w", err)
	}
	var payload milestonePayload
	if err := json.Unmarshal([]byte(stripMarkdownFences(raw)), &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if strings.TrimSpace(payload.ProposedPath) == "" || strings.TrimSpace(payload.Reasoning) == "" {
		return nil, ErrEmptyOutput
	}
	return &touchpoint.MilestoneRecommendation{
		ProposedPath: strings.TrimSpace(payload.ProposedPath),
		Reasoning:    strings.TrimSpace(payload.Reasoning),
	}, nil
}

func (a *Adapter) AnswerGeneral(ctx context.Context, question string) (string, error) {
	prompt := fmt.Sprintf("Answer concisely and helpfully. Respond ONLY with JSON: {\"answer\": \"...\"}.\n\nQuestion: %s\n", question)
	return a.generateAnswerJSON(ctx, prompt)
}

func (a *Adapter) generateAnswerJSON(ctx context.Context, prompt string) (string, error) {
	raw, err := a.transport.GenerateJSON(ctx, a.cfg.TextModel, prompt)
	if err != nil {
		return "", fmt.Errorf("gemini: generate answer: %w", err)
	}
	var payload answerPayload
	if err := json.Unmarshal([]byte(stripMarkdownFences(raw)), &payload); err != nil {
		return "", fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	ans := strings.TrimSpace(payload.Answer)
	if ans == "" {
		return "", ErrEmptyOutput
	}
	return ans, nil
}
