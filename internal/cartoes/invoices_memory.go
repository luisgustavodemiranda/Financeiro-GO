package cartoes

import (
	"context"
	"strconv"
)

func (r *MemoryRepository) hasCard(id string) bool {
	for _, card := range r.cards {
		if card.ID == id {
			return true
		}
	}
	return false
}

func (r *MemoryRepository) CreateInvoice(ctx context.Context, invoice Invoice) (Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Invoice{}, err
	}
	if !r.hasCard(invoice.CardID) {
		return Invoice{}, ErrNotFound
	}
	for _, existing := range r.invoices {
		if existing.CardID == invoice.CardID && invoice.StartDate <= existing.ClosingDate && existing.StartDate <= invoice.ClosingDate {
			return Invoice{}, ErrConflict
		}
	}
	invoice.ID = strconv.Itoa(len(r.invoices) + 1)
	r.invoices = append(r.invoices, cloneInvoice(invoice))
	return cloneInvoice(invoice), nil
}

func (r *MemoryRepository) ListInvoices(ctx context.Context, cardID string) ([]Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !r.hasCard(cardID) {
		return nil, ErrNotFound
	}
	invoices := []Invoice{}
	for _, invoice := range r.invoices {
		if invoice.CardID == cardID {
			invoices = append(invoices, cloneInvoice(invoice))
		}
	}
	return invoices, nil
}

func (r *MemoryRepository) GetInvoice(ctx context.Context, cardID, invoiceID string) (Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return Invoice{}, err
	}
	for _, invoice := range r.invoices {
		if invoice.CardID == cardID && invoice.ID == invoiceID {
			return cloneInvoice(invoice), nil
		}
	}
	return Invoice{}, ErrNotFound
}

func (r *MemoryRepository) UpdateInvoice(ctx context.Context, cardID, invoiceID string, change func(*Invoice) error) (Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Invoice{}, err
	}
	for i, before := range r.invoices {
		if before.CardID != cardID || before.ID != invoiceID {
			continue
		}
		after := cloneInvoice(before)
		if err := change(&after); err != nil {
			return Invoice{}, err
		}
		if !validInvoiceChange(before, after) {
			return Invoice{}, ErrInvoiceInvalid
		}
		if err := ctx.Err(); err != nil {
			return Invoice{}, err
		}
		if len(after.Purchases) > len(before.Purchases) {
			after.Purchases[len(after.Purchases)-1].ID = after.ID + "-" + strconv.Itoa(len(after.Purchases))
		}
		r.invoices[i] = cloneInvoice(after)
		return cloneInvoice(after), nil
	}
	return Invoice{}, ErrNotFound
}
