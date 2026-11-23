package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/llm/gemini"
	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	"github.com/alejogs4/whatsapp-assistant/internal/adapters/whatsapp"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/services"
)

type runner interface {
	Start(ctx context.Context) error
}

// runDaemon starts the scheduler and messaging transport and waits for context cancellation.
func runDaemon(ctx context.Context, sched runner, transport runner) error {
	errCh := make(chan error, 2)

	go func() {
		errCh <- sched.Start(ctx)
	}()

	if transport != nil {
		go func() {
			errCh <- transport.Start(ctx)
		}()
	}

	<-ctx.Done()
	if err := <-errCh; err != nil {
		return err
	}
	if transport != nil {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := LoadConfigFromEnv(os.Getenv)
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := boot(ctx, cfg, logger); err != nil {
		logger.Error("daemon exited with error", "err", err)
		os.Exit(1)
	}
}

func boot(ctx context.Context, cfg Config, logger *slog.Logger) error {
	db, err := sqlite.Open(cfg.SQLiteDSN)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}

	llmAdapter, err := gemini.NewAdapter(ctx, gemini.Config{APIKey: cfg.GeminiAPIKey})
	if err != nil {
		return fmt.Errorf("init gemini adapter: %w", err)
	}

	userRepo := sqlite.NewUserRepo(db)
	pathRepo := sqlite.NewLearningPathRepo(db)
	ledgerRepo := sqlite.NewMasteryLedgerRepo(db)
	sessionRepo := sqlite.NewTouchpointRepo(db)
	corpusStore := sqlite.NewCorpusStore(db, llmAdapter, nil)

	var routerRef touchpoint.InboundMessageHandler
	lazyHandler := inboundHandlerFunc(func(ctx context.Context, msg touchpoint.InboundMessage) error {
		if routerRef == nil {
			return nil
		}
		return routerRef.HandleInbound(ctx, msg)
	})

	waClient, err := whatsapp.NewClient(ctx, whatsapp.Config{
		SessionDBPath: cfg.WhatsAppSessionDB,
		Logger:        logger,
	}, lazyHandler)
	if err != nil {
		return fmt.Errorf("init whatsapp client: %w", err)
	}

	arbitrator := services.NewAttentionArbitrator(pathRepo, ledgerRepo, nil)
	coordinator := services.NewDrillCoordinator(userRepo, corpusStore, sessionRepo, llmAdapter, waClient, arbitrator)
	evaluator := services.NewPedagogicalEvaluator(corpusStore, ledgerRepo, sessionRepo, llmAdapter, waClient)
	commands := services.NewCommandController(userRepo, pathRepo, sessionRepo, waClient)
	scatterGather := services.NewCrossPathSynthesizer(corpusStore, llmAdapter, services.DefaultCorpusLimit)
	macroCoach := services.NewMacroCoach(pathRepo, ledgerRepo, llmAdapter, services.DefaultMasteryThreshold)
	routerRef = services.NewInboundRouter(
		userRepo, pathRepo, corpusStore, sessionRepo, waClient,
		llmAdapter, llmAdapter, commands, evaluator, scatterGather, macroCoach, services.RealClock{},
	)
	scheduler := services.NewScheduler(userRepo, sessionRepo, coordinator, waClient, services.RealClock{}, cfg.TickInterval, logger)

	logger.Info("whatsapp adaptive learning daemon started")
	return runDaemon(ctx, scheduler, waClient)
}

type inboundHandlerFunc func(ctx context.Context, msg touchpoint.InboundMessage) error

func (f inboundHandlerFunc) HandleInbound(ctx context.Context, msg touchpoint.InboundMessage) error {
	return f(ctx, msg)
}
