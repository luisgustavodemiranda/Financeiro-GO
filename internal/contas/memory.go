package contas

import (
	"context"
	"strconv"
	"sync"
)

// MemoryRepository perde todos os dados quando o processo termina.
type MemoryRepository struct {
	mu     sync.RWMutex
	states []State
}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{} }

func (r *MemoryRepository) Create(ctx context.Context, account Account) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	account.ID = strconv.Itoa(len(r.states) + 1)
	r.states = append(r.states, State{Account: account, Entries: []Entry{}})
	return account, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	accounts := make([]Account, 0, len(r.states))
	for _, state := range r.states {
		accounts = append(accounts, state.Account)
	}
	return accounts, nil
}

func clone(state State) State {
	state.Entries = append([]Entry{}, state.Entries...)
	return state
}

func (r *MemoryRepository) Get(ctx context.Context, id string) (State, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	for _, state := range r.states {
		if state.Account.ID == id {
			return clone(state), nil
		}
	}
	return State{}, ErrNotFound
}

func (r *MemoryRepository) Update(ctx context.Context, id string, change func(*State) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, state := range r.states {
		if state.Account.ID != id {
			continue
		}
		next := clone(state)
		if err := change(&next); err != nil {
			return err
		}
		if err := validateAppend(state, next); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		r.states[i] = clone(next)
		return nil
	}
	return ErrNotFound
}
