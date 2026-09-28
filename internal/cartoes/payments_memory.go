package cartoes

import (
	"context"
	"financeirogo/internal/contas"
	"fmt"
)

type MemoryPaymentRepository struct {
	accounts *contas.MemoryRepository
	cards    *MemoryRepository
}

func NewMemoryPaymentRepository(accounts *contas.MemoryRepository, cards *MemoryRepository) *MemoryPaymentRepository {
	return &MemoryPaymentRepository{accounts: accounts, cards: cards}
}

func (r *MemoryPaymentRepository) PayInvoice(ctx context.Context, cardID, invoiceID, accountID string, apply func(*Invoice, *contas.Account) error) (Invoice, error) {
	// Ordem fixa: cartão, depois conta. Nenhuma operação de conta adquire cartão.
	r.cards.mu.Lock()
	defer r.cards.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Invoice{}, err
	}
	for index, before := range r.cards.invoices {
		if before.CardID != cardID || before.ID != invoiceID {
			continue
		}
		after := cloneInvoice(before)
		if before.Payment != nil {
			if err := apply(&after, nil); err != nil {
				return Invoice{}, err
			}
			return cloneInvoice(before), nil
		}
		err := r.accounts.Update(ctx, accountID, func(state *contas.State) error {
			if err := apply(&after, &state.Account); err != nil {
				return err
			}
			if after.Payment == nil || after.Status != "paid" {
				return ErrInvoiceInvalid
			}
			entryID := fmt.Sprintf("%s-%d", accountID, len(state.Entries)+1)
			after.Payment.EntryID = entryID
			state.Entries = append(state.Entries, contas.Entry{ID: entryID, AccountID: accountID, Kind: "invoice_payment", Description: "Pagamento integral da fatura " + invoiceID, AmountCents: after.TotalCents, Date: after.Payment.Date, InvoiceID: invoiceID})
			return nil
		})
		if err != nil {
			return Invoice{}, err
		}
		// A conta confirmou. Não há mais validação, I/O ou cancelamento a partir
		// daqui: publicar a fatura não pode falhar. Leitores aguardam este mutex.
		r.cards.invoices[index] = cloneInvoice(after)
		return cloneInvoice(after), nil
	}
	return Invoice{}, ErrNotFound
}
