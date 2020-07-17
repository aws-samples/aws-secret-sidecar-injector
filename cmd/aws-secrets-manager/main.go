package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

func main() {
	secretArn := os.Getenv("SECRET_ARN")

	arns := strings.Split(secretArn, ",")
	for i := range arns {
		var AWSRegion string

		if arn.IsARN(secretArn) {
			arnobj, _ := arn.Parse(secretArn)
			AWSRegion = arnobj.Region
		} else {
			fmt.Println("ARN Provided: ", secretArn)
			fmt.Println("Not a valid ARN")
			os.Exit(1)
		}

		sess, _ := session.NewSession()
		svc := secretsmanager.New(sess, &aws.Config{
			Region: aws.String(AWSRegion),
		})

		result := getSecretValue(svc, arns[i])
		if len(result) != 0 {
			arnobj, _ := arn.Parse(arns[i])
			secretName := strings.Split(arnobj.Resource, "secret:")[1]
			writeOutput(result, secretName)
		}
	}
}

func getSecretValue(svc secretsmanageriface.SecretsManagerAPI, secretArn string) string {
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
			fmt.Println(err.Error())
		}
		return ""
	}
	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	if result.SecretString != nil {
		return *result.SecretString
	}

	decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
	len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
	if err != nil {
		fmt.Println("Base64 Decode Error:", err)
		return ""
	}
	return string(decodedBinarySecretBytes[:len])

}

func writeOutput(output string, secretName string) {
	// Create Secrets Directory
	_, err := os.Stat("/tmp/secrets")
	if os.IsNotExist(err) {
		errDir := os.MkdirAll("/tmp/secrets", 0755)
		if errDir != nil {
			fmt.Println(err)
		}
	}
	// Insert the secret into the filename
	filename := fmt.Sprintf("/tmp/secrets/%s", secretName)
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(output)
}
