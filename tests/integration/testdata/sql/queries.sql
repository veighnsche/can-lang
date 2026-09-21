-- I35 typed pool/query exercise schema. The test-only live harness issues
-- these statements one at a time through static templates; nothing here
-- splits, concatenates, or interpolates SQL. The unique name models the
-- constraint failure the live suite provokes with a duplicate insert.
-- The drops first remove any stale probe schema so the fixed CREATE
-- statements below always win.
DROP TABLE IF EXISTS can_i35_cover;
DROP TABLE IF EXISTS can_i35_accounts;
CREATE TABLE IF NOT EXISTS can_i35_accounts (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  display_name text NOT NULL UNIQUE
);
TRUNCATE can_i35_accounts RESTART IDENTITY;
INSERT INTO can_i35_accounts (display_name) VALUES
  ('Ann'),
  ('Jason'),
  ('Mason'),
  ('Bob'),
  ('O''Brien'),
  ('Zoë');
CREATE TABLE IF NOT EXISTS can_i35_cover (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  payload bytea NOT NULL,
  note text NULL
);
TRUNCATE can_i35_cover RESTART IDENTITY;
INSERT INTO can_i35_cover (payload, note) VALUES
  ('\x4142', 'n'),
  ('\x', NULL);
DROP TABLE can_i35_cover;
DROP TABLE can_i35_accounts;
