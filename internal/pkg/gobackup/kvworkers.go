package gobackup

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
//filename is a string with the drive location of a file to be uploaded
//hash is the hash of the file
//A normal hash should be 16 bytes and anything larger indicates the file has been split amoung several files. The additional length is the file number appended to the hash

func UploadKV(cf *Account, meta Metadata, filename string) bool {
	//max value size = 25 mb
	fmt.Println("UploadKV starting")

	client := &http.Client{}

	var fileUpload bytes.Buffer
	written := 0 //bytes written to fileUpload
	hash := meta.Hash

	/*	fmt.Fprintf(&fileUpload, "size: %d", 85)
		written += 1
		hash = hash + "1"
	*/

	if meta.FileNum == 0 {

		//open a pipe
		pr, pw := io.Pipe()
		errCh := make(chan error, 1)
		go zipInit(filename, pr, pw, errCh)
		written, err = io.CopyN(&fileUpload, pr, 24*1024*1024)
		if err != nil {
			log.Fatalln(err)
		} else { //continue the upload
			meta.FileNum += 1
			hash = hash + meta.FileNum
			written, err = io.CopyN(&fileUpload, meta.pr, 24*1024*1024)
			if err != nil {
				log.Fatalln(err)
			}
		}
	} //if

	//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
	if written == 24*1024*1024 {
		meta.pipe = pr
		UploadKV(cf, meta, filename)
	}

	//copy from file to the writer
	//		zipFile, _ := os.Create("file.zip")
	// Do stuff with pr here.
	//		_, _ = io.Copy(zipFile, pr)
	//create the zip file
	//		zipFile.Close()
	//file.Close()
	//	var exitCode int
	//	exitCode = 5

	//file, err := os.Open(filename)
	//if err != nil {
	//	log.Fatalln(err)
	//}

	//request with Metadata
	//	      PUT accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name?expiration=:expiration&expiration_ttl=:expiration_ttl
	//bulk request
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/bulk"
	//request accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name

	//normal
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash

	fmt.Println("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
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
	fmt.Println("Done with uploadKV")
	return true
}

//implementation of the workers kv download
func DownloadKV(cf *Account, dataKey string, filepath string) string {

	client := &http.Client{}

	//GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey

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

	req.Header.Set("X-Auth-Email", cf.Email)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	defer resp.Body.Close()

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
