package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	normalize "yadro.com/course/words/words"
)

type Config struct {
	Port string `yaml:"port" env:"WORDS_GRPC_PORT" env-default:"8080"`
}

func LoadConfig(configPath string) Config {
	var cfg Config

	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			slog.Error("failed to read config file", "err", err)
			os.Exit(1)
		}
	}

	// Always apply environment variables (they override file values)
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		slog.Error("failed to read env variables", "err", err)
		os.Exit(1)
	}

	if cfg.Port == "" {
		slog.Error("port is not set")
		os.Exit(1)
	}

	return cfg
}

type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	phrase := in.GetPhrase()
	if len(phrase) > (1 << 12) {
		return nil, status.Error(codes.ResourceExhausted, "String is too long")
	}
	words := normalize.Normalize(phrase)
	return &wordspb.WordsReply{
		Words: words,
	}, nil
}

func main() {
	var address string
	configPath := flag.String("config", "", "path to config file")
	flag.StringVar(&address, "address", "", "server address")
	flag.Parse()
	cfg := LoadConfig(*configPath)
	if address == "" {
		address = ":" + cfg.Port
	}
	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, &server{})
	reflection.Register(s)

	slog.Info("starting server", "port", cfg.Port)
	if err := StartServer(address, s); err != nil {
		slog.Error("failed to start server", "err", err)
	}
}

func StartServer(address string, r *grpc.Server) error {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		slog.Error("failed to listen: ", "err", err)
	}
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		if err := r.Serve(listener); err != nil {
			slog.Error("failed to serve: ", "err", err)
		}
		return nil
	})

	group.Go(func() error {
		<-groupCtx.Done()
		done := make(chan struct{})
		go func() {
			r.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(60 * time.Second):
			return errors.New("failed to stop")
		}
		return nil
	})

	return group.Wait()
}
