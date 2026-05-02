package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
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

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	query := "insert into comics (xkcd_id, url, words) Values ($1, $2, $3) ON CONFLICT (xkcd_id) DO NOTHING"
	_, err := db.conn.ExecContext(ctx, query, comics.ID, comics.URL, comics.Words)
	return err
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var pd core.DBStats
	query := `SELECT 
		COUNT(DISTINCT xkcd_id) as ComicsFetched,
		COUNT(DISTINCT word) FILTER (WHERE word IS NOT NULL) as WordsUnique,
		COALESCE(SUM(array_length(words, 1)), 0) as WordsTotal
	FROM comics
	LEFT JOIN LATERAL unnest(words) AS t(word) ON true`
	row := db.conn.QueryRowContext(ctx, query)
	err := row.Scan(&pd.ComicsFetched, &pd.WordsUnique, &pd.WordsTotal)
	return pd, err
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int
	query := "Select xkcd_id from comics"
	err := db.conn.SelectContext(ctx, &ids, query)
	return ids, err
}

func (db *DB) Drop(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, "truncate table comics")
	return err
}
