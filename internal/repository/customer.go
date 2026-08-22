// Package repository holds the two read-only queries this lambda needs
// against the shared auto-repair-shop Postgres database: resolving a CPF to
// its linked user_id, then that user's roles. Mirrors the query shapes in
// auto-repair-shop's internal/customer/repository and internal/auth/repository
// (customer.cpf / user_role+role join) — no writes, no other tables touched.
package repository

import (
	"context"
	"database/sql"
	"errors"
)

var ErrCustomerNotFound = errors.New("customer not found for given cpf")

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) UserIDByCPF(ctx context.Context, cpf string) (string, error) {
	const query = `SELECT c.user_id FROM customer c WHERE c.cpf = $1`

	var userID string
	err := r.db.QueryRowContext(ctx, query, cpf).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrCustomerNotFound
	}
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (r *CustomerRepository) RolesByUserID(ctx context.Context, userID string) ([]string, error) {
	const query = `SELECT r.name FROM user_role ur JOIN role r ON r.id = ur.role_id WHERE ur.user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}
