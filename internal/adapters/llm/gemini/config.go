package gemini

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingAPIKey      = errors.New("gemini: API key is required")
	ErrEmptyCorpus        = errors.New("gemini: grounded corpus chunks must not be empty")
	ErrPromptTooLong      = errors.New("gemini: generated micro-drill exceeds word limit")
	ErrEmptyOutput        = errors.New("gemini: model returned empty output")
	ErrInvalidScore       = errors.New("gemini: evaluation score must be between 0.0 and 1.0")
	ErrMalformedJSON      = errors.New("gemini: model returned malformed JSON")
	ErrEmbeddingCount     = errors.New("gemini: embedding count mismatch")
)

const (
	DefaultTextModel      = "gemini-2.5-flash"
	DefaultEmbeddingModel = "text-embedding-004"
	DefaultMaxRetries     = 1
	DefaultEmbedBatchSize = 100
	DefaultTimeout        = 30 * time.Second

	MaxMorningDrillWords = 199
	MaxEveningDrillWords = 60
	PassingScoreFloor    = 0.6
)

// Config configures the Gemini adapter.
type Config struct {
	APIKey         string
	TextModel      string
	EmbeddingModel string
	MaxRetries     int
	EmbedBatchSize int
	Timeout        time.Duration
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.TextModel) == "" {
		c.TextModel = DefaultTextModel
	}
	if strings.TrimSpace(c.EmbeddingModel) == "" {
		c.EmbeddingModel = DefaultEmbeddingModel
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.EmbedBatchSize <= 0 {
		c.EmbedBatchSize = DefaultEmbedBatchSize
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}
