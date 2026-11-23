package whatsapp

import (
	"context"
	"log/slog"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

// Dispatcher translates raw whatsmeow events into domain InboundMessages
// and invokes the configured InboundMessageHandler safely.
type Dispatcher struct {
	handler touchpoint.InboundMessageHandler
	logger  *slog.Logger
}

func NewDispatcher(handler touchpoint.InboundMessageHandler, logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Dispatcher{
		handler: handler,
		logger:  logger,
	}
}

// HandleEvent processes any whatsmeow event; non-1:1-DM events are ignored.
func (d *Dispatcher) HandleEvent(ctx context.Context, rawEvt any) {
	msgEvt, ok := rawEvt.(*events.Message)
	if !ok {
		return
	}
	msg, valid := ExtractInboundMessage(msgEvt)
	if !valid {
		return
	}
	if d.handler == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("whatsapp: panic recovered in inbound handler", "recover", r, "msg_id", msg.MessageID)
		}
	}()

	if err := d.handler.HandleInbound(ctx, msg); err != nil {
		d.logger.Error("whatsapp: inbound handler failed", "err", err, "msg_id", msg.MessageID)
	}
}
