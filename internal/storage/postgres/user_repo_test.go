package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewUserRepo(sqlxDB)

	assert.NotNil(t, repo)
	assert.Equal(t, sqlxDB, repo.db)
}

func TestUserRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewUserRepo(sqlxDB)

	tests := []struct {
		name    string
		user    *User
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful user creation",
			user: &User{
				Username: "testuser",
				Password: "hashedpassword",
			},
			mockFn: func() {
				mock.ExpectQuery(`INSERT INTO users \(username, password\) VALUES \(\$1,\$2\) RETURNING id`).
					WithArgs("testuser", "hashedpassword").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name: "duplicate username error",
			user: &User{
				Username: "existinguser",
				Password: "hashedpassword",
			},
			mockFn: func() {
				mock.ExpectQuery(`INSERT INTO users \(username, password\) VALUES \(\$1,\$2\) RETURNING id`).
					WithArgs("existinguser", "hashedpassword").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
		{
			name: "empty username",
			user: &User{
				Username: "",
				Password: "hashedpassword",
			},
			mockFn: func() {
				mock.ExpectQuery(`INSERT INTO users \(username, password\) VALUES \(\$1,\$2\) RETURNING id`).
					WithArgs("", "hashedpassword").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Create(context.Background(), tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.user.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewUserRepo(sqlxDB)

	tests := []struct {
		name     string
		username string
		mockFn   func()
		wantUser *User
		wantErr  bool
	}{
		{
			name:     "user found",
			username: "testuser",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "password"}).
					AddRow(1, "testuser", "hashedpassword")
				mock.ExpectQuery(`SELECT id, username, password FROM users WHERE username=\$1`).
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			wantUser: &User{
				ID:       1,
				Username: "testuser",
				Password: "hashedpassword",
			},
			wantErr: false,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			mockFn: func() {
				mock.ExpectQuery(`SELECT id, username, password FROM users WHERE username=\$1`).
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			wantUser: nil,
			wantErr:  true,
		},
		{
			name:     "database error",
			username: "testuser",
			mockFn: func() {
				mock.ExpectQuery(`SELECT id, username, password FROM users WHERE username=\$1`).
					WithArgs("testuser").
					WillReturnError(sql.ErrConnDone)
			},
			wantUser: nil,
			wantErr:  true,
		},
		{
			name:     "empty username",
			username: "",
			mockFn: func() {
				mock.ExpectQuery(`SELECT id, username, password FROM users WHERE username=\$1`).
					WithArgs("").
					WillReturnError(sql.ErrNoRows)
			},
			wantUser: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			user, err := repo.GetByUsername(context.Background(), tt.username)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.wantUser.ID, user.ID)
				assert.Equal(t, tt.wantUser.Username, user.Username)
				assert.Equal(t, tt.wantUser.Password, user.Password)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_CreateAndGet_Integration(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewUserRepo(sqlxDB)

	// Mock user creation
	mock.ExpectQuery(`INSERT INTO users \(username, password\) VALUES \(\$1,\$2\) RETURNING id`).
		WithArgs("integrationuser", "hashedpass").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	// Mock user retrieval
	rows := sqlmock.NewRows([]string{"id", "username", "password"}).
		AddRow(42, "integrationuser", "hashedpass")
	mock.ExpectQuery(`SELECT id, username, password FROM users WHERE username=\$1`).
		WithArgs("integrationuser").
		WillReturnRows(rows)

	// Create user
	user := &User{
		Username: "integrationuser",
		Password: "hashedpass",
	}
	err = repo.Create(context.Background(), user)
	require.NoError(t, err)
	assert.Equal(t, int64(42), user.ID)

	// Retrieve user
	retrievedUser, err := repo.GetByUsername(context.Background(), "integrationuser")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Password, retrievedUser.Password)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_ContextCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewUserRepo(sqlxDB)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock should not be called due to context cancellation
	user := &User{
		Username: "testuser",
		Password: "hashedpass",
	}

	err = repo.Create(ctx, user)
	assert.Error(t, err)

	_, err = repo.GetByUsername(ctx, "testuser")
	assert.Error(t, err)
}
