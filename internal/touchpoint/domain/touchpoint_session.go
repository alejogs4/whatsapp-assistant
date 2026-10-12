package domain

import (
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type SessionState string

const (
	SessionStatePending        SessionState = "PENDING"
	SessionStateAwaitingAnswer SessionState = "AWAITING_ANSWER"
	SessionStateEvaluating     SessionState = "EVALUATING"
	SessionStateCompleted      SessionState = "COMPLETED"
	SessionStateExpired        SessionState = "EXPIRED"
)

type TouchpointType string

const (
	TouchpointTypeMorningCatalyst    TouchpointType = "MORNING_CATALYST"
	TouchpointTypeEveningStreakSaver TouchpointType = "EVENING_STREAK_SAVER"
)

type TouchpointSession struct {
	ID                 shared.SessionID
	UserID             shared.UserID
	PathID             shared.PathID
	ConceptKey         shared.ConceptKey
	State              SessionState
	TouchpointType     TouchpointType
	PromptText         string
	UserAnswer         string
	EvaluationFeedback string
	CreatedAt          time.Time
	ExpiresAt          time.Time
	CompletedAt        *time.Time
}

func NewTouchpointSession(
	id shared.SessionID,
	userID shared.UserID,
	pathID shared.PathID,
	concept shared.ConceptKey,
	tpType TouchpointType,
	now time.Time,
) *TouchpointSession {
	return &TouchpointSession{
		ID:             id,
		UserID:         userID,
		PathID:         pathID,
		ConceptKey:     concept,
		State:          SessionStatePending,
		TouchpointType: tpType,
		CreatedAt:      now,
	}
}

func (s *TouchpointSession) DispatchPrompt(prompt string, expiresAt time.Time) error {
	if s.State != SessionStatePending {
		return shared.ErrInvalidSessionStateTransition
	}
	s.PromptText = prompt
	s.ExpiresAt = expiresAt
	s.State = SessionStateAwaitingAnswer
	return nil
}

func (s *TouchpointSession) SubmitAnswer(now time.Time, answer string) error {
	if s.CheckExpiration(now) {
		return shared.ErrSessionExpired
	}
	if s.State != SessionStateAwaitingAnswer {
		return shared.ErrInvalidSessionStateTransition
	}
	s.UserAnswer = answer
	s.State = SessionStateEvaluating
	return nil
}

func (s *TouchpointSession) Complete(now time.Time, feedback string) error {
	if s.State != SessionStateEvaluating {
		return shared.ErrInvalidSessionStateTransition
	}
	s.EvaluationFeedback = feedback
	s.CompletedAt = &now
	s.State = SessionStateCompleted
	return nil
}

func (s *TouchpointSession) CheckExpiration(now time.Time) bool {
	if s.State == SessionStateAwaitingAnswer && now.After(s.ExpiresAt) {
		s.State = SessionStateExpired
		return true
	}
	return s.State == SessionStateExpired
}
