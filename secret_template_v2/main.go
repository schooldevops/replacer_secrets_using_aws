package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
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

const DefaultBackDir = "orig/"

const DefaultConfigFile = "secretConfig.yml"

//	SecretConfig 시크릿 설정 구조체
type SecretConfig struct {
	Profile          string            `yaml:"profile"`
	ConfigFilePrefix string            `yaml:"configFilePrefix"`
	Ext              string            `yaml:"ext"`
	TargetPath       string            `yaml:"targetPath"`
	Region           string            `yaml:"region"`
	Secrets          string            `yaml:"secrets"`
	Environments     []string          `yaml:"environments"`
	SecretKeys       map[string]string `yaml:"secretkeys,omitempty"`
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

	myLogger.Println("------- Start Replacing Secrets. -------")

	extractParameter()

	// SecretConfig 읽기
	yamlFile := readFile(*configFile)
	err = yaml.Unmarshal([]byte(yamlFile), &secretConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v\n", err)
	}

	myLogger.Println("INFO Read %s", *configFile)

	//	환경 변수를 돌면서, 값을 조회하고 처리한다.
	for _, value := range secretConfig.Environments {
		// getTargetFile Path
		log.Println("target value: ", value)
		err, targetFile, destFile := makeTargetFile(value, secretConfig.ConfigFilePrefix, secretConfig.Ext, secretConfig.TargetPath)
		myLogger.Printf("INFO MakeTargetFile [%s, %s %s] \n", secretConfig.TargetPath, value, secretConfig.Ext, err)

		// process Replace Config File if not exists error
		if err == nil {
			result := replaceConfigFiles(&secretConfig, targetFile, destFile)
			myLogger.Println("INFO Replace result is ", result)
		} else {
			myLogger.Printf("ERROR File is not exists [%s, %s]\n", value, secretConfig.Ext)
		}
	}

	myLogger.Println("------- Done Replacing Secrets. -------")
}

// extract paramegers
func extractParameter() {
	configFile = flag.String("f", DefaultConfigFile, "secretConfig.yml")

	flag.Parse()

	myLogger.Printf("INFO Read Parameters are -f[%s]\n", *configFile)
}

// makeTargetFile is make target file for reading application config
// It returns sourceFile and backFile paths
func makeTargetFile(environment string, configFilePrefix string, ext string, targetPath string) (error, string, string) {
	var targetFile string
	var destFile string

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recover Exception ", err)
		}
	}()

	destDir := fmt.Sprintf("%s%s", targetPath, DefaultBackDir)

	// make target file
	if environment == "default" {
		if ext == "" {
			targetFile = fmt.Sprintf("%s%s", targetPath, configFilePrefix)
			destFile = fmt.Sprintf("%s%s%s", targetPath, DefaultBackDir, configFilePrefix)
		} else {
			targetFile = fmt.Sprintf("%s%s.%s", targetPath, configFilePrefix, ext)
			destFile = fmt.Sprintf("%s%s%s.%s", targetPath, DefaultBackDir, configFilePrefix, ext)
		}
	} else {
		if ext == "" {
			targetFile = fmt.Sprintf("%s%s-%s", targetPath, configFilePrefix, environment)
			destFile = fmt.Sprintf("%s%s%s-%s", targetPath, DefaultBackDir, configFilePrefix, environment)
		} else {
			targetFile = fmt.Sprintf("%s%s-%s.%s", targetPath, configFilePrefix, environment, ext)
			destFile = fmt.Sprintf("%s%s%s-%s.%s", targetPath, DefaultBackDir, configFilePrefix, environment, ext)
		}
	}

	// Create Directory for saving original config file
	if err := makeDestDirectory(destDir); err != nil {
		log.Println(err)
		return err, "", ""
	}

	// Check for existing a source file
	if err := Exists(targetFile); err == nil {
		return nil, targetFile, destFile
	} else {
		return err, "", ""
	}
}

