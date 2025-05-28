CREATE TABLE users (
  id            SERIAL PRIMARY KEY,
  username      TEXT UNIQUE NOT NULL,
  password TEXT NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now()
);