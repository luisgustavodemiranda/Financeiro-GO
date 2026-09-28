-- Pagamento é saída de caixa, sem reconhecer outra despesa.
ALTER TABLE financeiro.entries DROP CONSTRAINT entries_kind_check;
ALTER TABLE financeiro.entries ADD CONSTRAINT entries_kind_check
    CHECK (kind IN ('income','expense','invoice_payment'));
ALTER TABLE financeiro.entries ADD COLUMN invoice_id bigint REFERENCES financeiro.invoices(id);
ALTER TABLE financeiro.entries ADD CONSTRAINT entries_invoice_payment_check
    CHECK ((kind = 'invoice_payment') = (invoice_id IS NOT NULL));
ALTER TABLE financeiro.entries ADD CONSTRAINT entries_invoice_id_key UNIQUE (invoice_id);

ALTER TABLE financeiro.invoices DROP CONSTRAINT invoices_status_check;
ALTER TABLE financeiro.invoices ADD CONSTRAINT invoices_status_check CHECK (status IN ('open','closed','paid'));
ALTER TABLE financeiro.invoices ADD COLUMN paid_account_id bigint REFERENCES financeiro.accounts(id);
ALTER TABLE financeiro.invoices ADD COLUMN paid_date date;
ALTER TABLE financeiro.invoices ADD COLUMN paid_entry_id text UNIQUE REFERENCES financeiro.entries(id);
ALTER TABLE financeiro.invoices ADD CONSTRAINT invoices_payment_check CHECK (
    (status = 'paid' AND total_cents > 0 AND paid_account_id IS NOT NULL AND paid_entry_id IS NOT NULL
     AND paid_date IS NOT NULL AND paid_date >= closing_date AND paid_date <= DATE '9999-12-31')
    OR (status <> 'paid' AND paid_account_id IS NULL AND paid_date IS NULL AND paid_entry_id IS NULL)
);
