-- Compras no cartão não alteram contas nem lançamentos bancários.
CREATE TABLE financeiro.invoices (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    card_id bigint NOT NULL REFERENCES financeiro.cards(id),
    start_date date NOT NULL CHECK (start_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
    closing_date date NOT NULL CHECK (closing_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
    due_date date NOT NULL CHECK (due_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
    status text NOT NULL CHECK (status IN ('open', 'closed')),
    total_cents bigint NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    CHECK (start_date <= closing_date AND closing_date < due_date)
);
CREATE INDEX invoices_card_id_idx ON financeiro.invoices(card_id, id);

CREATE TABLE financeiro.purchases (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id bigint NOT NULL REFERENCES financeiro.invoices(id),
    description text NOT NULL CHECK (btrim(description) <> '' AND char_length(description) <= 200),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    purchase_date date NOT NULL CHECK (purchase_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31')
);
CREATE INDEX purchases_invoice_id_idx ON financeiro.purchases(invoice_id, id);
