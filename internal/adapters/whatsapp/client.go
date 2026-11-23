package whatsapp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"

	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

// Config configures the WhatsApp multi-device client.
type Config struct {
	SessionDBPath string
	QROutput      io.Writer
	Logger        *slog.Logger
}

// LifecycleClient is the minimal subset of *whatsmeow.Client needed for lifecycle/QR testing.
type LifecycleClient interface {
	IsLoggedIn() bool
	HasSession() bool
	GetQRChannel(ctx context.Context) (<-chan whatsmeow.QRChannelItem, error)
	Connect() error
	Disconnect()
}

type realLifecycleClient struct {
	cli *whatsmeow.Client
}

func (r *realLifecycleClient) IsLoggedIn() bool { return r.cli.IsLoggedIn() }
func (r *realLifecycleClient) HasSession() bool { return r.cli.Store != nil && r.cli.Store.ID != nil }
func (r *realLifecycleClient) GetQRChannel(ctx context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	return r.cli.GetQRChannel(ctx)
}
func (r *realLifecycleClient) Connect() error { return r.cli.Connect() }
func (r *realLifecycleClient) Disconnect()    { r.cli.Disconnect() }

// Client wraps a whatsmeow multi-device session, outbound MessagingPort, and inbound Dispatcher.
type Client struct {
	*Sender
	lifecycle  LifecycleClient
	dispatcher *Dispatcher
	qrOutput   io.Writer
	logger     *slog.Logger
	container  *sqlstore.Container
}

// NewClient initializes a persistent whatsmeow client backed by SQLite.
func NewClient(ctx context.Context, cfg Config, handler touchpoint.InboundMessageHandler) (*Client, error) {
	if strings.TrimSpace(cfg.SessionDBPath) == "" {
		return nil, ErrMissingSessionDB
	}
	qrOut := cfg.QROutput
	if qrOut == nil {
		qrOut = os.Stdout
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	dsn := cfg.SessionDBPath
	if !strings.Contains(dsn, "_pragma=foreign_keys") && !strings.Contains(dsn, "_foreign_keys") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "_pragma=foreign_keys(1)"
	}

	container, err := sqlstore.New(ctx, "sqlite", dsn, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: open session store: %w", err)
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: get device store: %w", err)
	}

	waClient := whatsmeow.NewClient(deviceStore, waLog.Noop)
	dispatcher := NewDispatcher(handler, logger)
	waClient.AddEventHandler(func(evt any) {
		dispatcher.HandleEvent(ctx, evt)
	})

	return &Client{
		Sender:     NewSender(waClient),
		lifecycle:  &realLifecycleClient{cli: waClient},
		dispatcher: dispatcher,
		qrOutput:   qrOut,
		logger:     logger,
		container:  container,
	}, nil
}

// NewClientWithLifecycle builds a Client around an injectable LifecycleClient for deterministic tests.
func NewClientWithLifecycle(lifecycle LifecycleClient, sender MessageSender, handler touchpoint.InboundMessageHandler, qrOut io.Writer) *Client {
	if qrOut == nil {
		qrOut = io.Discard
	}
	return &Client{
		Sender:     NewSender(sender),
		lifecycle:  lifecycle,
		dispatcher: NewDispatcher(handler, slog.Default()),
		qrOutput:   qrOut,
		logger:     slog.Default(),
	}
}

// Start connects to WhatsApp (rendering a terminal QR code if unpaired) and blocks until ctx is cancelled.
func (c *Client) Start(ctx context.Context) error {
	var qrDone chan struct{}
	if !c.lifecycle.HasSession() {
		qrChan, err := c.lifecycle.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("whatsapp: get qr channel: %w", err)
		}
		if err := c.lifecycle.Connect(); err != nil {
			return fmt.Errorf("whatsapp: connect: %w", err)
		}
		qrDone = make(chan struct{})
		go func() {
			defer close(qrDone)
			for item := range qrChan {
				if item.Event == "code" {
					qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, c.qrOutput)
				}
			}
		}()
	} else {
		if err := c.lifecycle.Connect(); err != nil {
			return fmt.Errorf("whatsapp: connect: %w", err)
		}
	}

	<-ctx.Done()
	c.lifecycle.Disconnect()
	if qrDone != nil {
		<-qrDone
	}
	if c.container != nil {
		_ = c.container.Close()
	}
	return nil
}
