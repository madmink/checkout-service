package repository

import (
	"context"
	"database/sql"
)

type DBExecutor interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
