package publisher

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	log *slog.Logger
	nc  *nats.Conn
}

func New(nc *nats.Conn, log *slog.Logger) *Publisher {
	return &Publisher{nc: nc, log: log}
}

func (p *Publisher) PublishDatabaseUpdated(ctx context.Context) error {
	if err := p.nc.Publish("xkcd.db.updated", nil); err != nil {
		p.log.Error("could not publish message", "error", err)
		return err
	}
	return p.nc.Flush()
}

func (p *Publisher) PublishDatabaseDrop(ctx context.Context) error {
	if err := p.nc.Publish("xkcd.db.drop", nil); err != nil {
		p.log.Error("could not publish drop event", "error", err)
		return err
	}
	return p.nc.Flush()
}
