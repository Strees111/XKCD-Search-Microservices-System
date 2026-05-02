package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

type Status struct {
	Status core.UpdateStatus `json:"status"`
}

type SearchResponse struct {
	Comics []core.Comics `json:"comics"`
	Total  int           `json:"total"`
}

type Person struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	}
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := PingResponse{
			Replies: make(map[string]string),
		}
		for service, pinger := range pingers {
			err := pinger.Ping(r.Context())
			if err != nil {
				log.Error("ping failed", "service", service, "error", err)
				res.Replies[service] = "unavailable"
			} else {
				res.Replies[service] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			log.Error("encode ping response failed", "error", err)
		}
	}
}

type Authenticator interface {
	Login(user, password string) (string, error)
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var person Person
		if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token, err := auth.Login(person.Name, person.Password)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(token))
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Update(r.Context())
		if err != nil {
			if err == core.ErrAlreadyExists {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		UpdateStats, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("stats failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(core.UpdateStats{
			WordsTotal:    UpdateStats.WordsTotal,
			WordsUnique:   UpdateStats.WordsUnique,
			ComicsFetched: UpdateStats.ComicsFetched,
			ComicsTotal:   UpdateStats.ComicsTotal,
		}); err != nil {
			log.Error("encode ping response failed", "error", err)
		}
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateStatus, err := updater.Status(r.Context())
		if err != nil {
			log.Error("status failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Status{Status: updateStatus}); err != nil {
			log.Error("encode status response failed", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Drop(r.Context())
		if err != nil {
			log.Error("status failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "phrase is required", http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			l, err := strconv.Atoi(limitStr)
			if err != nil || l <= 0 {
				http.Error(w, "invalid limit parameter", http.StatusBadRequest)
				return
			}
			limit = l
		}
		comics, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			log.Error("search failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		res := SearchResponse{
			Comics: comics,
			Total:  len(comics),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			log.Error("encode status response failed", "error", err)
		}
	}
}

func NewSearchHandlerIndex(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "phrase is required", http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			l, err := strconv.Atoi(limitStr)
			if err != nil || l <= 0 {
				http.Error(w, "invalid limit parameter", http.StatusBadRequest)
				return
			}
			limit = l
		}
		comics, err := searcher.SearchIndex(r.Context(), phrase, limit)
		if err != nil {
			log.Error("search failed", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		res := SearchResponse{
			Comics: comics,
			Total:  len(comics),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			log.Error("encode status response failed", "error", err)
		}
	}
}
