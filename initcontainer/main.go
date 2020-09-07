package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

func main() {
	secretsArn := os.Getenv("SECRET_ARN")

	arns := strings.Split(secretsArn, ",")
	for i := range arns {
		var AWSRegion string

		secretArn, mountPath, err := GetArnAndMountPath(arns[i])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		arnobj, _ := arn.Parse(secretArn)
		AWSRegion = arnobj.Region
		sess, _ := session.NewSession(&aws.Config{
			Region:              aws.String(AWSRegion),
			STSRegionalEndpoint: endpoints.RegionalSTSEndpoint,
		})
		svc := secretsmanager.New(sess, &aws.Config{
			Region: aws.String(AWSRegion),
		})

		result := getSecretValue(svc, secretArn)
		if len(result) == 0 {
			fmt.Println("No value for secret arn: ", secretArn)
			os.Exit(1) // Fail if secret value not found
		}
		writeOutput(result, mountPath)
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

func writeOutput(output string, mountPath string) {
	// Write inside /tmp, the volume mounted by the init-container
	mountPath = filepath.Join("/tmp", mountPath)
	// Create Secrets Directory
	dir := filepath.Dir(mountPath)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dir, 0755)
		if errDir != nil {
			fmt.Println(err)
		}
	}
	// Insert the secret into the filename
	f, err := os.Create(mountPath)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(output)
}

//arn:aws:secretsmanager:us-east-1:123456789012:secret:database-password-hlRvvF:/var/log/secret
func GetArnAndMountPath(secretStr string) (secretArn string, mountPath string, err error) {
	// regex parse to extract arn and mount path
	var re = regexp.MustCompile(`(?m)(.*:secret:[^:]*):?(.*)`)
	match := re.FindStringSubmatch(secretStr)
	if len(match) == 3 {
		//<secret-arn>:<mounting-path>
		secretArn = match[1]
		mountPath = match[2]
	} else {
		secretArn = secretStr
	}

	if !arn.IsARN(secretArn) {
		return "", "", fmt.Errorf("Not a valid ARN: %s", secretArn)
	}
	arnobj, err := arn.Parse(secretArn)
	if err != nil {
		return "", "", fmt.Errorf("Can not parse arn: %s %v \n", secretArn, err)
	}
	if mountPath == "" {
		secretName := strings.Split(arnobj.Resource, "secret:")[1]
		mountPath = filepath.Join("/secrets", secretName)
	}
	return secretArn, mountPath, nil
}
