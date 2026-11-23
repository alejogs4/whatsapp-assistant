package whatsapp

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

// ExtractInboundMessage filters out self-sent messages, group chats, status broadcasts,
// and non-text payloads, returning a domain-neutral InboundMessage for valid 1:1 text DMs.
func ExtractInboundMessage(evt *events.Message) (touchpoint.InboundMessage, bool) {
	if evt == nil || evt.Message == nil {
		return touchpoint.InboundMessage{}, false
	}
	if evt.Info.IsFromMe || evt.Info.IsGroup {
		return touchpoint.InboundMessage{}, false
	}
	if evt.Info.Chat.Server == types.BroadcastServer || evt.Info.Sender.Server == types.BroadcastServer {
		return touchpoint.InboundMessage{}, false
	}

	text := extractText(evt)
	if text == "" {
		return touchpoint.InboundMessage{}, false
	}

	return touchpoint.InboundMessage{
		MessageID:   evt.Info.ID,
		SenderPhone: JIDToPhone(evt.Info.Sender),
		Text:        text,
		ReceivedAt:  evt.Info.Timestamp.UTC(),
	}, true
}

func extractText(evt *events.Message) string {
	if c := evt.Message.GetConversation(); strings.TrimSpace(c) != "" {
		return strings.TrimSpace(c)
	}
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		if t := ext.GetText(); strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
	}
	return ""
}
