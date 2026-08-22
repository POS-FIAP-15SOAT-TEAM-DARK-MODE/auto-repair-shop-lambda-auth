package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserIDByCPF_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT c.user_id FROM customer c WHERE c.cpf = \$1`).
		WithArgs("52998224725").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-abc"))

	repo := NewCustomerRepository(db)
	userID, err := repo.UserIDByCPF(context.Background(), "52998224725")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-abc" {
		t.Errorf("userID = %q, want %q", userID, "user-abc")
	}
}

func TestUserIDByCPF_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT c.user_id FROM customer c WHERE c.cpf = \$1`).
		WithArgs("00000000000").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

	repo := NewCustomerRepository(db)
	_, err = repo.UserIDByCPF(context.Background(), "00000000000")
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Errorf("err = %v, want ErrCustomerNotFound", err)
	}
}

func TestRolesByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT r.name FROM user_role ur JOIN role r ON r.id = ur.role_id WHERE ur.user_id = \$1`).
		WithArgs("user-abc").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("CUSTOMER"))

	repo := NewCustomerRepository(db)
	roles, err := repo.RolesByUserID(context.Background(), "user-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 || roles[0] != "CUSTOMER" {
		t.Errorf("roles = %v, want [CUSTOMER]", roles)
	}
}
