package storage

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Storage struct {
	DB *sqlx.DB
}

type Agent struct {
	ID     uuid.UUID `db:"id"`
	Status string    `db:"status"`
}

type Task struct {
	ID         uuid.UUID `db:"id"`
	Expression string    `db:"expression"`
	Status     string    `db:"status"`
}
