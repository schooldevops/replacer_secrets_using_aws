# AWS Secret 조회 모듈

# 들어가기

프로그램을 개발하다보면, 단일 프로그램만이 수행되는 경우는 없다. 

- 데이터베이스에 접속한다. 
- 연관된 API 서버에 접속하여 데이터를 조회하거나 정보를 전송한다. 
- 메시지 큐를 이용한다. 

이러한 작업들은 모두 해당 서버에 접근하기 위해서 시크릿 정보가 필요하다. 

데이터베이스는 username/password, 연관 API 는 Token 등이 필요하고, 메시지 큐 역시 id/password 가 필요하다. 

일반적으로 가장 많이 사용하는 방법이 plain text 를 설정파일 application.yaml 이나 자바 코드를 이용하여 저장하여 사용한다. 
그러나 이는 매우 보안에 취약하며, 중요한 서버들의 접근 정보를 타인에게 공유하는 것이나 다름이 없다. 

회사의 보안 망 안에서만 소스코드를 관리한다고 하더라도, 이는 소스에 관련이 없는 사람이 서버의 정보를 봐야할 이유는 없기 때문에 좋은 선택이 아니다. 

## AWS Secret Manager

AWS Secret Manager 은 다양한 방법으로 secret 정보들을 저장할 수 있는 수단을 제공한다. 

우선 어떻게 만드는지 한번 알아보자. 

### AWS Secret Manager 열기 

