package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

var _ touchpoint.MessagingPort = (*Sender)(nil)

// MessageSender is the minimal subset of *whatsmeow.Client needed for outbound delivery.
type MessageSender interface {
	SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error)
}

// Sender implements touchpoint.MessagingPort over a WhatsApp MessageSender.
type Sender struct {
	transport MessageSender
}

func NewSender(transport MessageSender) *Sender {
	return &Sender{transport: transport}
}

func (s *Sender) SendText(ctx context.Context, recipientPhoneNumber string, body string) error {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ErrEmptyMessageBody
	}
	jid, err := PhoneToJID(recipientPhoneNumber)
	if err != nil {
		return err
	}

	_, err = s.transport.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(trimmed),
	})
	if err != nil {
		return fmt.Errorf("whatsapp: send text to %s: %w", jid.String(), err)
	}
	return nil
}

func (s *Sender) SendDrillPrompt(ctx context.Context, recipientPhoneNumber string, prompt string) error {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return ErrEmptyMessageBody
	}
	formatted := formatDrillForMobile(trimmed)
	return s.SendText(ctx, recipientPhoneNumber, formatted)
}

func formatDrillForMobile(prompt string) string {
	return fmt.Sprintf("*Quick Drill*\n\n%s", prompt)
}
