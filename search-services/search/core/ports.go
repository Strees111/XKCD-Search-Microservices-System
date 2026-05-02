package core

import (
	"context"
)

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comics, error)
	SearchIndex(ctx context.Context, phrase string, limit int) ([]Comics, error)
	BuildIndex(ctx context.Context)
}

type DB interface {
	Search(ctx context.Context, keyword string) ([]int, error)
	Get(ctx context.Context, ID int) (Comics, error)
	GetAll(ctx context.Context) ([]Comics, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type EventSubscriber interface {
	SubscribeForUpdates(ctx context.Context, onUpdate func() error) error
	SubscribeForDrop(ctx context.Context, onDrop func() error) error
}
