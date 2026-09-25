-- Cadastro somente: não gera receitas, despesas, saldo ou faturas.
CREATE TABLE financeiro.cards (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 100)
);
