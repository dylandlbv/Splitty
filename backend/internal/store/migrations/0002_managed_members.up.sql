-- Members without a login ("managed" members) are user rows with
-- has_login = false: they have a synthetic, never-exposed email and no
-- password, and an admin manages their incomes and expenses for them.
ALTER TABLE users ADD COLUMN has_login boolean NOT NULL DEFAULT true;
