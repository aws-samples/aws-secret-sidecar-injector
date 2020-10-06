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

func TestPathtWith(t *testing.T) {
	withoutMount := "arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF"
	withMount := withoutMount + ":/var/my-secret"
	secretArn, mount, err := GetArnAndMountPath(withoutMount)
	if err != nil {
		t.Error(err)
	}
	if secretArn != withoutMount {
		t.Errorf("secret arn is wrong when annotation does not define mount path: %s", secretArn)
	}
	if mount != "/secrets/database-password-hlRvvF" {
		t.Errorf("secret mount path when annotation does not define mount path: %s", mount)
	}

	secretArn, mount, err = GetArnAndMountPath(withMount)
	if err != nil {
		t.Error(err)
	}
	if secretArn != withoutMount {
		t.Errorf("secret arn is wrong when annotation defines mount path: %s", secretArn)
	}
	if mount != "/var/my-secret" {
		t.Errorf("secret mount path when annotation defines mount path: %s", mount)
	}
}
