package gobackup

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//	"net/url"
)

//validate that the preferences file has all the correct fields
func ValidateCF(cloud *Account) bool {
	//	Account, Data, Email, Namespace, Key, Token, Location, Zip, Backup string
	pass := true

	//check the required fields are not blank
	if cloud.Email == "" {
		pass = false
		fmt.Println("Email field is required. Do not leave blank!")
	}
	if cloud.Namespace == "" {
		pass = false
		fmt.Println("Namespace field is required. Do not leave blank!")
	}
	if cloud.Account == "" {
		pass = false
		fmt.Println("Account field is required. Do not leave blank!")
	}
	if cloud.Key == "" || cloud.Token == "" {
		pass = false
		fmt.Println("Must have a valid Key or Token. Do not leave blank!")
	}
	//check the length of data

	return pass
}

func VerifyKV(cf *Account) bool {
	//max value size = 25 mb
	//	await namespace.put(key, value)
	client := &http.Client{}

	//	value, err := ioutil.ReadFile(file)
	//	check(err)

	//	dataKey := md5file(file)

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	//	requestBody.Write([]byte(value))

	request := "https://api.cloudflare.com/client/v4/user/tokens/verify"

	fmt.Println("Verify token:" + request)
	//get request to upload the data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	bearer := cf.Token
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("X-Auth-Email", cf.Email)
	//	req.Header.Set("X-Auth-Key", cf.Key)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(string(body))

	return true
}

//get the stored keys on the account
func GetKVkeys(cf *Account) {
	client := &http.Client{}

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	//request     accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	//request GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/keys
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/keys"

	fmt.Println("GET KEY REQUEST:" + request)
	//get request to get the key data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read

	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(body))
}

//implementation of the workers kv upload
//file is a string with the drive location of a file to be uploaded
func UploadKV(cf *Account, dat *Data1) bool {
	//max value size = 25 mb
	//	await namespace.put(key, value)

	fmt.Println("UploadKV starting")

	client := &http.Client{}

	//TODO change BuildData return value to a stream
	//buildata2 should only build a string as large as 100 MB, must do another upload otherwise
	stringValue, err := BuildData2(dat)
	if err != nil {
		log.Fatalln(err)
	}

	value := []byte(stringValue)

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	requestBody.Write([]byte(value))

	//request with Metadata
	//	      PUT accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name?expiration=:expiration&expiration_ttl=:expiration_ttl
	//bulk request
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/bulk"
	//request accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name

	//normal
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey

	fmt.Println("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")
	//	req.Header.Set("Content-Type", "multipart/form-data")

	//for write/read

	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(string(body))
	fmt.Println("Done with uploadKV")
	return true
}

//implementation of the workers kv download
func DownloadKV(cf *Account, dataKey string, filepath string) string {

	client := &http.Client{}

	//GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey

	//	req, err := http.NewRequest(http.MethodPut, request, &requestBody)
	req, err := http.NewRequest("GET", request, nil)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("REQUEST:" + request)

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	//	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("X-Auth-Email", cf.Email)
	//	req.Header.Set("X-Auth-Key", cf.Key)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()

	/*
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		//	fmt.Println(string(body))
		// Create the file
	*/

	out, err := os.Create(filepath)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	return ("Done Downloading!")

}

//******* This struct contains the data needed to access the cloudflare infrastructure. It is stored on drive in the file preferences.toml *****
type Account struct {
	//cloudflare account information
	// namespace is called the "namespace id" on the cloudflare website for Workers KV
	// account is called "account id" on the cloudflare dashboard
	// key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	// Token is used instead of the key and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	// email is the email associated with your cloudflare account

	Account, Data, Email, Namespace, Key, Token, Location, Zip, Backup string
}
