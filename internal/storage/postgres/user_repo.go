package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db} }

type User struct {
	ID       int64  `db:"id"`
	Username string `db:"username"`
	Password string `db:"password"`
}

func (r *UserRepo) Create(ctx context.Context, u *User) error {
	return r.db.GetContext(ctx, &u.ID,
		`INSERT INTO users (username, password) VALUES ($1,$2) RETURNING id`,
		u.Username, u.Password,
	)
}
func (r *UserRepo) GetByUsername(ctx context.Context, uname string) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, username, password FROM users WHERE username=$1`, uname,
	)
	return &u, err
}
