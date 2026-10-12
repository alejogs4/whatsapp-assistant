package touchpoint

import (
	"context"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

type TouchpointRepository interface {
	FindByID(ctx context.Context, id shared.SessionID) (*domain.TouchpointSession, error)
	FindActiveSession(ctx context.Context, userID shared.UserID) (*domain.TouchpointSession, error)
	Save(ctx context.Context, session *domain.TouchpointSession) error
}

type MessagingPort interface {
	SendText(ctx context.Context, recipientPhoneNumber string, body string) error
	SendDrillPrompt(ctx context.Context, recipientPhoneNumber string, prompt string) error
}

type DrillRequest struct {
	UserID     shared.UserID
	PathID     shared.PathID
	ConceptKey shared.ConceptKey
	Corpus     []string
	Type       domain.TouchpointType
}

type DrillResponse struct {
	PromptText string
}

type EvaluateRequest struct {
	Prompt     string
	UserAnswer string
	Corpus     []string
}

type EvaluateResponse struct {
	Score    float64
	Feedback string
	Passed   bool
}

type LLMPort interface {
	GenerateMicroDrill(ctx context.Context, req DrillRequest) (*DrillResponse, error)
	EvaluateAnswer(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error)
}

type CognitiveBandwidth string

const (
	BandwidthHigh CognitiveBandwidth = "HIGH"
	BandwidthLow  CognitiveBandwidth = "LOW"
)

type CalendarTelemetryPort interface {
	GetCognitiveBandwidth(ctx context.Context, userID shared.UserID, date time.Time) (CognitiveBandwidth, error)
}
