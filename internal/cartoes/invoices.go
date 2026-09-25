package cartoes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvoiceInvalid = errors.New("dados de fatura ou compra inválidos")
	ErrNotFound       = errors.New("cartão ou fatura não encontrado")
	ErrConflict       = errors.New("fatura fechada ou período sobreposto")
	ErrOverflow       = errors.New("total da fatura excede int64")
)

type Purchase struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
	Date        string `json:"date"`
}

type Invoice struct {
	ID          string     `json:"id"`
	CardID      string     `json:"card_id"`
	StartDate   string     `json:"start_date"`
	ClosingDate string     `json:"closing_date"`
	DueDate     string     `json:"due_date"`
	Status      string     `json:"status"`
	TotalCents  int64      `json:"total_cents"`
	Purchases   []Purchase `json:"purchases"`
}

func validInvoiceDate(value string) bool {
	d, err := time.Parse("2006-01-02", value)
	return err == nil && d.Year() >= 1 && d.Year() <= 9999 && d.Format("2006-01-02") == value
}

func (s *Service) CreateInvoice(ctx context.Context, cardID, start, closing, due string) (Invoice, error) {
	if !validInvoiceDate(start) || !validInvoiceDate(closing) || !validInvoiceDate(due) || start > closing || due <= closing {
		return Invoice{}, fmt.Errorf("%w: datas AAAA-MM-DD, início <= fechamento < vencimento", ErrInvoiceInvalid)
	}
	return s.repo.CreateInvoice(ctx, Invoice{CardID: cardID, StartDate: start, ClosingDate: closing, DueDate: due, Status: "open", Purchases: []Purchase{}})
}

func (s *Service) ListInvoices(ctx context.Context, cardID string) ([]Invoice, error) {
	return s.repo.ListInvoices(ctx, cardID)
}

func (s *Service) GetInvoice(ctx context.Context, cardID, invoiceID string) (Invoice, error) {
	return s.repo.GetInvoice(ctx, cardID, invoiceID)
}

func (s *Service) RegisterPurchase(ctx context.Context, cardID, invoiceID, description string, amount int64, date string) (Invoice, error) {
	description = strings.TrimSpace(description)
	if description == "" || !utf8.ValidString(description) || strings.ContainsRune(description, 0) || utf8.RuneCountInString(description) > 200 || amount <= 0 || !validInvoiceDate(date) {
		return Invoice{}, ErrInvoiceInvalid
	}
	return s.repo.UpdateInvoice(ctx, cardID, invoiceID, func(invoice *Invoice) error {
		if invoice.Status != "open" {
			return ErrConflict
		}
		if date < invoice.StartDate || date > invoice.ClosingDate {
			return ErrInvoiceInvalid
		}
		if invoice.TotalCents > math.MaxInt64-amount {
			return ErrOverflow
		}
		invoice.Purchases = append(invoice.Purchases, Purchase{Description: description, AmountCents: amount, Date: date})
		invoice.TotalCents += amount
		return nil
	})
}

// Fechar novamente é idempotente: devolve a mesma fatura, sem nova movimentação.
func (s *Service) CloseInvoice(ctx context.Context, cardID, invoiceID string) (Invoice, error) {
	return s.repo.UpdateInvoice(ctx, cardID, invoiceID, func(invoice *Invoice) error {
		invoice.Status = "closed"
		return nil
	})
}

func cloneInvoice(invoice Invoice) Invoice {
	invoice.Purchases = append([]Purchase{}, invoice.Purchases...)
	return invoice
}

// Restringe a atualização a uma compra ou ao fechamento, preservando o histórico.
func validInvoiceChange(before, after Invoice) bool {
	if before.ID != after.ID || before.CardID != after.CardID || before.StartDate != after.StartDate || before.ClosingDate != after.ClosingDate || before.DueDate != after.DueDate {
		return false
	}
	if slices.Equal(before.Purchases, after.Purchases) {
		return before.TotalCents == after.TotalCents && after.Status == "closed"
	}
	if before.Status != "open" || after.Status != "open" || len(after.Purchases) != len(before.Purchases)+1 || !slices.Equal(before.Purchases, after.Purchases[:len(before.Purchases)]) {
		return false
	}
	p := after.Purchases[len(after.Purchases)-1]
	return p.ID == "" && p.AmountCents > 0 && before.TotalCents <= math.MaxInt64-p.AmountCents && after.TotalCents == before.TotalCents+p.AmountCents
}
