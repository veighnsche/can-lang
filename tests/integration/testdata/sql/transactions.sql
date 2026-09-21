-- I38 scoped-transaction exercise schema. The test-only live harness issues
-- these statements one at a time through static templates; nothing here
-- splits, concatenates, or interpolates SQL. The unique name models the
-- constraint failure the live suite provokes with a duplicate commit, and
-- the identity sequence proves rolled-back inserts never land: Bob keeps
-- id 4, the committed Zed takes 7, the rolled-back insert consumes 8, and
-- the later Yara commit takes 9.
-- The drop first removes any stale probe schema so the fixed CREATE
-- statement below always wins.
DROP TABLE IF EXISTS can_i38_accounts;
CREATE TABLE IF NOT EXISTS can_i38_accounts (
  id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  display_name text NOT NULL UNIQUE
);
TRUNCATE can_i38_accounts RESTART IDENTITY;
INSERT INTO can_i38_accounts (display_name) VALUES
  ('Ann'),
  ('Jason'),
  ('Mason'),
  ('Bob'),
  ('O''Brien'),
  ('Zoë');
DROP TABLE can_i38_accounts;
