package core

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusIdle    ServiceStatus = "idle"
)

type DBStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
}

type ServiceStats struct {
	DBStats
	ComicsTotal int `json:"comics_total"`
}

type Comics struct {
	ID    int
	URL   string
	Words []string
}

type XKCDInfo struct {
	ID          int
	URL         string
	Title       string
	Description string
	Transcript  string
}
