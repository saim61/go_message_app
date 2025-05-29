CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE messages (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  room       TEXT NOT NULL,
  author_id  INT  NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);