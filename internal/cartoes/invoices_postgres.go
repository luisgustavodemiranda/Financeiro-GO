package cartoes

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

const invoiceColumns = `id::text,card_id::text,to_char(start_date,'YYYY-MM-DD'),to_char(closing_date,'YYYY-MM-DD'),to_char(due_date,'YYYY-MM-DD'),status,total_cents`

func rollbackInvoice(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func numericID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != id {
		return 0, ErrNotFound
	}
	return n, nil
}

func scanInvoice(row pgx.Row) (Invoice, error) {
	i := Invoice{Purchases: []Purchase{}}
	err := row.Scan(&i.ID, &i.CardID, &i.StartDate, &i.ClosingDate, &i.DueDate, &i.Status, &i.TotalCents)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return i, err
}

func loadPurchases(ctx context.Context, tx pgx.Tx, invoice *Invoice) error {
	id, err := numericID(invoice.ID)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,description,amount_cents,to_char(purchase_date,'YYYY-MM-DD') FROM financeiro.purchases WHERE invoice_id=$1 ORDER BY id`, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var purchase Purchase
		if err := rows.Scan(&purchase.ID, &purchase.Description, &purchase.AmountCents, &purchase.Date); err != nil {
			return err
		}
		invoice.Purchases = append(invoice.Purchases, purchase)
	}
	return rows.Err()
}

func (r *PostgresRepository) CreateInvoice(ctx context.Context, invoice Invoice) (Invoice, error) {
	cardID, err := numericID(invoice.CardID)
	if err != nil {
		return Invoice{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer rollbackInvoice(tx)
	// O bloqueio do cartão serializa a verificação de sobreposição e o INSERT.
	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM financeiro.cards WHERE id=$1 FOR UPDATE`, cardID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	var overlap bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM financeiro.invoices WHERE card_id=$1 AND start_date <= $3::date AND closing_date >= $2::date)`, cardID, invoice.StartDate, invoice.ClosingDate).Scan(&overlap)
	if err != nil {
		return Invoice{}, err
	}
	if overlap {
		return Invoice{}, ErrConflict
	}
	created, err := scanInvoice(tx.QueryRow(ctx, `INSERT INTO financeiro.invoices(card_id,start_date,closing_date,due_date,status) VALUES($1,$2,$3,$4,'open') RETURNING `+invoiceColumns, cardID, invoice.StartDate, invoice.ClosingDate, invoice.DueDate))
	if err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return created, nil
}

func (r *PostgresRepository) ListInvoices(ctx context.Context, card string) ([]Invoice, error) {
	cardID, err := numericID(card)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer rollbackInvoice(tx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM financeiro.cards WHERE id=$1)`, cardID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT `+invoiceColumns+` FROM financeiro.invoices WHERE card_id=$1 ORDER BY id`, cardID)
	if err != nil {
		return nil, err
	}
	invoices := []Invoice{}
	for rows.Next() {
		invoice, err := scanInvoice(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		invoices = append(invoices, invoice)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range invoices {
		if err := loadPurchases(ctx, tx, &invoices[i]); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *PostgresRepository) GetInvoice(ctx context.Context, card, invoice string) (Invoice, error) {
	cardID, err := numericID(card)
	if err != nil {
		return Invoice{}, err
	}
	invoiceID, err := numericID(invoice)
	if err != nil {
		return Invoice{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Invoice{}, err
	}
	defer rollbackInvoice(tx)
	result, err := scanInvoice(tx.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM financeiro.invoices WHERE card_id=$1 AND id=$2`, cardID, invoiceID))
	if err != nil {
		return Invoice{}, err
	}
	if err := loadPurchases(ctx, tx, &result); err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return result, nil
}

func (r *PostgresRepository) UpdateInvoice(ctx context.Context, card, invoice string, change func(*Invoice) error) (Invoice, error) {
	cardID, err := numericID(card)
	if err != nil {
		return Invoice{}, err
	}
	invoiceID, err := numericID(invoice)
	if err != nil {
		return Invoice{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer rollbackInvoice(tx)
	before, err := scanInvoice(tx.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM financeiro.invoices WHERE card_id=$1 AND id=$2 FOR UPDATE`, cardID, invoiceID))
	if err != nil {
		return Invoice{}, err
	}
	if err := loadPurchases(ctx, tx, &before); err != nil {
		return Invoice{}, err
	}
	after := cloneInvoice(before)
	if err := change(&after); err != nil {
		return Invoice{}, err
	}
	if !validInvoiceChange(before, after) {
		return Invoice{}, ErrInvoiceInvalid
	}
	if len(after.Purchases) > len(before.Purchases) {
		p := &after.Purchases[len(after.Purchases)-1]
		err := tx.QueryRow(ctx, `INSERT INTO financeiro.purchases(invoice_id,description,amount_cents,purchase_date) VALUES($1,$2,$3,$4) RETURNING id::text`, invoiceID, p.Description, p.AmountCents, p.Date).Scan(&p.ID)
		if err != nil {
			return Invoice{}, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE financeiro.invoices SET status=$1,total_cents=$2 WHERE id=$3`, after.Status, after.TotalCents, invoiceID); err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return after, nil
}
