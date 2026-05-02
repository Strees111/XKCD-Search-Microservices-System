package subscribe

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type Subscriber struct {
	log *slog.Logger
	nc  *nats.Conn
}

func New(nc *nats.Conn, log *slog.Logger) *Subscriber {
	return &Subscriber{nc: nc, log: log}
}

func (s *Subscriber) SubscribeForUpdates(ctx context.Context, onUpdate func() error) error {
	return s.subscribeTopic(ctx, "xkcd.db.updated", onUpdate)
}

func (s *Subscriber) SubscribeForDrop(ctx context.Context, onDrop func() error) error {
	return s.subscribeTopic(ctx, "xkcd.db.drop", onDrop)
}

func (s *Subscriber) subscribeTopic(ctx context.Context, topic string, handler func() error) error {
	ch := make(chan *nats.Msg, 4096)
	sub, err := s.nc.ChanSubscribe(topic, ch)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if err := sub.Unsubscribe(); err != nil {
				panic(err)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				s.log.Info("received event", "topic", topic, "data", string(msg.Data))
				drained := 0
			drainLoop:
				for {
					select {
					case <-ch:
						drained++
					default:
						break drainLoop
					}
				}
				if drained > 0 {
					s.log.Debug("drained concurrent events to single update", "topic", topic, "count", drained)
				}

				if err := handler(); err != nil {
					s.log.Error("error handling event", "topic", topic, "error", err)
				}
			}
		}
	}()
	return nil
}
