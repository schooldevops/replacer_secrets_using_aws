package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"gopkg.in/yaml.v2"
)

// DefaultRegion is default aws region
const DefaultRegion = "ap-northeast-2"

// DefaultAwsCredentialPath is your aws credential path
const DefaultAwsCredentialPath = "~/.aws"

// DefaultSecretName is default SecretName
const DefaultSecretName = ""

// DefaultProfile is default profile
const DefaultProfile = "default"

const DefaultBackDir = "orig/"

const DefaultConfigFile = "secretConfig.yml"

//	SecretConfig 시크릿 설정 구조체
type SecretConfig struct {
	Profile        string            `yaml:"profile"`
	Region         string            `yaml:"region"`
	Secrets        string            `yaml:"secrets"`
	Environments   []string          `yaml:"environments"`
	Context        string            `yaml:"context"`
	Namespace      string            `yaml:"namespace"`
	SecretsName    string            `yaml:"secretsName"`
	SecretKeys     map[string]string `yaml:"secretkeys,omitempty"`
	ConfigMapsName string            `yaml:"configMapsName"`
	ConfigKeys     map[string]string `yaml:"configkeys,omitempty"`
}

// create SecretConfig Instance
var secretConfig = SecretConfig{}

// Define logger
var myLogger *log.Logger

var configFile *string

func main() {
	// 로그파일 오픈
	fpLog, err := os.OpenFile("logfile.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer fpLog.Close()
	myLogger = log.New(fpLog, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	myLogger.Println("------- Start K8S Config and Secrets. -------")

	extractParameter()

	// SecretConfig 읽기
	yamlFile := readFile(*configFile)
	err = yaml.Unmarshal([]byte(yamlFile), &secretConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v\n", err)
	}

	myLogger.Printf("INFO Read %s\n", *configFile)

	//	환경 변수를 돌면서, 값을 조회하고 처리한다.
	for _, value := range secretConfig.Environments {
		// getTargetFile Path
		log.Println("target value: ", value)

		secretsMap, configsMap, err := getSecretsFromAWS(&secretConfig, value)

		if err != nil {
			myLogger.Printf("ERROR Retrive secrets from AWS [%s]\n", value)
		} else {
			fmt.Println("going to processing")
			fmt.Println(secretsMap, configsMap)
		}
	}

	myLogger.Println("------- Done K8S Config and Secrets. -------")
}

// extract paramegers
func extractParameter() {
	configFile = flag.String("f", DefaultConfigFile, "secretConfig.yml")
	flag.Parse()
	myLogger.Printf("INFO Read Parameters are -f[%s]\n", *configFile)
}

// getSecretsFromAWS is replace secrets values from AWS
func getSecretsFromAWS(secretConfig *SecretConfig, targetEnv string) (map[string]interface{}, map[string]interface{}, error) {
	myLogger.Println("INFO Process getSecretsFromAWS: ", targetEnv)

	err, keyValueMap := getSecret(targetEnv)
	if err != nil {
		myLogger.Println("ERROR reading secrets from AWS ", err)
		log.Fatal(err)
		return nil, nil, err
	}
	myLogger.Println("INFO Success reading secrets from AWS SecretsManager.")

	// create mapping for replacing secrets
	mappedSecretMap := keyMapping(keyValueMap, secretConfig.SecretKeys)
	myLogger.Printf("INFO Parsed SecretMap by SecretKeys [%v]\n", secretConfig.SecretKeys)

	// create mapping for replacing secrets
	mappedConfigMap := keyMapping(keyValueMap, secretConfig.ConfigKeys)
	myLogger.Printf("INFO Parsed SecretMap by SecretKeys [%v]\n", secretConfig.ConfigKeys)
	return mappedSecretMap, mappedConfigMap, nil
}

// keyMapping is mapping from secret key to config key placeholder
func keyMapping(secretMap map[string]interface{}, configMap map[string]string) map[string]interface{} {
	keyValueMap := make(map[string]interface{})

	for key, value := range configMap {
		keyValueMap[key] = secretMap[value]
	}

	return keyValueMap
}

// getSecret() is get secret from aws secretManager
func getSecret(targetEnv string) (error, map[string]interface{}) {

	targetSecrets := fmt.Sprintf("%s/%s", secretConfig.Secrets, targetEnv)
	myLogger.Println("INFO Load Secrets from AWS SecretsManager. from: ", targetSecrets)
	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(secretConfig.Region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(targetSecrets),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		fmt.Println("error: ", err)
		return err, nil
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	var keyValueMap map[string]interface{}

	if result.SecretString != nil {
		secretString = *result.SecretString
		json.Unmarshal([]byte(secretString), &keyValueMap)

		return nil, keyValueMap
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			// fmt.Println("Base64 Decode Error:", err)
			return err, nil
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])

		// b, _ := json.Marshal(decodedBinarySecret)
		json.Unmarshal([]byte(decodedBinarySecret), &keyValueMap)
		return nil, keyValueMap
	}
}

// readFile is read a config file from path
func readFile(filename string) string {
	confFile, err := ioutil.ReadFile(filename)
	if err != nil {
		myLogger.Fatalf("ERROR secretConfig read err %v \n", err)
	}

	return string(confFile)
}
