CREATE TABLE users (
	id text PRIMARY KEY,
	email text NOT NULL UNIQUE,
	name text NOT NULL DEFAULT '',
	password_hash text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE households (
	id text PRIMARY KEY,
	name text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
	household_id text NOT NULL REFERENCES households(id) ON DELETE CASCADE,
	user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
	PRIMARY KEY (household_id, user_id)
);

CREATE TABLE incomes (
	household_id text NOT NULL REFERENCES households(id) ON DELETE CASCADE,
	user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	month date NOT NULL,
	amount_cents bigint NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
	PRIMARY KEY (household_id, user_id, month)
);

CREATE TABLE expenses (
	id text PRIMARY KEY,
	household_id text NOT NULL REFERENCES households(id) ON DELETE CASCADE,
	name text NOT NULL,
	amount_cents bigint NOT NULL CHECK (amount_cents > 0),
	paid_by text NOT NULL REFERENCES users(id),
	is_common boolean NOT NULL DEFAULT true,
	split_mode text NOT NULL CHECK (split_mode IN ('equal', 'proportional')),
	month date NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX expenses_household_month_idx ON expenses (household_id, month);
CREATE INDEX incomes_household_month_idx ON incomes (household_id, month);
