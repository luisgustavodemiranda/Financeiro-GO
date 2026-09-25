-- Aplicada pelo comando cmd/migrate dentro de uma única transação.
-- Somente bancos financeiro_go ou financeiro_go_test; nunca o Fiscal-GO.
CREATE SCHEMA financeiro;

CREATE TABLE financeiro.schema_migrations (
    version integer PRIMARY KEY,
    checksum text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE financeiro.accounts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> ''),
    initial_balance_cents bigint NOT NULL,
    initial_balance_date date NOT NULL CHECK (initial_balance_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
    balance_cents bigint NOT NULL
);

CREATE TABLE financeiro.entries (
    id text PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES financeiro.accounts(id),
    sequence bigint NOT NULL CHECK (sequence > 0),
    kind text NOT NULL CHECK (kind IN ('income', 'expense')),
    description text NOT NULL CHECK (btrim(description) <> ''),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    entry_date date NOT NULL CHECK (entry_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
    UNIQUE (account_id, sequence),
    CHECK (id = account_id::text || '-' || sequence::text)
);
