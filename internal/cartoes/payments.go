package cartoes

import (
	"context"
	"errors"
	"financeirogo/internal/contas"
	"math"
)

var ErrPaymentConflict = errors.New("pagamento exige fatura fechada com valor positivo; pagamento existente não pode ser alterado")

type Payment struct {
	AccountID   string `json:"account_id"`
	Date        string `json:"date"`
	AmountCents int64  `json:"amount_cents"`
	EntryID     string `json:"entry_id"`
}

// PaymentRepository coordena conta e fatura atomicamente. Em repetição, account
// pode ser nil: o serviço deve conferir o pagamento existente sem novo débito.
type PaymentRepository interface {
	PayInvoice(context.Context, string, string, string, func(*Invoice, *contas.Account) error) (Invoice, error)
}

type PaymentService struct{ repo PaymentRepository }

func NewPaymentService(repo PaymentRepository) *PaymentService { return &PaymentService{repo: repo} }

func (s *PaymentService) Pay(ctx context.Context, cardID, invoiceID, accountID, date string) (Invoice, error) {
	if accountID == "" || !validInvoiceDate(date) {
		return Invoice{}, ErrInvoiceInvalid
	}
	return s.repo.PayInvoice(ctx, cardID, invoiceID, accountID, func(i *Invoice, a *contas.Account) error {
		if i.Payment != nil {
			if i.Payment.AccountID != accountID || i.Payment.Date != date {
				return ErrPaymentConflict
			}
			return nil
		}
		if i.Status != "closed" || i.TotalCents <= 0 {
			return ErrPaymentConflict
		}
		if a == nil {
			return contas.ErrNotFound
		}
		if date < i.ClosingDate || date < a.InitialBalanceDate {
			return ErrInvoiceInvalid
		}
		if a.BalanceCents < math.MinInt64+i.TotalCents {
			return contas.ErrOverflow
		}
		a.BalanceCents -= i.TotalCents
		i.Status = "paid"
		i.Payment = &Payment{AccountID: accountID, Date: date, AmountCents: i.TotalCents}
		return nil
	})
}

func samePayment(a, b *Payment) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
