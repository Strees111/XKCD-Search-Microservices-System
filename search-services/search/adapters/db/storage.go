package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"yadro.com/course/search/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Search(ctx context.Context, keyword string) ([]int, error) {
	var IDs []int
	err := db.conn.SelectContext(
		ctx, &IDs,
		"SELECT xkcd_id FROM comics WHERE $1 = ANY(words)",
		keyword,
	)

	return IDs, err
}

type Comics struct {
	XKCDID int            `db:"xkcd_id"`
	URL    string         `db:"url"`
	Words  pq.StringArray `db:"words"`
}

func toDomain(r Comics) core.Comics {
	return core.Comics{
		ID:    r.XKCDID,
		URL:   r.URL,
		Words: []string(r.Words),
	}
}

func (db *DB) Get(ctx context.Context, id int) (core.Comics, error) {
	var comics Comics
	err := db.conn.GetContext(
		ctx, &comics,
		"SELECT xkcd_id, url, words FROM comics WHERE xkcd_id = $1",
		id,
	)

	return toDomain(comics), err
}

func (db *DB) GetAll(ctx context.Context) ([]core.Comics, error) {
	var rows []Comics

	err := db.conn.SelectContext(ctx, &rows,
		`SELECT xkcd_id, url, words FROM comics ORDER BY xkcd_id`)
	if err != nil {
		return nil, err
	}

	comics := make([]core.Comics, 0, len(rows))
	for _, r := range rows {
		comics = append(comics, toDomain(r))
	}

	return comics, nil
}