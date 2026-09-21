-- I37 descriptor exercise schema. The test-only live harness issues these
-- statements one at a time through static templates; nothing here splits,
-- concatenates, or interpolates SQL. I35 reuses this database with its own
-- tables; this file documents the I37 account surface.
CREATE TABLE IF NOT EXISTS can_i37_accounts (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  display_name text NOT NULL
);
TRUNCATE can_i37_accounts RESTART IDENTITY;
INSERT INTO can_i37_accounts (display_name) VALUES
  ('Ann'),
  ('Jason'),
  ('Mason'),
  ('Bob'),
  ('O''Brien'),
  ('Zoë');
DROP TABLE can_i37_accounts;
