package whatsapp_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/whatsapp"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

func TestPhoneJIDNormalization(t *testing.T) {
	jid, err := whatsapp.PhoneToJID("+1 (555) 123-4567")
	if err != nil {
		t.Fatalf("PhoneToJID: %v", err)
	}
	if jid.User != "15551234567" || jid.Server != types.DefaultUserServer {
		t.Fatalf("unexpected JID: %+v", jid)
	}
	if got := whatsapp.JIDToPhone(jid); got != "+15551234567" {
		t.Fatalf("expected +15551234567, got %q", got)
	}
	if _, err := whatsapp.PhoneToJID("abc"); !errors.Is(err, whatsapp.ErrInvalidPhoneNumber) {
		t.Fatalf("expected ErrInvalidPhoneNumber, got %v", err)
	}
}

func TestExtractInboundMessage_FiltersAndExtracts(t *testing.T) {
	senderJID := types.NewJID("15551234567", types.DefaultUserServer)

	selfEvt := &events.Message{
		Info:    types.MessageInfo{MessageSource: types.MessageSource{IsFromMe: true, Sender: senderJID, Chat: senderJID}},
		Message: &waE2E.Message{Conversation: proto.String("self")},
	}
	if _, ok := whatsapp.ExtractInboundMessage(selfEvt); ok {
		t.Fatalf("expected self message to be filtered")
	}

	groupEvt := &events.Message{
		Info:    types.MessageInfo{MessageSource: types.MessageSource{IsGroup: true, Sender: senderJID, Chat: types.NewJID("12345", types.GroupServer)}},
		Message: &waE2E.Message{Conversation: proto.String("group")},
	}
	if _, ok := whatsapp.ExtractInboundMessage(groupEvt); ok {
		t.Fatalf("expected group message to be filtered")
	}

	broadcastEvt := &events.Message{
		Info:    types.MessageInfo{MessageSource: types.MessageSource{Sender: senderJID, Chat: types.NewJID("status", types.BroadcastServer)}},
		Message: &waE2E.Message{Conversation: proto.String("status")},
	}
	if _, ok := whatsapp.ExtractInboundMessage(broadcastEvt); ok {
		t.Fatalf("expected broadcast message to be filtered")
	}

	now := time.Date(2026, 11, 23, 15, 0, 0, 0, time.UTC)
	dmEvt := &events.Message{
		Info: types.MessageInfo{
			ID:        "msg-1",
			Timestamp: now,
			MessageSource: types.MessageSource{
				Sender: senderJID,
				Chat:   senderJID,
			},
		},
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("  Encapsulation hides state  "),
			},
		},
	}
	msg, ok := whatsapp.ExtractInboundMessage(dmEvt)
	if !ok {
		t.Fatalf("expected valid 1:1 DM to extract")
	}
	if msg.SenderPhone != "+15551234567" || msg.Text != "Encapsulation hides state" || msg.MessageID != "msg-1" {
		t.Fatalf("unexpected extracted message: %+v", msg)
	}
}

type fakeWASender struct {
	sentTo   []types.JID
	sentBody []string
}

func (f *fakeWASender) SendMessage(_ context.Context, to types.JID, msg *waE2E.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	f.sentTo = append(f.sentTo, to)
	f.sentBody = append(f.sentBody, msg.GetConversation())
	return whatsmeow.SendResponse{}, nil
}

func TestSender_OutboundTextAndDrillPrompt(t *testing.T) {
	fake := &fakeWASender{}
	sender := whatsapp.NewSender(fake)

	if err := sender.SendText(context.Background(), "+15551234567", "Great answer!"); err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if len(fake.sentTo) != 1 || fake.sentTo[0].String() != "15551234567@s.whatsapp.net" || fake.sentBody[0] != "Great answer!" {
		t.Fatalf("unexpected SendText dispatch: to=%v body=%v", fake.sentTo, fake.sentBody)
	}

	if err := sender.SendDrillPrompt(context.Background(), "+15551234567", "   "); !errors.Is(err, whatsapp.ErrEmptyMessageBody) {
		t.Fatalf("expected ErrEmptyMessageBody, got %v", err)
	}
	if err := sender.SendDrillPrompt(context.Background(), "+15551234567", "What is a Coordinator?"); err != nil {
		t.Fatalf("SendDrillPrompt: %v", err)
	}
	if len(fake.sentBody) != 2 || !strings.Contains(fake.sentBody[1], "What is a Coordinator?") {
		t.Fatalf("unexpected drill body: %v", fake.sentBody)
	}
}

type mockInboundHandler struct {
	received []touchpoint.InboundMessage
	err      error
}

func (m *mockInboundHandler) HandleInbound(_ context.Context, msg touchpoint.InboundMessage) error {
	m.received = append(m.received, msg)
	return m.err
}

func TestDispatcher_ForwardsValidDMsAndIsolatesErrors(t *testing.T) {
	handler := &mockInboundHandler{err: errors.New("boom")}
	d := whatsapp.NewDispatcher(handler, nil)

	senderJID := types.NewJID("15551234567", types.DefaultUserServer)
	groupEvt := &events.Message{
		Info:    types.MessageInfo{MessageSource: types.MessageSource{IsGroup: true, Sender: senderJID, Chat: types.NewJID("123", types.GroupServer)}},
		Message: &waE2E.Message{Conversation: proto.String("ignore me")},
	}
	dmEvt := &events.Message{
		Info:    types.MessageInfo{ID: "m-2", MessageSource: types.MessageSource{Sender: senderJID, Chat: senderJID}},
		Message: &waE2E.Message{Conversation: proto.String("answer")},
	}

	d.HandleEvent(context.Background(), groupEvt)
	d.HandleEvent(context.Background(), dmEvt)

	if len(handler.received) != 1 || handler.received[0].Text != "answer" {
		t.Fatalf("expected exactly 1 DM forwarded, got %+v", handler.received)
	}
}

type fakeLifecycle struct {
	mu           sync.Mutex
	hasSession   bool
	qrCh         chan whatsmeow.QRChannelItem
	connected    bool
	disconnected bool
}

func (f *fakeLifecycle) IsLoggedIn() bool { return f.hasSession }
func (f *fakeLifecycle) HasSession() bool { return f.hasSession }
func (f *fakeLifecycle) GetQRChannel(_ context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	return f.qrCh, nil
}
func (f *fakeLifecycle) Connect() error {
	f.mu.Lock()
	f.connected = true
	f.mu.Unlock()
	return nil
}
func (f *fakeLifecycle) Disconnect() {
	f.mu.Lock()
	f.disconnected = true
	f.mu.Unlock()
}

func TestClient_ConfigValidationAndQRLifecycle(t *testing.T) {
	if _, err := whatsapp.NewClient(context.Background(), whatsapp.Config{}, nil); !errors.Is(err, whatsapp.ErrMissingSessionDB) {
		t.Fatalf("expected ErrMissingSessionDB, got %v", err)
	}

	qrCh := make(chan whatsmeow.QRChannelItem, 1)
	qrCh <- whatsmeow.QRChannelItem{Event: "code", Code: "MOCK-QR-PAIRING-CODE"}
	close(qrCh)

	var qrBuf bytes.Buffer
	lc := &fakeLifecycle{hasSession: false, qrCh: qrCh}
	cli := whatsapp.NewClientWithLifecycle(lc, &fakeWASender{}, nil, &qrBuf)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- cli.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if !lc.connected || !lc.disconnected {
		t.Fatalf("expected connected=true and disconnected=true, got %+v", lc)
	}
	if qrBuf.Len() == 0 {
		t.Fatalf("expected QR code to be rendered to qrBuf")
	}
}
