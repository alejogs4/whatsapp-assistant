package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

func TestTouchpointSession_LifecycleHappyPath(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	expiresAt := now.Add(2 * time.Hour)
	answerTime := now.Add(15 * time.Minute)
	completeTime := now.Add(16 * time.Minute)

	session := domain.NewTouchpointSession(
		shared.SessionID("ses_100"),
		shared.UserID("usr_100"),
		shared.PathID("path_100"),
		shared.ConceptKey("clean_architecture"),
		domain.TouchpointTypeMorningCatalyst,
		now,
	)

	if session.State != domain.SessionStatePending {
		t.Fatalf("expected initial state PENDING, got %v", session.State)
	}

	// 1. Dispatch prompt
	prompt := "Explain the dependency rule in Hexagonal Architecture."
	if err := session.DispatchPrompt(prompt, expiresAt); err != nil {
		t.Fatalf("DispatchPrompt failed: %v", err)
	}
	if session.State != domain.SessionStateAwaitingAnswer {
		t.Fatalf("expected state AWAITING_ANSWER, got %v", session.State)
	}
	if session.PromptText != prompt || !session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("prompt fields mismatch: text=%s, expiresAt=%v", session.PromptText, session.ExpiresAt)
	}

	// 2. Submit answer before expiration
	userAnswer := "Dependencies point inward towards domain entities and use cases."
	if err := session.SubmitAnswer(answerTime, userAnswer); err != nil {
		t.Fatalf("SubmitAnswer failed: %v", err)
	}
	if session.State != domain.SessionStateEvaluating {
		t.Fatalf("expected state EVALUATING, got %v", session.State)
	}
	if session.UserAnswer != userAnswer {
		t.Fatalf("expected userAnswer=%s, got %s", userAnswer, session.UserAnswer)
	}

	// 3. Complete evaluation
	feedback := "Excellent explanation adhering to clean architecture principles."
	if err := session.Complete(completeTime, feedback); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if session.State != domain.SessionStateCompleted {
		t.Fatalf("expected state COMPLETED, got %v", session.State)
	}
	if session.EvaluationFeedback != feedback {
		t.Fatalf("feedback mismatch: expected %s, got %s", feedback, session.EvaluationFeedback)
	}
	if session.CompletedAt == nil || !session.CompletedAt.Equal(completeTime) {
		t.Fatalf("expected CompletedAt=%v, got %v", completeTime, session.CompletedAt)
	}
}

func TestTouchpointSession_ExpirationAndSilentBackoff(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	expiresAt := now.Add(1 * time.Hour) // 10:00:00

	t.Run("timely submission succeeds and does not expire", func(t *testing.T) {
		session := domain.NewTouchpointSession(
			shared.SessionID("ses_timely"),
			shared.UserID("usr_1"),
			shared.PathID("path_1"),
			shared.ConceptKey("rdd"),
			domain.TouchpointTypeMorningCatalyst,
			now,
		)
		_ = session.DispatchPrompt("Drill prompt", expiresAt)

		beforeExpiry := expiresAt.Add(-5 * time.Minute)
		if expired := session.CheckExpiration(beforeExpiry); expired {
			t.Fatalf("expected session not expired, but CheckExpiration returned true")
		}
		if session.State != domain.SessionStateAwaitingAnswer {
			t.Fatalf("state should remain AWAITING_ANSWER, got %v", session.State)
		}

		err := session.SubmitAnswer(beforeExpiry, "On time answer")
		if err != nil {
			t.Fatalf("expected timely submission to succeed, got %v", err)
		}
		if session.State != domain.SessionStateEvaluating {
			t.Fatalf("expected state EVALUATING, got %v", session.State)
		}
	})

	t.Run("late submission transitions to EXPIRED and returns ErrSessionExpired", func(t *testing.T) {
		session := domain.NewTouchpointSession(
			shared.SessionID("ses_late"),
			shared.UserID("usr_1"),
			shared.PathID("path_1"),
			shared.ConceptKey("rdd"),
			domain.TouchpointTypeMorningCatalyst,
			now,
		)
		_ = session.DispatchPrompt("Drill prompt", expiresAt)

		lateTime := expiresAt.Add(1 * time.Minute) // 10:01:00
		err := session.SubmitAnswer(lateTime, "Late answer")
		if !errors.Is(err, shared.ErrSessionExpired) {
			t.Fatalf("expected ErrSessionExpired, got %v", err)
		}
		if session.State != domain.SessionStateExpired {
			t.Fatalf("expected state EXPIRED, got %v", session.State)
		}

		// Subsequent submission attempts should also fail with ErrSessionExpired
		errAgain := session.SubmitAnswer(lateTime.Add(5*time.Minute), "Another attempt")
		if !errors.Is(errAgain, shared.ErrSessionExpired) {
			t.Fatalf("expected ErrSessionExpired on repeated late attempt, got %v", errAgain)
		}
	})

	t.Run("explicit CheckExpiration triggers EXPIRED transition", func(t *testing.T) {
		session := domain.NewTouchpointSession(
			shared.SessionID("ses_check_exp"),
			shared.UserID("usr_1"),
			shared.PathID("path_1"),
			shared.ConceptKey("rdd"),
			domain.TouchpointTypeMorningCatalyst,
			now,
		)
		_ = session.DispatchPrompt("Drill prompt", expiresAt)

		pastExpiry := expiresAt.Add(10 * time.Second)
		if !session.CheckExpiration(pastExpiry) {
			t.Fatal("expected CheckExpiration to return true past expiry")
		}
		if session.State != domain.SessionStateExpired {
			t.Fatalf("expected state EXPIRED, got %v", session.State)
		}
	})
}

