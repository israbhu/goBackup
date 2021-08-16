package cf

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

	"github.com/golang/glog"

	"github.com/israbhu/goBackup/internal/pkg/gobackup"
)

var wg sync.WaitGroup

//get the stored keys on the account
func (cf *Account) GetKVkeys() ([]byte, error) {
	client := &http.Client{}

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	//request     accounts/:account_identifier/storage/kv/namespaces/:namespace_i3dentifier/values/:key_name
	//request GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/keys
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.AccountID + "/storage/kv/namespaces/" + cf.Namespace + "/keys"

	glog.Infoln("GET KEY REQUEST:" + request)

	//get request to get the key data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		return nil, err
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

	if cf.LocalOnly {
		return []byte("dry run"), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	//debug information
	glog.V(2).Info("Printing Response Header info: \n")
	glog.V(2).Infoln(resp)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//verbose information showing the "response" json data
	//name and metadata fields
	glog.V(1).Infoln("Response Body: " + string(body))
	//	fmt.Println("Response Body: " + string(body))

	return body, nil
}

func UploadMultiPart(cf *Account, meta gobackup.Metadata) error {

	filename := meta.FileName
	//max value size = 25 mb
	glog.Infoln("UploadKV starting")

	client := &http.Client{}

	// Prepare a form that you will submit to that URL.
	var fileUpload bytes.Buffer

	hash := meta.Hash

	//get a multipart writer
	w := multipart.NewWriter(&fileUpload)

	//create the name="value" part of the upload
	formWriter, err := w.CreateFormFile("value", filename)
	if err != nil {
		return err
	}

	//if the foreign key is blank, we need to upload stuff
	if meta.ForeignKey == "" {
		//open a pipe
		pr, pw := io.Pipe()

		if cf.Zip == "zstandard" {
			if glog.V(1) {
				glog.Infoln("*************")
				glog.Infoln("zstandard")
				glog.Infoln("*************")
			}
			go gobackup.ZStandardInit(filename, pw)
		} else if cf.Zip == "zip" {
			if glog.V(1) {
				glog.Infoln("*************")
				glog.Infoln("zip")
				glog.Infoln("*************")
			}
			errCh := make(chan error, 1)
			go gobackup.ZipInit(filename, pr, pw, errCh)
		} else { //no compression
			if glog.V(1) {
				glog.Infoln("*************")
				glog.Infoln("no compression")
				glog.Infoln("*************")
			}
			go gobackup.CopyFile(filename, pr, pw)
		}

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(formWriter, pr, 24*1024*1024)
		if err != nil && err != io.EOF {
			return fmt.Errorf("Err != nil, %d Bytes written: %v", written, err)
		}

		glog.Infof("%v encoded %v bytes\n", cf.Zip, written)
	} //end check of foreign key

	formWriter, err = w.CreateFormField("metadata")
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("Could not marshal metadata %+v: %v", meta, err)
	}
	formWriter.Write(jsonBytes) //send metadata

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.AccountID + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/valu
	/*
		if cf.LocalOnly {
			request = "127.0.0.1"
		}
	*/
	glog.V(2).Infoln("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
	if err != nil {
		return err
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	glog.V(2).Infof("Request to be sent: %q\n", fmt.Sprintf("%+v", req))

	if cf.LocalOnly {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	glog.V(2).Infoln(resp)

	var response CloudflareResponse

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(body, &response)

	if !response.Success {
		if bool(glog.V(2)) {
			glog.Infoln("***Body of response***")
			glog.Infoln(string(body))
			glog.Infoln("***End Body of response***")
			glog.Infoln("File was not uploaded, exiting")
		}
		return fmt.Errorf("No upload!")
	} else {
		glog.Infof("Successfully uploaded %v\n", filename)
	}
	//wait for all uploads and downloads to complete
	// FIXME This wait group may not have anything in it.
	wg.Wait()

	return nil

}

//implementation of the workers kv upload
//filename is a string with the drive location of a file to be uploaded
//hash is the hash of the file
//A normal hash should be 16 bytes and anything larger indicates the file has been split amoung several files. The additional length is the file number appended to the hash
// FIXME The only caller of this function is the function itself. Being
// launched recursively and concurrently makes it very difficult to return an
// value or an error. This function should probably be rewritten, and it should
// not panic on error.
func UploadKV(cf *Account, meta gobackup.Metadata) bool {

	filename := meta.FileName
	//max value size = 25 mb
	glog.Infoln("UploadKV starting")

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
		go gobackup.ZStandardInit(filename, pw)

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(&fileUpload, pr, 24*1024*1024)
		if err != nil && err != io.EOF {
			glog.Errorf("Bytes written: %d, err: %v\n", written, err)
			return false
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			glog.Infof("***File is larger than 24MB, new upload initiated***")
			meta.FileNum += 1
			meta.PR = pr

			//asynchronousUpload
			wg.Add(1)
			go UploadKV(cf, meta)
		}

	} else {

		//create the hash with appended file number
		hash = fmt.Sprintf("%s%d", hash, meta.FileNum)

		//copy up to 24MB using the pipereader
		written, err := io.CopyN(&fileUpload, meta.PR, 24*1024*1024)
		if err != nil {
			glog.Errorln(err)
			return false
		}

		//if written is exactly at the maximum N, then we haven't finished using the data in the pipe
		if written == 24*1024*1024 {
			glog.Infof("***File is larger than 24MB, new upload initiated***")

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
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.AccountID + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + hash + "?value=testvalue?metadata=testmetadata"
	//	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/valu
	glog.Infoln("UPLOAD REQUEST:" + request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &fileUpload)
	if err != nil {
		glog.Errorln(err)
		return false
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
		glog.Errorln(err)
		return false
	}

	glog.Infoln(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorln(err)
		return false
	}
	glog.Infoln(string(body))
	glog.Infoln("Done with uploadKV")

	//wait for all uploads and downloads to complete
	wg.Wait()

	return true
}

//implementation of the workers kv download
//dataKey should be a unique md5 key used as the primary key on cloudflare
//filepath is the path to create the downloaded file
// FIXME There are no callers of this function.
func DownloadKV(cf *Account, dataKey string, downloadPath string) error {

	client := &http.Client{}

	//GET accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.AccountID + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey
	/*
		if cf.LocalOnly {
			request = "127.0.0.1"
		}
	*/
	req, err := http.NewRequest("GET", request, nil)
	if err != nil {
		return err
	}

	glog.Infoln("REQUEST:" + request)

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	if cf.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cf.Token)
	} else if cf.Key != "" {
		req.Header.Set("X-Auth-Key", cf.Key)
	}

	req.Header.Set("X-Auth-Email", cf.Email)

	if cf.LocalOnly {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	glog.V(2).Infoln(resp)

	defer resp.Body.Close()

	out, err := os.Create(downloadPath)
	if err != nil {
		glog.Infoln(err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		glog.Infoln(err)
	}

	return nil

}

type CloudflareResponse struct {
	Result   []gobackup.MyData `json:"result"`
	Success  bool              `json:"success"`
	Errors   []string          `json:"errors"`
	Messages []string          `json:"messages"`
}
