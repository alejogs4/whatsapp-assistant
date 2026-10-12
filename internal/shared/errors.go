package shared

import "errors"

var (
	ErrTouchpointBudgetExhausted      = errors.New("touchpoint budget exhausted for current calendar day")
	ErrInvalidSessionStateTransition = errors.New("invalid session state transition")
	ErrSessionExpired                = errors.New("touchpoint session has expired; silent backoff enforced")
	ErrUserNotFound                  = errors.New("user not found")
	ErrLearningPathNotFound          = errors.New("learning path not found")
	ErrInvalidMasteryScore           = errors.New("mastery score must be between 0.0 and 1.0")
	ErrInvalidTimezone               = errors.New("invalid or unparseable IANA timezone identifier")
)