func TestTouchpointSession_InvalidStateTransitions(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(2 * time.Hour)

	tests := []struct {
		name       string
		setup      func() *domain.TouchpointSession
		action     func(s *domain.TouchpointSession) error
		expectedErr error
	}{
		{
			name: "SubmitAnswer while Pending",
			setup: func() *domain.TouchpointSession {
				return domain.NewTouchpointSession(
					shared.SessionID("s1"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
			},
			action: func(s *domain.TouchpointSession) error {
				return s.SubmitAnswer(now, "Premature answer")
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "Complete while Pending",
			setup: func() *domain.TouchpointSession {
				return domain.NewTouchpointSession(
					shared.SessionID("s2"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
			},
			action: func(s *domain.TouchpointSession) error {
				return s.Complete(now, "Premature complete")
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "Complete while AwaitingAnswer",
			setup: func() *domain.TouchpointSession {
				s := domain.NewTouchpointSession(
					shared.SessionID("s3"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
				_ = s.DispatchPrompt("prompt", expiresAt)
				return s
			},
			action: func(s *domain.TouchpointSession) error {
				return s.Complete(now, "Complete without answer")
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "DispatchPrompt while already AwaitingAnswer",
			setup: func() *domain.TouchpointSession {
				s := domain.NewTouchpointSession(
					shared.SessionID("s4"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
				_ = s.DispatchPrompt("prompt", expiresAt)
				return s
			},
			action: func(s *domain.TouchpointSession) error {
				return s.DispatchPrompt("second prompt", expiresAt)
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "DispatchPrompt after Completed",
			setup: func() *domain.TouchpointSession {
				s := domain.NewTouchpointSession(
					shared.SessionID("s5"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
				_ = s.DispatchPrompt("prompt", expiresAt)
				_ = s.SubmitAnswer(now.Add(5*time.Minute), "answer")
				_ = s.Complete(now.Add(6*time.Minute), "feedback")
				return s
			},
			action: func(s *domain.TouchpointSession) error {
				return s.DispatchPrompt("prompt again", expiresAt)
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "SubmitAnswer after Completed",
			setup: func() *domain.TouchpointSession {
				s := domain.NewTouchpointSession(
					shared.SessionID("s6"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
				_ = s.DispatchPrompt("prompt", expiresAt)
				_ = s.SubmitAnswer(now.Add(5*time.Minute), "answer")
				_ = s.Complete(now.Add(6*time.Minute), "feedback")
				return s
			},
			action: func(s *domain.TouchpointSession) error {
				return s.SubmitAnswer(now.Add(10*time.Minute), "another answer")
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
		{
			name: "Complete after Expired",
			setup: func() *domain.TouchpointSession {
				s := domain.NewTouchpointSession(
					shared.SessionID("s7"), shared.UserID("u1"), shared.PathID("p1"), shared.ConceptKey("c1"),
					domain.TouchpointTypeMorningCatalyst, now,
				)
				_ = s.DispatchPrompt("prompt", expiresAt)
				_ = s.SubmitAnswer(expiresAt.Add(10*time.Minute), "too late")
				return s
			},
			action: func(s *domain.TouchpointSession) error {
				return s.Complete(now, "feedback for expired")
			},
			expectedErr: shared.ErrInvalidSessionStateTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()
			err := tt.action(s)
			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}
