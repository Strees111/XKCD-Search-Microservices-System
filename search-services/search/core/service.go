package core

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
	index map[string][]int
	mx    *sync.Mutex
	nc    EventSubscriber
}

func NewService(log *slog.Logger, db DB, words Words, index map[string][]int, mx *sync.Mutex, nc EventSubscriber) (*Service, error) {

	return &Service{
		log:   log,
		db:    db,
		words: words,
		index: index,
		mx:    mx,
		nc:    nc,
	}, nil
}

func (s *Service) ListenForUpdates(ctx context.Context) error {
	return s.nc.SubscribeForUpdates(ctx, func() error {
		s.BuildIndex(ctx)
		return nil
	})
}

func (s *Service) ListenForDrop(ctx context.Context) error {
	return s.nc.SubscribeForDrop(ctx, func() error {
		s.ResetIndex()
		return nil
	})
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comics, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to find keywords", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)

	// comics ID -> number of findings
	scores := map[int]int{}
	for _, keyword := range keywords {
		IDs, err := s.db.Search(ctx, keyword)
		if err != nil {
			s.log.Error("failed to search keyword in DB", "error", err)
			return nil, err
		}
		for _, ID := range IDs {
			scores[ID]++
		}
	}
	s.log.Debug("relevant comics", "count", len(scores))

	// sort by number of findings
	sorted := slices.SortedFunc(maps.Keys(scores), func(a, b int) int {
		return cmp.Compare(scores[b], scores[a]) // desc
	})

	// limit results
	if len(sorted) < limit {
		limit = len(sorted)
	}
	sorted = sorted[:limit]

	// fetch comics
	result := make([]Comics, 0, len(sorted))
	for _, ID := range sorted {
		comics, err := s.db.Get(ctx, ID)
		if err != nil {
			s.log.Error("failed to fetch comics", "id", ID, "error", err)
			return nil, err
		}
		result = append(result, comics)
	}
	s.log.Debug("returning comics", "count", len(result))
	return result, nil
}

func (s *Service) SearchIndex(ctx context.Context, phrase string, limit int) ([]Comics, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to find keywords", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)

	// comics ID -> number of findings
	scores := map[int]int{}
	for _, keyword := range keywords {
		s.mx.Lock()
		IDs := s.index[keyword]
		s.mx.Unlock()
		for _, ID := range IDs {
			scores[ID]++
		}
	}
	s.log.Debug("relevant comics", "count", len(scores))

	// sort by number of findings
	sorted := slices.SortedFunc(maps.Keys(scores), func(a, b int) int {
		return cmp.Compare(scores[b], scores[a]) // desc
	})

	// limit results
	if len(sorted) < limit {
		limit = len(sorted)
	}
	sorted = sorted[:limit]

	// fetch comics
	result := make([]Comics, 0, len(sorted))
	for _, ID := range sorted {
		comics, err := s.db.Get(ctx, ID)
		if err != nil {
			s.log.Debug("comic not found in db", "id", ID)
			continue
		}
		result = append(result, comics)
	}
	s.log.Debug("returning comics", "count", len(result))

	return result, nil
}

func (s *Service) BuildIndex(ctx context.Context) {
	comics, err := s.db.GetAll(ctx)
	if err != nil {
		s.log.Error("failed to find comics", "error", err)
		return
	}

	newIndex := make(map[string][]int)

	for _, comic := range comics {
		for _, word := range comic.Words {
			newIndex[word] = append(newIndex[word], comic.ID)
		}
	}

	s.mx.Lock()
	s.index = newIndex
	s.mx.Unlock()

	s.log.Info("index built", "words_count", len(newIndex))
}

func (s *Service) ResetIndex() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.index = make(map[string][]int)
	s.log.Info("index reset")
}
