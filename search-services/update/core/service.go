package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int
	mx          sync.Mutex
	wg          sync.WaitGroup
	updateed    bool
	status      ServiceStatus
	nc          EventPublisher
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, concurrency int, nc EventPublisher,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
		nc:          nc,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	s.mx.Lock()
	if s.updateed {
		s.mx.Unlock()
		return ErrAlreadyExists
	}
	s.updateed = true
	s.status = StatusRunning
	s.mx.Unlock()

	defer func() {
		s.mx.Lock()
		s.updateed = false
		s.status = StatusIdle
		s.mx.Unlock()
	}()

	existingIDs, err := s.db.IDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	existingSet := make(map[int]bool)
	for _, v := range existingIDs {
		existingSet[v] = true
	}

	var toFetch []int
	total, err := s.xkcd.LastID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get total comics: %w", err)
	}
	for i := 1; i <= total; i++ {
		if !existingSet[i] {
			toFetch = append(toFetch, i)
		}
	}

	jobs := make(chan int, len(toFetch))
	for _, v := range toFetch {
		jobs <- v
	}
	close(jobs)
	CountWorkers := s.concurrency
	for i := 0; i < CountWorkers; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for id := range jobs {
				xkcd, err := s.xkcd.Get(ctx, id)
				if err != nil {
					s.log.Error("failed to fetch comic", "id", id, "error", err)
					comic := Comics{
						ID:    id,
						URL:   "",
						Words: []string{},
					}
					if err := s.db.Add(ctx, comic); err != nil {
						s.log.Error("failed to add missing comic", "id", id, "error", err)
					}
					continue
				}
				combinedText := xkcd.Title + " " + xkcd.Description + " " + xkcd.Transcript
				words, err := s.words.Norm(ctx, combinedText)
				if err != nil {
					s.log.Error("failed to normalize words", "id", id, "error", err)
					comic := Comics{
						ID:    xkcd.ID,
						URL:   xkcd.URL,
						Words: []string{},
					}
					if err := s.db.Add(ctx, comic); err != nil {
						s.log.Error("failed to add comic", "id", id, "error", err)
					}
					continue
				}
				if words == nil {
					words = []string{}
				}
				comic := Comics{
					ID:    xkcd.ID,
					URL:   xkcd.URL,
					Words: words,
				}
				if err := s.db.Add(ctx, comic); err != nil {
					s.log.Error("failed to add comic", "id", id, "error", err)
					continue
				}
				s.log.Debug("processed comic", "id", id)
			}
		}()
	}
	s.wg.Wait()
	if s.nc != nil && len(toFetch) > 0 {
		if err := s.nc.PublishDatabaseUpdated(ctx); err != nil {
			s.log.Error("failed to publish update event", "error", err)
		}
	}
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("failed to get stats: %w", err)
	}
	total, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("failed to get total comics: %w", err)
	}
	stats := ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: total,
	}
	return stats, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.status
}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop: %w", err)
	}

	if s.nc != nil {
		if err := s.nc.PublishDatabaseDrop(ctx); err != nil {
			s.log.Error("failed to publish drop event", "error", err)
			return err
		}
	}

	return nil
}
