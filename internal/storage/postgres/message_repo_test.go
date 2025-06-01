package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewMessageRepo(sqlxDB)

	assert.NotNil(t, repo)
	assert.Equal(t, sqlxDB, repo.db)
}

func TestMessageRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewMessageRepo(sqlxDB)

	now := time.Now().UTC().Format(time.RFC3339)

	tests := []struct {
		name    string
		message *Message
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful message save",
			message: &Message{
				ID:        "msg-123",
				Room:      "general",
				Author:    "testuser",
				Body:      "Hello, world!",
				CreatedAt: now,
			},
			mockFn: func() {
				mock.ExpectExec(`INSERT INTO messages`).
					WithArgs("msg-123", "general", "testuser", "Hello, world!", now).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			message: &Message{
				ID:        "msg-error",
				Room:      "general",
				Author:    "testuser",
				Body:      "Error message",
				CreatedAt: now,
			},
			mockFn: func() {
				mock.ExpectExec(`INSERT INTO messages`).
					WithArgs("msg-error", "general", "testuser", "Error message", now).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Save(context.Background(), tt.message)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
