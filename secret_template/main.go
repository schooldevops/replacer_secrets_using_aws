package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"gopkg.in/yaml.v2"
)

//	SecretConfig 시크릿 설정 구조체
type SecretConfig struct {
	Profile string `yaml:"profile"`
	Region string	`yaml:"region"`
	Secrets string `yaml:"secrets"`
	SecretKeys map[string]string `yaml:"secretkeys,omitempty"`
}

var secretConfig = SecretConfig{}

func main() {
	yamlFile := readFile("secretConfig.yml")
	err := yaml.Unmarshal([]byte(yamlFile), &secretConfig)
	if err != nil {
			log.Fatalf("Unmarshal: %v", err)
	}

	err, keyValueMap := getSecret()
	if err != nil {
		log.Fatal(err);
	}

	log.Println(keyValueMap)


	mappedSecretMap := keyMapping(keyValueMap, secretConfig.SecretKeys)
	confFile := readFile("application.yml")
	log.Println(makingTemplate(string(confFile), mappedSecretMap))

}

func readFile(filename string) string {
	confFile, err := ioutil.ReadFile(filename)
	if err != nil {
			log.Printf("secretConfig read err   #%v ", err)
	}

	return string(confFile)
}

func keyMapping(secretMap map[string]interface{}, configMap map[string]string) map[string]interface{} {
	keyValueMap := make(map[string]interface{})

	for key, value := range configMap {
		keyValueMap[key] = secretMap[value]

		fmt.Println(key, value)
	}

	return keyValueMap
}

func makingTemplate(a string, b map[string]interface{}) string {
	tmpl := template.Must(template.New("").Parse(a))
	buf := &bytes.Buffer{}
	err := tmpl.Execute(buf, b)
	if err != nil {
		panic(err)
	}
	s := buf.String()
	return s
}

// getSecret() is get secret from aws secretManager
func getSecret() (error, map[string]interface{}) {

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
		return nil, keyValueMap;
	}
}