// makeDestDirectory is creating destination path when it isn't exists
func makeDestDirectory(destDir string) error {
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err := os.Mkdir(destDir, 0755)
		return err
	}

	return nil
}

// Exists is check file is existsing.
func Exists(name string) error {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// replaceConfigFiles is replace secrets values to target placehold in application.yml
func replaceConfigFiles(secretConfig *SecretConfig, targetFile string, destFile string) bool {
	myLogger.Println("INFO Process targetFile: ", targetFile)

	err, keyValueMap := getSecret()
	if err != nil {
		log.Fatal(err)
	}
	myLogger.Println("INFO Success reading secrets from AWS SecretsManager.")

	// create mapping for replacing secrets
	mappedSecretMap := keyMapping(keyValueMap, secretConfig.SecretKeys)
	myLogger.Println("INFO Parsed SecretMap by SecretKeys [%v]", secretConfig.SecretKeys)

	err, resultByte := makingTemplate(targetFile, mappedSecretMap)

	if err != nil {
		myLogger.Fatalf("Error Fail to make a Template ", err)
	}

	// move original config file to backup directory
	err = moveOriginFile(targetFile, destFile)
	if err != nil {
		log.Fatal(err)
	}

	// create new config file it had replaced config placeholders
	writeFile(targetFile, resultByte)

	// log.Println("result \n", resultByte)
	myLogger.Println("INFO Done replacing config file what is %s", targetFile)
	return true
}

// writeFile is write new config value to a new file
func writeFile(targetFile string, result string) {
	err := ioutil.WriteFile(targetFile, []byte(result), 0755)
	if err != nil {
		myLogger.Fatalf("ERROR write replaced file error %s\n", targetFile)
	}
}

// moveOriginFile is backup original config files
func moveOriginFile(origFile string, destFile string) error {
	err := os.Rename(origFile, destFile)
	if err != nil {
		log.Fatal(err)
		myLogger.Fatalf("ERROR move file error\n")
		return err
	}

	myLogger.Printf("INFO move from origFile [%s] to destFile [%s]. \n", origFile, destFile)
	return nil
}

// readFile is read a config file from path
func readFile(filename string) string {
	confFile, err := ioutil.ReadFile(filename)
	if err != nil {
		myLogger.Fatalf("ERROR secretConfig read err %v \n", err)
	}

	return string(confFile)
}

// keyMapping is mapping from secret key to config key placeholder
func keyMapping(secretMap map[string]interface{}, configMap map[string]string) map[string]interface{} {
	keyValueMap := make(map[string]interface{})

	for key, value := range configMap {
		keyValueMap[key] = secretMap[value]
	}

	return keyValueMap
}

// makingTemplate is replacing secret values
func makingTemplate(a string, b map[string]interface{}) (error, string) {

	myLogger.Printf("INFO MakingTemplate targetFile is [%s]\n", a)
	myLogger.Printf("INFO All replace keys are %v\n", reflect.ValueOf(b).MapKeys())
	var bt bytes.Buffer

	file, err := os.Open(a)
	defer file.Close()

	if err != nil {
		fmt.Println(err)
		return err, ""
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			m := regexp.MustCompile("\\$\\{(.*?)\\}")
			res := m.FindAllStringSubmatch(line, 1)

			if len(res) > 0 {
				for i := range res {
					key := strings.Split(res[i][1], ":")

					if b[key[0]] != "" && b[key[0]] != nil {
						bt.WriteString(m.ReplaceAllLiteralString(line, b[key[0]].(string)))
						bt.WriteString("\n")
						myLogger.Printf("INFO Replacing done [%s].\n", key[0])
					} else {
						bt.WriteString(line)
						bt.WriteString("\n")
						myLogger.Printf("INFO Key[%s] is not replacing target.", key[0])
					}
				}
			} else {
				bt.WriteString(line)
				bt.WriteString("\n")
			}
		}
	}

	return nil, bt.String()
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