![secret_manager01](https://user-images.githubusercontent.com/66154381/108319488-ce35cc80-7204-11eb-907b-d069b02efb9b.png)

새 보안 암호 저장 을 선택한다. 

### AWS Secret Manager 

![secret_manager02](https://user-images.githubusercontent.com/66154381/108319550-e86faa80-7204-11eb-93bd-d46731f56e87.png)

- RDS 데이터베이스에 대한 자격 증명: RDS 의 계정/암호를 저장할 수 있으며, 대상 계정에 계정과 암호를 세팅할 수 있다. 
- DocumentDB 데이터베이스에 대한 자격 증명: NoSQL 인 DocumentDB 에 계정/암호를 저장한다. 
- Redshift 클러스터에 대한 자격 증명: 빅데이터 저장소에 대해 계정/암호를 저장한다. 
- 기타 데이터베이스에 대한 자격 증명: 기타 데이터베이스에 대해 계정/암호를 저장한다. 
- 다른 유형의 보안 암호: 키-값 형태로 시크릿을 저장할 수 있게 한다. 

우리는 여기서 `다른 유형의 보안 암호` 를 선택할 것이다. 

그리고 하단에 `보안 암호 키/값` 탭에서 이미지와 같이 입력하자. 

사실 아무 값이나 입력해도 된다. 

### 새 보안 암호 이름 설정 

![secret_manager03](https://user-images.githubusercontent.com/66154381/108319577-f4f40300-7204-11eb-8060-ef2f3627df45.png)

보안 암호 이름: 보안 암호 이름은 디렉토리 구조로 지정한다. 보통 하나의 계정에서 여러 보안 암호를 이용하기 때문에 이런 디렉토리 구조로 팀/프로젝트/대상서버 등의 형태로 작성해 주면 좋다. 

설명, 태그도 같이 입력해 주자.

### 암호 교체 방식 설정 

![secret_manager04](https://user-images.githubusercontent.com/66154381/108319673-16ed8580-7205-11eb-8eb7-52a0953c3b99.png)

자동 교체 구성을 설정한다. 

- 자동 교체 비활성화: 자동 교체 비활성화는 고정된 암호를 사용하는 것이다. 이것은 이미 설정한 암호를 지정하는 경우 사용하며, 프로그램에 시크릿이 고정되어야 할 경우 주로 사용한다. 보통의 케이스는 자동 교체 비활성화를 선택하면 된다. 
- 자동 교체 활성화: 자동교체 활성화는 자동으로 특정 시스템의 암호가 변경되기를 원하는 경우 사용하면 된다. 더욱 강화된 보안을 제공하지만 해당 시스템에 접근할때마다 보안 암호를 가져와서 접속을 해야하는 경우 적합하다. 

### 샘플 코드 보기 

![secret_manager05](https://user-images.githubusercontent.com/66154381/108319712-2371de00-7205-11eb-90e1-db5080ee2afd.png)

보는바와 같이 여러 방법으로 시크릿에 접근할 수 있도록 샘플 코드를 제공하고 있다. 
선호하는 프로그래밍 언어를 선택해서 이를 이용하면 될 것이다. 

### 생성 결과 보기 

![secret_manager06](https://user-images.githubusercontent.com/66154381/108319741-2e2c7300-7205-11eb-8d89-f1a946fe82bf.png)

우리가 이전에 입력한 보안 암호 이름으로 보안 정보를 볼 수 있다. 

해당 보안 내용을 클릭하면 상세 정보를 볼 수 있다. 

![secret_manager07](https://user-images.githubusercontent.com/66154381/108320960-d4c54380-7206-11eb-82ca-d216218767e1.png)

지금까지 보안 암호 설정을 알아 보았다. 

## GO 초기화 

```
go mod init com.schooldevops.go.secret
```

## 필요 모듈 당겨오기

```
go get github.com/aws/aws-sdk-go/service/secretsmanager
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/aws/awserr
go get github.com/aws/aws-sdk-go/aws/session
```

## secret 조회 코드 작성하기. 

### parameter 설계 

우리 프로그램은 다음과 같은 파라미터를 받을 수 있다. 

- `-s`: Secret Name 을 지정한다. aws에서 생성한 secret 이름이면 된다. 
- `-r`: region 지정한다. 기본ㄱ밧은 ap-northeast-2 이다. 
- `-k`: 저장된 secret 의 키를 지정하면 해당 값을 가져오도록 한다. 
- `-p`: 프로파일을 지정한다. aws configure 를 지정할때 --profile 지정한경우 원하는 profile 을 선택할 수 있다. 

### Source Part01 import 

우선 import 부분부터 살펴보자. 

```
package main 
import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)
```

- 단일 프로그램으로 작성할 것이기 때문에 package 를 main 으로 잡아주었다. 
- encoding/base64, encoding/json 이 두 정보는 우리가 받아온 secret 값을 파싱하기 위해 사용한다. 
- flag 는 파라미터를 파싱하는데 사용한다. 
- aws-sdk-go ... 해당 임포트를 이용하여 aws 에 접근할 수 있도록 한다. 

### Source Part02 constant 및 글로벌 변수 

```
// DefaultRegion is default aws region
const DefaultRegion = "ap-northeast-2"

// DefaultAwsCredentialPath is your aws credential path
const DefaultAwsCredentialPath = "~/.aws"

// DefaultSecretName is default SecretName
const DefaultSecretName = "testprj/userinfo/key"

// DefaultProfile is default profile
const DefaultProfile = "default"

var secretName *string  // 시크릿 이름으로 -s로 전달된 값이 저장된다. 
var region *string      // 우리가 접속할 region 설정한다. -r 파라미터에 해당한다. 
var keyValue *string    // 조회할 키 값을 가져온다. 없을경우 해당 전체 시크릿 정보를 json 형태로 반환한다. -k 파라미터에 해당한다. 
var profile *string     // 선택할 profile 값이다. -p 파라미터에 해당한다. 
```

### Source Part03 main 함수 

이제 메인 함수를 살펴보자. 간략한 전체 플로우를 볼 수 있다. 

```
func main() {
  // 전달된 파라미터를 추출하여 글로벌 변수에 할당한다. 
	extractParameter()
  // aws 로 시크릿 정보를 가져온다. 
	err, secret := getSecret()
	if err != nil {
		return;
	}
  // 결과를 화면에 출력한다. 
	fmt.Println(secret + "")

}   
```

위 내요을 보면 다음과 같은 흐름으로 진행된다. 

1. 파라미터를 추출한다. 
2. 시크릿 정보를 aws 에서 조회한다. 
3. 결과 시크릿을 콘솔에 출력한다. 

### Source Part04 extractParameter() 

go 에서 파라미터로 전달된 옵션들을 추출하여 변수로 저장한다. 

```
func extractParameter() {
	secretName = flag.String("s", DefaultSecretName, "<projectName>/<path1>/<path2>...")
	region = flag.String("r", DefaultRegion, "ap-northeast-2")
	keyValue = flag.String("k", "", "usersecretkey")
	profile = flag.String("p", DefaultProfile, "default")

	flag.Parse()
}
```

### Source Part05 시크릿 정보 조회하기 

이제 aws 에 접속해서 시크릿 정보를 어떻게 조회 하는지 살펴 보자. 

```
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

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	var keyValueMap map[string]interface{}

	if result.SecretString != nil {
		secretString = *result.SecretString

		if *keyValue == "" {
			return nil, secretString
		}

		// b, _ := json.Marshal(secretString)
		json.Unmarshal([]byte(secretString), &keyValueMap)
		if keyValueMap[*keyValue] == nil {
			return nil, ""
		} else {
			return nil, keyValueMap[*keyValue].(string)
		}
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			// fmt.Println("Base64 Decode Error:", err)
			return err, ""
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])

		if *keyValue == "" {
			return nil, decodedBinarySecret
		}

		// b, _ := json.Marshal(decodedBinarySecret)
		json.Unmarshal([]byte(decodedBinarySecret), &keyValueMap)
		if keyValueMap[*keyValue] == nil {
			return nil, ""
		} else {
			return nil, keyValueMap[*keyValue].(string)
		}	
	}
}
```

- secretsmanager.New: 이 소스는 aws configure 에 의해서 생성된 시크릿 정보를 이용하여 aws secretmanager 에 접속을 생성하는 코드이다. 여기서는 region 정보를 전달하고 있다. 
- secretsmanager.GetSecretValueInput: 이 부분으로 시크릿에 해당하는 암오화 키/값을 조회하도록 필요한 시크릿 정보를 전달하는 코드이다. 
- svc.GetSecretValue(input): 생성한 시크릿 정보를 바탕으로 실제 시크릿 키/값을 조회한다. 
- var keyValueMap map[string]interface{}: 획득한 시크릿 정보를 key/value 자료 구조를 위해서 map 을 생성했다. 키는 string 이며 값은 interface 가 되어 필요한 값으로 변환이 가능하다. 
- json.Unmarshal([]byte(secretString), &keyValueMap): 결과 정보를 맵으로 변환 작업을 한다. Unmarshal 을 통해서 스트링을 맵으로 변환이 가능하다. 
- 아래 else 부분은 바이트 스트링으로 넘어온 정보를 디코딩 하는 것으로 string plain text 와 처리 로직은 동일하며 변환 과정만 다르다. 

### 테스트하기. 

이제 테스트를 해보자. 

#### 전체 secret 키/값 정보 조회하기. 

```
go run main.go -s myproject/schooldevops/db -p schooldevops

{"password":"pass!@#qwe","username":"schooldevops","usertoken":"EzAFK6lV5AzEy4VFv2ND44g3nhqo3bTgnt3SMZFARLA="}
```

k 파라미터를 제외한경우 전체를 출력하고 있다. 

#### 특정 시크릿 키를 이용하여 값 조회하기. 

```
go run main.go -s myproject/schooldevops/db -p schooldevops -k usertoken

EzAFK6lV5AzEy4VFv2ND44g3nhqo3bTgnt3LA=
```

usertoken 에 해당하는 값을 조회했다. 

### 환경변수 활용하기. 

환경 변수에 세팅해서 활용하는 방법의 경우 다음과 같이 지정해 줄 수 있다. 

setsecret.sh 파일을 다음과 같이 작성해보자. 

```
#!/bin/bash

export DB_PASSWD="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k password`"
export DB_USERNAME="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k username`"
export USER_TOKEN="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k usertoken`"

echo $DB_PASSWD
echo $DB_USERNAME
echo $USER_TOKEN

```

쉘을 실행해보면 다음과 같은 결과를 볼 수 있다. 

```
sudo chmod 700 setsecret.sh

./setsecret.sh

pass!@#qwe
schooldevops
EzAFK6lV5AzEy4VFv2ND44g3nhqo3bTgnt3SMZFARLA=
```

shell 스크립트 내에서만 시크릿을 설정되므로, 다음과 같이 echo 로도 볼 수 없다. 

```
echo $DB_PASSWD

```

결과가 나타나지 않는다. 

즉, 쉘로 어플리케이션을 수행하는 방향, 혹은 컨테이너로 수행해서 환경변수를 전달하는 코드를 작성하면 될 것이다. 

## 결론 

이로써 aws secret 값을 조회하는 방법을 확인해 봤다. 

위 코드의 빌드 버젼을 배포 클러스터에서 활용하면, 필요한 환경 변수 값을 세팅할 수 있다는 것도 살펴 보았다. 

이 코드는 실제 어플리케이션 코드에 시크릿을 조회하는 코드를 작성하는 것 보다 유연성이 더 높다. 



