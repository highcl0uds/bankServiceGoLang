CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(72)  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_email    ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

CREATE TABLE IF NOT EXISTS accounts (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance    NUMERIC(20,2)  NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
    currency   CHAR(3)        NOT NULL DEFAULT 'RUB',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS cards (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id       UUID        NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    number_encrypted TEXT        NOT NULL,
    number_hmac      BYTEA       NOT NULL,
    expiry_encrypted TEXT        NOT NULL,
    cvv_hash         VARCHAR(72) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cards_account_id ON cards(account_id);

DO $$ BEGIN
    CREATE TYPE transaction_type AS ENUM (
        'deposit','withdrawal','transfer_out','transfer_in',
        'credit_disbursement','credit_repayment'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS transactions (
    id              UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    from_account_id UUID             REFERENCES accounts(id),
    to_account_id   UUID             REFERENCES accounts(id),
    amount          NUMERIC(20,2)    NOT NULL CHECK (amount > 0),
    type            transaction_type NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_transactions_from ON transactions(from_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_to   ON transactions(to_account_id,   created_at DESC);

DO $$ BEGIN
    CREATE TYPE credit_status AS ENUM ('active','paid','defaulted');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS credits (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id        UUID          NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    principal         NUMERIC(20,2) NOT NULL,
    interest_rate     NUMERIC(6,4)  NOT NULL,
    term_months       INT           NOT NULL,
    monthly_payment   NUMERIC(20,2) NOT NULL,
    remaining_balance NUMERIC(20,2) NOT NULL,
    status            credit_status NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credits_account_id  ON credits(account_id);
CREATE INDEX IF NOT EXISTS idx_credits_active      ON credits(status) WHERE status = 'active';

DO $$ BEGIN
    CREATE TYPE schedule_status AS ENUM ('pending','paid','overdue');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS payment_schedules (
    id        UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_id UUID            NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    due_date  DATE            NOT NULL,
    amount    NUMERIC(20,2)   NOT NULL,
    status    schedule_status NOT NULL DEFAULT 'pending',
    paid_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_schedules_credit_id ON payment_schedules(credit_id, due_date);
CREATE INDEX IF NOT EXISTS idx_schedules_pending   ON payment_schedules(due_date, status) WHERE status IN ('pending','overdue');
