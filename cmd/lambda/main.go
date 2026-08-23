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
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/authtoken"
	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/logging"
	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/repository"
	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/requestid"
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
	log          *zap.Logger
	db           *sql.DB
	customerRepo *repository.CustomerRepository
	jwtSecret    []byte
	jwtExpiry    time.Duration
)

func init() {
	log = logging.New()
	ctx := context.Background()

	secret, err := secrets.Load(ctx, mustEnv("APP_SECRET_ID"))
	if err != nil {
		log.Fatal("failed to load app secret", zap.Error(err))
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
		log.Fatal("failed to open db connection", zap.Error(err))
	}
	customerRepo = repository.NewCustomerRepository(db)

	jwtExpiry, err = time.ParseDuration(envOrDefault("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		log.Fatal("invalid JWT_EXPIRES_IN", zap.Error(err))
	}
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	reqID := requestid.From(req.Headers, req.RequestContext.RequestID)
	reqLog := log.With(zap.String("request_id", reqID))

	var body loginRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.CPF == "" {
		reqLog.Warn("customer_login.validation_failed", zap.String("reason", "missing or malformed cpf"))
		return jsonResponse(400, reqID, errorResponse{Code: 400, Errors: []string{"cpf is required"}})
	}

	// CPF is PII — never logged, per the app's redaction convention. Only
	// non-sensitive derived values (user_id, roles) appear in log fields.
	userID, err := customerRepo.UserIDByCPF(ctx, body.CPF)
	if errors.Is(err, repository.ErrCustomerNotFound) {
		reqLog.Info("customer_login.not_found")
		return jsonResponse(404, reqID, errorResponse{Code: 404, Errors: []string{"customer not found"}})
	}
	if err != nil {
		reqLog.Error("customer_login.lookup_failed", zap.Error(err))
		return jsonResponse(500, reqID, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	roles, err := customerRepo.RolesByUserID(ctx, userID)
	if err != nil {
		reqLog.Error("customer_login.roles_lookup_failed", zap.String("user_id", userID), zap.Error(err))
		return jsonResponse(500, reqID, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	expiresAt := time.Now().Add(jwtExpiry)
	token, err := authtoken.GenerateToken(jwtSecret, userID, roles, expiresAt)
	if err != nil {
		reqLog.Error("customer_login.token_generation_failed", zap.String("user_id", userID), zap.Error(err))
		return jsonResponse(500, reqID, errorResponse{Code: 500, Errors: []string{"internal error"}})
	}

	reqLog.Info("customer_login.token_issued", zap.String("user_id", userID), zap.Strings("roles", roles))
	return jsonResponse(200, reqID, loginResponse{Token: token, ExpiresIn: int64(jwtExpiry.Seconds())})
}

func jsonResponse(status int, reqID string, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Request-Id": reqID,
		},
		Body: string(b),
	}, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal("missing required env var", zap.String("key", key))
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
	defer log.Sync() //nolint:errcheck
	lambda.Start(handler)
}
