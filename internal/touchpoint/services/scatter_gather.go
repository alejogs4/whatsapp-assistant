package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

// CrossPathSynthesizer executes Scatter-Gather concurrent corpus retrieval across
// multiple active LearningPaths while strictly preserving Bounded Context isolation
// (zero MasteryLedger mutations).
type CrossPathSynthesizer struct {
	corpus    learning.SourceCorpusPort
	synthesis touchpoint.SynthesisPort
	limit     int
}

func NewCrossPathSynthesizer(
	corpus learning.SourceCorpusPort,
	synthesis touchpoint.SynthesisPort,
	limit int,
) *CrossPathSynthesizer {
	if limit <= 0 {
		limit = DefaultCorpusLimit
	}
	return &CrossPathSynthesizer{
		corpus:    corpus,
		synthesis: synthesis,
		limit:     limit,
	}
}

func (s *CrossPathSynthesizer) Synthesize(
	ctx context.Context,
	userID shared.UserID,
	question string,
	matchedPaths []*learningdomain.LearningPath,
) (string, error) {
	if len(matchedPaths) == 0 {
		return "", fmt.Errorf("scatter_gather: at least one path is required")
	}

	contexts := make([]touchpoint.PathCorpusContext, len(matchedPaths))
	errs := make([]error, len(matchedPaths))

	var wg sync.WaitGroup
	for i, p := range matchedPaths {
		wg.Add(1)
		go func(idx int, path *learningdomain.LearningPath) {
			defer wg.Done()
			chunks, err := s.corpus.RetrieveChunks(ctx, userID, path.ID, question, s.limit)
			if err != nil {
				errs[idx] = fmt.Errorf("path %q: %w", path.ID, err)
				return
			}
			texts := make([]string, 0, len(chunks))
			for _, ch := range chunks {
				texts = append(texts, ch.Content)
			}
			contexts[idx] = touchpoint.PathCorpusContext{
				PathID:    path.ID,
				PathTitle: path.Title,
				Chunks:    texts,
			}
		}(i, p)
	}
	wg.Wait()

	nonEmpty := make([]touchpoint.PathCorpusContext, 0, len(contexts))
	for i, pc := range contexts {
		if errs[i] != nil {
			continue
		}
		if len(pc.Chunks) > 0 {
			nonEmpty = append(nonEmpty, pc)
		}
	}

	if len(nonEmpty) == 0 {
		for _, err := range errs {
			if err != nil {
				return "", fmt.Errorf("scatter_gather: all path retrievals failed: %w", err)
			}
		}
		return s.synthesis.AnswerGeneral(ctx, question)
	}

	return s.synthesis.SynthesizeCrossPath(ctx, question, nonEmpty)
}
