package cartoes

import (
	"context"
	"errors"
	"financeirogo/internal/contas"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) PayInvoice(ctx context.Context, card, invoice, account string, apply func(*Invoice, *contas.Account) error) (Invoice, error) {
	cardID, err := numericID(card)
	if err != nil {
		return Invoice{}, err
	}
	invoiceID, err := numericID(invoice)
	if err != nil {
		return Invoice{}, err
	}
	accountID, err := numericID(account)
	if err != nil {
		return Invoice{}, contas.ErrNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Invoice{}, err
	}
	defer rollbackInvoice(tx)
	i, err := scanInvoice(tx.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM financeiro.invoices WHERE card_id=$1 AND id=$2 FOR UPDATE`, cardID, invoiceID))
	if err != nil {
		return Invoice{}, err
	}
	if err := loadPurchases(ctx, tx, &i); err != nil {
		return Invoice{}, err
	}
	if i.Payment != nil {
		before := cloneInvoice(i)
		if err := apply(&i, nil); err != nil {
			return Invoice{}, err
		}
		return before, nil
	}
	var a contas.Account
	err = tx.QueryRow(ctx, `SELECT id::text,to_char(initial_balance_date,'YYYY-MM-DD'),balance_cents FROM financeiro.accounts WHERE id=$1 FOR UPDATE`, accountID).Scan(&a.ID, &a.InitialBalanceDate, &a.BalanceCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, contas.ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	if err := apply(&i, &a); err != nil {
		return Invoice{}, err
	}
	if i.Payment == nil || i.Status != "paid" {
		return Invoice{}, ErrInvoiceInvalid
	}
	var sequence int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0) FROM financeiro.entries WHERE account_id=$1`, accountID).Scan(&sequence); err != nil {
		return Invoice{}, err
	}
	if sequence == math.MaxInt64 {
		return Invoice{}, contas.ErrOverflow
	}
	sequence++
	i.Payment.EntryID = fmt.Sprintf("%s-%d", account, sequence)
	_, err = tx.Exec(ctx, `INSERT INTO financeiro.entries(id,account_id,sequence,kind,description,amount_cents,entry_date,invoice_id) VALUES($1,$2,$3,'invoice_payment',$4,$5,$6::text::date,$7)`, i.Payment.EntryID, accountID, sequence, "Pagamento integral da fatura "+invoice, i.TotalCents, i.Payment.Date, invoiceID)
	if err != nil {
		return Invoice{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE financeiro.accounts SET balance_cents=$1 WHERE id=$2`, a.BalanceCents, accountID); err != nil {
		return Invoice{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE financeiro.invoices SET status='paid',paid_account_id=$1,paid_date=$2::text::date,paid_entry_id=$3 WHERE id=$4`, accountID, i.Payment.Date, i.Payment.EntryID, invoiceID); err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return i, nil
}
