package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// DefaultRegion is default aws region
const DefaultRegion = "ap-northeast-2"

// DefaultAwsCredentialPath is your aws credential path
const DefaultAwsCredentialPath = "~/.aws"

// DefaultSecretName is default SecretName
const DefaultSecretName = "testprj/userinfo/key"

// DefaultProfile is default profile
const DefaultProfile = "default"

var secretName *string
var region *string
var keyValue *string
var profile *string

var myLogger *log.Logger

func main() {
	// 로그파일 오픈
	fpLog, err := os.OpenFile("logfile.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer fpLog.Close()
	myLogger = log.New(fpLog, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	myLogger.Println("------- Start Set Env Secrets. -------")

	extractParameter()
	err, secret := getSecret()
	if err != nil {
		return
	}

	fmt.Println(secret + "")

	myLogger.Println("------- Done Set Env Secrets. -------")
}

// extract paramegers
func extractParameter() {
	secretName = flag.String("s", DefaultSecretName, "<projectName>/<path1>/<path2>...")
	region = flag.String("r", DefaultRegion, "ap-northeast-2")
	keyValue = flag.String("k", "", "usersecretkey")
	profile = flag.String("p", DefaultProfile, "default")

	flag.Parse()

	myLogger.Printf("INFO Read Parameters are -s[%s], -r[%s], -k[%s], -p[%s]\n", secretName, region, keyValue, profile)
}

// getSecret() is get secret from aws secretManager
func getSecret() (error, string) {

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(*region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(*secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		return err, ""
	}

	myLogger.Println("Success Secret values")

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	var keyValueMap map[string]interface{}

	if result.SecretString != nil {
		myLogger.Println("INFO success reading secretString from aws.")
		secretString = *result.SecretString

		if *keyValue == "" {

			myLogger.Println("ERROR there are any secret key and value on aws.")
			return nil, secretString
		}

		// b, _ := json.Marshal(secretString)
		json.Unmarshal([]byte(secretString), &keyValueMap)
		if keyValueMap[*keyValue] == nil {
			myLogger.Println("ERROR there aren't any mapped secret key and value on aws.")
			return nil, ""
		} else {
			myLogger.Println("INFO success finding secret key and value on aws.")
			return nil, keyValueMap[*keyValue].(string)
		}
	} else {
		myLogger.Println("INFO read secretString from aws.")

		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			// fmt.Println("Base64 Decode Error:", err)
			return err, ""
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])

		if *keyValue == "" {
			myLogger.Println("ERROR [Decoding] there are any secret key and value on aws. ")
			return nil, decodedBinarySecret
		}

		// b, _ := json.Marshal(decodedBinarySecret)
		json.Unmarshal([]byte(decodedBinarySecret), &keyValueMap)
		if keyValueMap[*keyValue] == nil {
			myLogger.Println("ERROR [Decoding] there aren't any mapped secret key and value on aws.")
			return nil, ""
		} else {
			myLogger.Println("INFO [Decoding] success finding secret key and value on aws.")
			return nil, keyValueMap[*keyValue].(string)
		}
	}
}
