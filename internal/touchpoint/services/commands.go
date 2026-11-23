package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/user"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

// CommandController handles explicit slash operational commands deterministically
// without consuming LLM tokens or mutating MasteryLedger state.
type CommandController struct {
	users     user.UserRepository
	paths     learning.LearningPathRepository
	sessions  touchpoint.TouchpointRepository
	messaging touchpoint.MessagingPort
}

func NewCommandController(
	users user.UserRepository,
	paths learning.LearningPathRepository,
	sessions touchpoint.TouchpointRepository,
	messaging touchpoint.MessagingPort,
) *CommandController {
	return &CommandController{
		users:     users,
		paths:     paths,
		sessions:  sessions,
		messaging: messaging,
	}
}

// IsCommand returns true if the message starts with a recognized operational command.
func IsCommand(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "/")
}

// Handle executes an operational command and replies via MessagingPort.
// Returns handled=true if the message was processed as a command.
func (c *CommandController) Handle(ctx context.Context, u *userdomain.User, text string) (bool, error) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return false, nil
	}

	fields := strings.Fields(trimmed)
	cmd := strings.ToLower(fields[0])

	switch cmd {
	case "/skip":
		return true, c.handleSkip(ctx, u)
	case "/pause":
		return true, c.handleSetPathsStatus(ctx, u, learningdomain.PathStatusPaused, "All active learning paths paused. Send /resume whenever you're ready.")
	case "/resume":
		return true, c.handleSetPathsStatus(ctx, u, learningdomain.PathStatusActive, "Learning paths resumed! Ready for your next scheduled touchpoint.")
	case "/stats":
		return true, c.handleStats(ctx, u)
	case "/schedule":
		return true, c.handleSchedule(ctx, u, fields[1:])
	default:
		return true, c.messaging.SendText(ctx, u.PhoneNumber, "Unknown command. Available: /pause, /resume, /skip, /stats, /schedule morning|evening <0-23>")
	}
}

func (c *CommandController) handleSkip(ctx context.Context, u *userdomain.User) error {
	active, err := c.sessions.FindActiveSession(ctx, u.ID)
	if errors.Is(err, shared.ErrTouchpointSessionNotFound) {
		return c.messaging.SendText(ctx, u.PhoneNumber, "No active drill to skip.")
	}
	if err != nil {
		return fmt.Errorf("commands: find active session: %w", err)
	}
	active.State = tpdomain.SessionStateExpired
	if err := c.sessions.Save(ctx, active); err != nil {
		return fmt.Errorf("commands: expire skipped session: %w", err)
	}
	return c.messaging.SendText(ctx, u.PhoneNumber, "Skipped current drill. No penalty recorded.")
}

func (c *CommandController) handleSetPathsStatus(
	ctx context.Context,
	u *userdomain.User,
	target learningdomain.PathStatus,
	reply string,
) error {
	paths, err := c.paths.FindByUser(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("commands: find paths: %w", err)
	}
	for _, p := range paths {
		if p.Status == learningdomain.PathStatusCompleted {
			continue
		}
		p.Status = target
		if err := c.paths.Save(ctx, p); err != nil {
			return fmt.Errorf("commands: save path %q: %w", p.ID, err)
		}
	}
	return c.messaging.SendText(ctx, u.PhoneNumber, reply)
}

func (c *CommandController) handleStats(ctx context.Context, u *userdomain.User) error {
	paths, err := c.paths.FindByUser(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("commands: find paths: %w", err)
	}
	activeCount := 0
	for _, p := range paths {
		if p.Status == learningdomain.PathStatusActive {
			activeCount++
		}
	}
	msg := fmt.Sprintf(
		"Stats: %d active path(s), %d/2 daily touches used, schedule %02d:00 / %02d:00 (%s).",
		activeCount,
		u.DailyTouchesUsed,
		u.CircadianBaseline.MorningCatalystHour,
		u.CircadianBaseline.EveningReflectionHour,
		u.Timezone,
	)
	return c.messaging.SendText(ctx, u.PhoneNumber, msg)
}

func (c *CommandController) handleSchedule(ctx context.Context, u *userdomain.User, args []string) error {
	if len(args) != 2 {
		return c.messaging.SendText(ctx, u.PhoneNumber, "Usage: /schedule morning <0-23> or /schedule evening <0-23>")
	}
	slot := strings.ToLower(args[0])
	hour, err := strconv.Atoi(args[1])
	if err != nil || hour < 0 || hour > 23 {
		return c.messaging.SendText(ctx, u.PhoneNumber, "Hour must be an integer between 0 and 23.")
	}
	switch slot {
	case "morning":
		u.CircadianBaseline.MorningCatalystHour = hour
	case "evening":
		u.CircadianBaseline.EveningReflectionHour = hour
	default:
		return c.messaging.SendText(ctx, u.PhoneNumber, "Slot must be 'morning' or 'evening'.")
	}

	if err := c.users.Save(ctx, u); err != nil {
		return fmt.Errorf("commands: save user schedule: %w", err)
	}
	return c.messaging.SendText(ctx, u.PhoneNumber, fmt.Sprintf("Updated %s touchpoint hour to %02d:00 (%s).", slot, hour, u.Timezone))
}
