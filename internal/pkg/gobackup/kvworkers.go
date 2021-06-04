package gobackup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
)

var wg sync.WaitGroup

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

	//request     accounts/:account_identifier/storage/kv/namespaces/:namespace_i3dentifier/values/:key_name
	//request GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/keys
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/keys"

	fmt.Println("GET KEY REQUEST:" + request)
	//get request to get the key data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type
	req.Header.Set("Content-Type", "application/json")

	//for write/read

	//use token if available, try global key next
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

//func UploadMultiPart(client *http.Client, url string, values map[string]io.Reader, filename string) (err error) {
func UploadMultiPart(cf *Account, meta Metadata) bool {

	filename := meta.FileName
	//max value size = 25 mb
	fmt.Println("UploadKV starting")

	client := &http.Client{}

	// Prepare a form that you will submit to that URL.
	var fileUpload bytes.Buffer

	hash := meta.Hash

	//get a multipart writer
	w := multipart.NewWriter(&fileUpload)

	//create the name="value" part of the upload
	formWriter, err := w.CreateFormFile("value", filename)
	if err != nil {
		log.Fatalln(err)
	}

	//open a pipe
	pr, pw := io.Pipe()

	if cf.Zip == "zstandard" {
		fmt.Println("*************")
		fmt.Println("zstandard")
		fmt.Println("*************")
		go zStandardInit(filename, pw)
	} else if cf.Zip == "zip" {
		fmt.Println("*************")
		fmt.Println("zip")
		fmt.Println("*************")
		errCh := make(chan error, 1)
		go zipInit(filename, pr, pw, errCh)
	} else { //no compression
		fmt.Println("*************")
		fmt.Println("no compression")
		fmt.Println("*************")
		go copyFile(filename, pr, pw)
	}

	//copy up to 24MB using the pipereader
	written, err := io.CopyN(formWriter, pr, 24*1024*1024)
	if err != nil && err != io.EOF {
		fmt.Printf("Err != nil, Bytes written:%v", written)
		log.Fatalln(err)
	}

	formWriter, err = w.CreateFormField("metadata")
	if err != nil {
		log.Fatalln(err)
	}

	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		log.Fatalf("Could not marshal metadata %+v: %v", meta, err)
	}
	formWriter.Write(jsonBytes) //send metadata

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/valu

	fmt.Println("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
	if err != nil {
		log.Fatalln(err)
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	fmt.Printf("Request to be sent: %+q\n", req)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp)

	var response cloudflareResponse

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(body, &response)

	if !response.Success {
		fmt.Println("***Body of response***")
		fmt.Println(string(body))
		fmt.Println("***End Body of response***")
		fmt.Println("File was not uploaded, exiting")
		log.Fatalln("No upload!")
	} else {
		fmt.Printf("Successfully uploaded %v\n", filename)
	}
	//wait for all uploads and downloads to complete
	wg.Wait()

	return true

}

//implementation of the workers kv upload
//filename is a string with the drive location of a file to be uploaded
//hash is the hash of the file
//A normal hash should be 16 bytes and anything larger indicates the file has been split amoung several files. The additional length is the file number appended to the hash

func UploadKV(cf *Account, meta Metadata) bool {

	filename := meta.FileName
	//max value size = 25 mb
	fmt.Println("UploadKV starting")

	client := &http.Client{}

	var fileUpload bytes.Buffer
	hash := meta.Hash

	/*	fmt.Fprintf(&fileUpload, "size: %d", 85)
		written += 1
		hash = hash + "1"
	*/

	if meta.FileNum == 0 {

		//open a pipe
		pr, pw := io.Pipe()
		//		errCh := make(chan error, 1)
		//		go zipInit(filename, pr, pw, errCh)
		go zStandardInit(filename, pw)

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(&fileUpload, pr, 24*1024*1024)
		if err != nil && err != io.EOF {
			log.Fatalf("Bytes written: %d, err: %v\n", written, err)
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			fmt.Printf("***File is larger than 24MB, new upload initiated***")
			meta.FileNum += 1
			meta.pr = pr

			//asynchronousUpload
			wg.Add(1)
			go UploadKV(cf, meta)
		}

	} else {

		//create the hash with appended file number
		hash = fmt.Sprintf("%s%d", hash, meta.FileNum)

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(&fileUpload, meta.pr, 24*1024*1024)
		if err != nil {
			log.Fatalln(err)
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			fmt.Printf("***File is larger than 24MB, new upload initiated***")

			meta.FileNum += 1

			//asynchronousUpload
			wg.Add(1)
			go UploadKV(cf, meta)
		}

		//decrement waitgroup
		wg.Done()
	}

	//normal
	// 5/10/21	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash + "?value=testvalue?metadata=testmetadata"
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/valu
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

	//wait for all uploads and downloads to complete
	wg.Wait()

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

type cloudflareResponse struct {
	Result   string   `json:"result"`
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Messages []string `json:"messages"`
}
