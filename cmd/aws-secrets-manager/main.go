package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func main() {
	secretArn := os.Getenv("SECRET_ARN")
	secretFilename := os.Getenv("SECRET_FILENAME")
	mountPath := os.Getenv("MOUNT_PATH")
	var AWSRegion string

	if arn.IsARN(secretArn) {
		arnobj, _ := arn.Parse(secretArn)
		AWSRegion = arnobj.Region
	} else {
		log.Println("Not a valid ARN")
		os.Exit(1)
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Panic(err)
	}
	svc := secretsmanager.New(sess, &aws.Config{
		Region: aws.String(AWSRegion),
	})

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretArn),
		VersionStage: aws.String("AWSCURRENT"),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeInvalidRequestException:
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
			case secretsmanager.ErrCodeDecryptionFailure:
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
			case secretsmanager.ErrCodeInternalServiceError:
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return
	}
	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	if result.SecretString != nil {
		secretString = *result.SecretString
		writeOutput(secretString, mountPath, secretFilename)
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			log.Println("Base64 Decode Error:", err)
			return
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])
		writeOutput(decodedBinarySecret, mountPath, secretFilename)
	}
}
func writeOutput(output string, path string, name string) error {
	if path == "" {
		path = "/tmp"
	}
	if name == "" {
		name = "secret"
	}
	if filepath.IsAbs(filepath.Join(path, name)) {
		f, err := os.Create(filepath.Join(path, name))
		if err != nil {
			return fmt.Errorf("error creating file, %w", err)
		}
		defer f.Close()
		f.WriteString(output)
		return nil
	}
	return fmt.Errorf("not a valid file path")
}
