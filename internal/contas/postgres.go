package contas

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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

const accountColumns = `id::text, name, initial_balance_cents,
 to_char(initial_balance_date, 'YYYY-MM-DD'), balance_cents`

func scanAccount(row pgx.Row) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Name, &a.InitialBalanceCents, &a.InitialBalanceDate, &a.BalanceCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

func accountID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != id {
		return 0, ErrNotFound
	}
	return n, nil
}

func (r *PostgresRepository) Create(ctx context.Context, a Account) (Account, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return scanAccount(r.pool.QueryRow(ctx, `INSERT INTO financeiro.accounts
 (name, initial_balance_cents, initial_balance_date, balance_cents)
 VALUES ($1, $2, $3::text::date, $2) RETURNING `+accountColumns,
		a.Name, a.InitialBalanceCents, a.InitialBalanceDate))
}

func (r *PostgresRepository) List(ctx context.Context) ([]Account, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT `+accountColumns+` FROM financeiro.accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func readState(ctx context.Context, tx pgx.Tx, id int64, lock bool) (State, error) {
	query := `SELECT ` + accountColumns + ` FROM financeiro.accounts WHERE id=$1`
	if lock {
		query += ` FOR UPDATE`
	}
	a, err := scanAccount(tx.QueryRow(ctx, query, id))
	if err != nil {
		return State{}, err
	}
	rows, err := tx.Query(ctx, `SELECT id, account_id::text, kind, description, amount_cents,
 to_char(entry_date, 'YYYY-MM-DD') FROM financeiro.entries WHERE account_id=$1 ORDER BY sequence`, id)
	if err != nil {
		return State{}, err
	}
	defer rows.Close()
	state := State{Account: a, Entries: []Entry{}}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.AccountID, &e.Kind, &e.Description, &e.AmountCents, &e.Date); err != nil {
			return State{}, err
		}
		state.Entries = append(state.Entries, e)
	}
	return state, rows.Err()
}

// Rollback precisa funcionar mesmo se o contexto da requisição foi cancelado.
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (State, error) {
	n, err := accountID(id)
	if err != nil {
		return State{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	// As duas consultas enxergam o mesmo snapshot, mesmo com novos lançamentos.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return State{}, err
	}
	defer rollback(tx)
	state, err := readState(ctx, tx, n, false)
	if err != nil {
		return State{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, err
	}
	return state, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, change func(*State) error) error {
	n, err := accountID(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer rollback(tx)
	// Serializa escritores da mesma conta, inclusive em processos diferentes.
	before, err := readState(ctx, tx, n, true)
	if err != nil {
		return err
	}
	after := clone(before)
	if err := change(&after); err != nil {
		return err
	}
	if err := validateAppend(before, after); err != nil {
		return err
	}
	e := after.Entries[len(before.Entries)]
	if e.AccountID != id {
		return ErrInvalid
	}
	_, err = tx.Exec(ctx, `INSERT INTO financeiro.entries
 (id, account_id, sequence, kind, description, amount_cents, entry_date)
 VALUES ($1,$2,$3,$4,$5,$6,$7::text::date)`,
		e.ID, n, len(after.Entries), e.Kind, e.Description, e.AmountCents, e.Date)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE financeiro.accounts SET balance_cents=$1 WHERE id=$2`, after.Account.BalanceCents, n)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
