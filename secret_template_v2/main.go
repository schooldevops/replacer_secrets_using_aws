package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"strings"

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

//	SecretConfig 시크릿 설정 구조체
type SecretConfig struct {
	Profile      string            `yaml:"profile"`
	Ext          string            `yaml:"ext"`
	TargetPath   string            `yaml:"targetPath"`
	Region       string            `yaml:"region"`
	Secrets      string            `yaml:"secrets"`
	Environments []string          `yaml:"environments"`
	SecretKeys   map[string]string `yaml:"secretkeys,omitempty"`
}

var secretConfig = SecretConfig{}

var myLogger *log.Logger

func main() {
	// 로그파일 오픈
	fpLog, err := os.OpenFile("logfile.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer fpLog.Close()
	myLogger = log.New(fpLog, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	myLogger.Println("------- Start Replacing Secrets. -------")
	yamlFile := readFile("secretConfig.yml")
	err = yaml.Unmarshal([]byte(yamlFile), &secretConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v\n", err)
	}

	myLogger.Println("INFO Read secretConfig.yml")

	//	환경 변수를 돌면서, 값을 조회하고 처리한다.
	for _, value := range secretConfig.Environments {
		err, targetFile := makeTargetFile(value, secretConfig.Ext, secretConfig.TargetPath)
		myLogger.Printf("INFO MakeTargetFile [%s, %s %s] \n", secretConfig.TargetPath, value, secretConfig.Ext, err)
		if err == nil {
			result := replaceConfigFiles(&secretConfig, targetFile)
			myLogger.Println("INFO Replace result is ", result)
		} else {
			myLogger.Fatalf("ERROR File is not exists [%s, %s]\n", value, secretConfig.Ext)
		}
	}

	myLogger.Println("------- Done Replacing Secrets. -------")
}

func makeTargetFile(environment string, ext string, targetPath string) (error, string) {
	var targetFile string

	prefix := "application"
	// fmt.Println("env: ", environment, " ext: ", ext)
	if environment == "default" {
		targetFile = fmt.Sprintf("%s%s.%s", targetPath, prefix, ext)
	} else {
		targetFile = fmt.Sprintf("%s%s-%s.%s", targetPath, prefix, environment, ext)
	}

	if err := Exists(targetFile); err == nil {
		return nil, targetFile
	} else {
		return err, ""
	}
}

func Exists(name string) error {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func replaceConfigFiles(secretConfig *SecretConfig, targetFile string) bool {
	myLogger.Println("INFO Process targetFile: ", targetFile)

	err, keyValueMap := getSecret()
	if err != nil {
		log.Fatal(err)
	}
	myLogger.Println("INFO Success reading secrets from AWS SecretsManager.")

	mappedSecretMap := keyMapping(keyValueMap, secretConfig.SecretKeys)
	myLogger.Println("INFO Parsed SecretMap by SecretKeys [%v]", secretConfig.SecretKeys)

	resultByte := makingTemplate(targetFile, mappedSecretMap)
	log.Println("result \n", resultByte.String())
	myLogger.Println("INFO Done replacing config file what is %s", targetFile)
	return true
}

func readFile(filename string) string {
	confFile, err := ioutil.ReadFile(filename)
	if err != nil {
		myLogger.Fatalf("ERROR secretConfig read err   #%v \n", err)
	}

	return string(confFile)
}

func keyMapping(secretMap map[string]interface{}, configMap map[string]string) map[string]interface{} {
	keyValueMap := make(map[string]interface{})

	for key, value := range configMap {
		keyValueMap[key] = secretMap[value]

		// fmt.Println(key, value)
	}

	return keyValueMap
}

func makingTemplate(a string, b map[string]interface{}) bytes.Buffer {

	var bt bytes.Buffer

	file, err := os.Open(a)
	defer file.Close()

	if err != nil {
		fmt.Println(err)
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			m := regexp.MustCompile("\\$\\{(.*?)\\}")
			res := m.FindAllStringSubmatch(line, 1)

			if len(res) > 0 {
				for i := range res {
					key := strings.Split(res[i][1], ":")

					if b[key[0]] != "" {
						bt.WriteString(m.ReplaceAllLiteralString(line, b[key[0]].(string)))
						bt.WriteString("\n")
					}
				}
			} else {
				bt.WriteString(line)
				bt.WriteString("\n")
			}
		}
	}

	return bt
}

// getSecret() is get secret from aws secretManager
func getSecret() (error, map[string]interface{}) {

	myLogger.Println("INFO Load Secrets from AWS SecretsManager.")
	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(secretConfig.Region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretConfig.Secrets),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
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
