package cartoes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

var _ Repository = (*PostgresRepository)(nil)
var _ Repository = (*MemoryRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool, timeout time.Duration) *PostgresRepository {
	return &PostgresRepository{pool: pool, timeout: timeout}
}

func (r *PostgresRepository) Create(ctx context.Context, card Card) (Card, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	err := r.pool.QueryRow(ctx, `INSERT INTO financeiro.cards(name) VALUES($1) RETURNING id::text,name`, card.Name).Scan(&card.ID, &card.Name)
	if err != nil {
		return Card{}, err
	}
	return card, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Card, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT id::text,name FROM financeiro.cards ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := []Card{}
	for rows.Next() {
		var card Card
		if err := rows.Scan(&card.ID, &card.Name); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}
