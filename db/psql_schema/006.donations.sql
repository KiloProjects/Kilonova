
CREATE TYPE donation_source AS enum (
    'buymeacoffee',
    'paypal',
    'other'
);

CREATE TYPE donation_type AS enum (
    'onetime',
    'monthly',
    'yearly'
)

CREATE TABLE IF NOT EXISTS donations (
    donated_at timestamptz      NOT NULL,
    user_id    bigint           REFERENCES users(id),
    amount     real             NOT NULL DEFAULT 0,

    source     donation_source  NOT NULL DEFAULT 'other',
    type       donation_type    NOT NULL DEFAULT 'onetime',

    transaction_id text         NOT NULL
)