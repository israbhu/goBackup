package gobackup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
)

var wg sync.WaitGroup
var DryRun = false

//validate that the preferences file has all the correct fields
func ValidateCF(cloud *Account) bool {
	//	Account, Data, Email, Namespace, Key, Token, Location, Zip, Backup string
	pass := true

	if DryRun {
		return true
	}
	//check the required fields are not blank
	if cloud.Email == "" {
		Logger.Fatalf("Email information is empty. Please edit your preferences.toml with the email associated with your cloudflare account")
		pass = false
	}
	if cloud.Namespace == "" {
		pass = false
		Logger.Fatalf("Namespace information is empty. Please edit your preferences.toml with valid info")
	}
	if cloud.Account == "" {
		pass = false
		Logger.Fatalf("Account information is empty. Please edit your preferences.toml with valid info")
	}
	if cloud.Key == "" || cloud.Token == "" {
		pass = false
		Logger.Fatalf("Key and Token are empty. Please edit your preferences.toml with a valid key or token. It is best practice to access your account through a least priviledged token.")
	}
	//check the length of data

	return pass
}

//get the stored keys on the account
func GetKVkeys(cf *Account) []byte {
	client := &http.Client{}

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	//request     accounts/:account_identifier/storage/kv/namespaces/:namespace_i3dentifier/values/:key_name
	//request GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/keys
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/keys"

	Logger.Println("GET KEY REQUEST:" + request)

	//get request to get the key data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		Logger.Fatalln(err)
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

	if DryRun {
		return []byte("dry run")
	}

	resp, err := client.Do(req)
	if err != nil {
		Logger.Fatalln(err)
	}

	//debug information
	if Debug {
		Logger.Print("Printing Response Header info: \n")
		Logger.Println(resp)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Fatalln(err)
	}

	//verbose information showing the "response" json data
	//name and metadata fields
	if Verbose {
		Logger.Println("Response Body: \n" + string(body))
	}

	return body
}

//func UploadMultiPart(client *http.Client, url string, values map[string]io.Reader, filename string) (err error) {
func UploadMultiPart(cf *Account, meta Metadata) bool {

	filename := meta.FileName
	//max value size = 25 mb
	Logger.Println("UploadKV starting")

	client := &http.Client{}

	// Prepare a form that you will submit to that URL.
	var fileUpload bytes.Buffer

	hash := meta.Hash

	//get a multipart writer
	w := multipart.NewWriter(&fileUpload)

	//create the name="value" part of the upload
	formWriter, err := w.CreateFormFile("value", filename)
	if err != nil {
		Logger.Fatalln(err)
	}

	//if the foreign key is blank, we need to upload stuff
	if meta.ForeignKey == "" {
		//open a pipe
		pr, pw := io.Pipe()

		if cf.Zip == "zstandard" {
			if Verbose {
				Logger.Println("*************")
				Logger.Println("zstandard")
				Logger.Println("*************")
			}
			go zStandardInit(filename, pw)
		} else if cf.Zip == "zip" {
			if Verbose {
				Logger.Println("*************")
				Logger.Println("zip")
				Logger.Println("*************")
			}
			errCh := make(chan error, 1)
			go zipInit(filename, pr, pw, errCh)
		} else { //no compression
			if Verbose {
				Logger.Println("*************")
				Logger.Println("no compression")
				Logger.Println("*************")
			}
			go copyFile(filename, pr, pw)
		}

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(formWriter, pr, 24*1024*1024)
		if err != nil && err != io.EOF {
			Logger.Printf("Err != nil, Bytes written:%v", written)
			Logger.Fatalln(err)
		}

		Logger.Printf("%v encoded %v bytes\n", cf.Zip, written)
	} //end check of foreign key

	formWriter, err = w.CreateFormField("metadata")
	if err != nil {
		Logger.Fatalln(err)
	}

	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		Logger.Fatalf("Could not marshal metadata %+v: %v", meta, err)
	}
	formWriter.Write(jsonBytes) //send metadata

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/valu
	/*
		if DryRun {
			request = "127.0.0.1"
		}
	*/
	Logger.Println("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
	if err != nil {
		Logger.Fatalln(err)
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	if Debug {
		Logger.Printf("Request to be sent: %q\n", fmt.Sprintf("%+v", req))
	}

	if DryRun {
		return true
	}
	resp, err := client.Do(req)
	if err != nil {
		Logger.Fatalln(err)
	}

	Logger.Println(resp)

	var response CloudflareResponse

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Fatalln(err)
	}
	json.Unmarshal(body, &response)

	if !response.Success {
		Logger.Println("***Body of response***")
		Logger.Println(string(body))
		Logger.Println("***End Body of response***")
		Logger.Println("File was not uploaded, exiting")
		Logger.Fatalln("No upload!")
	} else {
		Logger.Printf("Successfully uploaded %v\n", filename)
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
	Logger.Println("UploadKV starting")

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
			Logger.Fatalf("Bytes written: %d, err: %v\n", written, err)
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			Logger.Printf("***File is larger than 24MB, new upload initiated***")
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
			Logger.Fatalln(err)
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			Logger.Printf("***File is larger than 24MB, new upload initiated***")

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
	Logger.Println("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
	if err != nil {
		Logger.Fatalln(err)
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
		Logger.Fatalln(err)
	}

	Logger.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger.Fatalln(err)
	}
	Logger.Println(string(body))
	Logger.Println("Done with uploadKV")

	//wait for all uploads and downloads to complete
	wg.Wait()

	return true
}

//implementation of the workers kv download
func DownloadKV(cf *Account, dataKey string, filepath string) bool {

	client := &http.Client{}

	//GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey
	/*
		if DryRun {
			request = "127.0.0.1"
		}
	*/
	req, err := http.NewRequest("GET", request, nil)
	if err != nil {
		Logger.Fatalln(err)
	}

	Logger.Println("REQUEST:" + request)

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	if DryRun {
		return true
	}
	resp, err := client.Do(req)
	if err != nil {
		Logger.Fatalln(err)
	}

	Logger.Println(resp)

	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		Logger.Println(err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		Logger.Println(err)
	}

	return true
	//	return ("Done Downloading!")

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

type CloudflareResponse struct {
	Result   []MyData `json:"result"`
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Messages []string `json:"messages"`
}

type MyData struct {
	Name        string   `json:"name"`
	TheMetadata Metadata `json:"metadata"`
}
