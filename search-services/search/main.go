package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"yadro.com/course/closers"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/adapters/db"
	searchgrpc "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/adapters/subscribe"
	"yadro.com/course/search/adapters/words"
	"yadro.com/course/search/config"
	"yadro.com/course/search/core"
)

func main() {

	// config
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	// logger
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	nc, err := nats.Connect(cfg.BrokerAddress)
	if err != nil {
		panic(err)
	}
	defer nc.Close()
	newSubscriber := subscribe.New(nc, log)
	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer closers.CloseOrLog(storage, log)

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer closers.CloseOrLog(words, log)

	mx := new(sync.Mutex)
	index := make(map[string][]int)
	// service
	searcher, err := core.NewService(log, storage, words, index, mx, newSubscriber)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	searcher.BuildIndex(context.Background())
	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServiceServer(s, searchgrpc.NewServer(searcher))
	reflection.Register(s)

	// context for Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err = searcher.ListenForUpdates(ctx); err != nil {
		log.Error("listen for updates failed", "error", err)
	}

	if err = searcher.ListenForDrop(ctx); err != nil {
		log.Error("drop subscription failed", "error", err)
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
