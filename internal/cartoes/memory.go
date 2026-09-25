package cartoes

import (
	"context"
	"strconv"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	cards    []Card
	invoices []Invoice
}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{} }

func (r *MemoryRepository) Create(ctx context.Context, card Card) (Card, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Card{}, err
	}
	card.ID = strconv.Itoa(len(r.cards) + 1)
	r.cards = append(r.cards, card)
	return card, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Card, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]Card{}, r.cards...), nil
}
