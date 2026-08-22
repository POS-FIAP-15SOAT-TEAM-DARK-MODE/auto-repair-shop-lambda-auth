// cmd/lambda is the entrypoint for the customer-login lambda: given a CPF,
// resolve it to a real user_id/roles via a minimal read against the shared
// Postgres database, then issue a JWT the main app's existing auth
// middleware accepts unmodified. See README.md for the full request/response
// shape and the deliberate scope limits (no CPF format validation, no
// customer "status" check — see infra-k8s issue #4 for why).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"

	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/authtoken"
	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/repository"
	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/secrets"
)

type loginRequest struct {
	CPF string `json:"cpf"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

type errorResponse struct {
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

var (
	db           *sql.DB
	customerRepo *repository.CustomerRepository
	jwtSecret    []byte
	jwtExpiry    time.Duration
)

func init() {
	ctx := context.Background()

	secret, err := secrets.Load(ctx, mustEnv("APP_SECRET_ID"))
	if err != nil {
		log.Fatalf("failed to load app secret: %v", err)
	}
	jwtSecret = []byte(secret.JWTSecret)

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		mustEnv("DB_HOST"), envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "postgres"), secret.PostgresPassword,
		envOrDefault("DB_NAME", "autorepairshop"),
	)

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db connection: %v", err)
	}
	customerRepo = repository.NewCustomerRepository(db)

	jwtExpiry, err = time.ParseDuration(envOrDefault("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		log.Fatalf("invalid JWT_EXPIRES_IN: %v", err)
	}
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body loginRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.CPF == "" {
		return jsonResponse(400, errorResponse{Code: 400, Errors: []string{"cpf is required"}})
	}

	userID, err := customerRepo.UserIDByCPF(ctx, body.CPF)
	if errors.Is(err, repository.ErrCustomerNotFound) {
		return jsonResponse(404, errorResponse{Code: 404, Errors: []string{"customer not found"}})
	}
	if err != nil {
		log.Printf("UserIDByCPF error: %v", err)
		return jsonResponse(500, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	roles, err := customerRepo.RolesByUserID(ctx, userID)
	if err != nil {
		log.Printf("RolesByUserID error: %v", err)
		return jsonResponse(500, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	expiresAt := time.Now().Add(jwtExpiry)
	token, err := authtoken.GenerateToken(jwtSecret, userID, roles, expiresAt)
	if err != nil {
		log.Printf("GenerateToken error: %v", err)
		return jsonResponse(500, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	return jsonResponse(200, loginResponse{Token: token, ExpiresIn: int64(jwtExpiry.Seconds())})
}

func jsonResponse(status int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	lambda.Start(handler)
}
