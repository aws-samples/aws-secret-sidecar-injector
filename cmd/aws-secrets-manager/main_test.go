package main

import (
	"testing"

	aws "github.com/aws/aws-sdk-go/aws"
	secretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	secretsmanageriface "github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

type mockSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI
}

func (m *mockSecretsManagerClient) GetSecretValue(input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		ARN:          aws.String("aws:foo"),
		SecretString: aws.String("some-value"),
	}, nil
}

func TestGetSecretValue(t *testing.T) {
	mockSvc := &mockSecretsManagerClient{}

	response := getSecretValue(mockSvc, "arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF")
	if response != "some-value" {
		t.Errorf("Vale was incorrect, got: %s, want: value.", response)
	}
}
