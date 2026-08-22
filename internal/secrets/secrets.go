// Package secrets fetches the app secret (JWT_SECRET, POSTGRES_PASSWORD)
// from the Secrets Manager entry created by auto-repair-shop-infra-db, so
// this lambda signs tokens with the exact same key the main app validates
// against — no secret is duplicated into Terraform state or env vars here.
package secrets

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type AppSecret struct {
	PostgresPassword string `json:"POSTGRES_PASSWORD"`
	JWTSecret        string `json:"JWT_SECRET"`
}

func Load(ctx context.Context, secretID string) (*AppSecret, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := secretsmanager.NewFromConfig(cfg)
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return nil, err
	}

	var secret AppSecret
	if err := json.Unmarshal([]byte(*out.SecretString), &secret); err != nil {
		return nil, err
	}

	return &secret, nil
}
