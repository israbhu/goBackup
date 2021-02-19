package main

import (
	//flags
	"flag"
	"fmt"
//	"strings"
	//hash
	"crypto/md5"
	"encoding/hex"

	//read and write files
//	"bufio"
	//    	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	//http
	"log"
	"net/http"
	//	"io"
	"bytes"
	//	"mime/multipart"

	//toml
	"github.com/komkom/toml"
	"encoding/json"
)

//***************global variables*************
var cf Account

//convert hex bytes into a string
func hashToString(in []byte) string {
	return hex.EncodeToString(in)
}

//run md5 hash on a string
func md5string(a string) string {
	data := md5.Sum([]byte(a))
	return hashToString(data[:])
}

//run md5 hash on a file
func md5file(in string) string {
	dat, err := ioutil.ReadFile(in)
	check(err)

	data := md5.Sum(dat)
	return hashToString(data[:])
}

func verifyKV() bool {
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

	fmt.Println("Verify token:"+request)
	//get request to upload the data
	req, err := http.NewRequest("GET", request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	bearer := "cBqbac8aKYO570JE6CnT5J0uJvGn5kBvTNyCzZVC"
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


//implementation of the workers kv upload
//file is a string with the drive location of a file to be uploaded
func uploadKV(file, dataKey string) bool {
	//max value size = 25 mb
	//	await namespace.put(key, value)
	client := &http.Client{}

	value, err := ioutil.ReadFile(file)
	check(err)

	//	dataKey := md5file(file)

	//buffer to store our request body as bytes
	var requestBody bytes.Buffer

	requestBody.Write([]byte(value))

	//request accounts/:account_identifier/storage/kv/namespaces/:namespace_identifier/values/:key_name
	request := "https://api.cloudflare.com/client/v4/accounts/" + cf.Account + "/storage/kv/namespaces/" + cf.Namespace + "/values/" + dataKey

	fmt.Println("UPLOAD REQUEST:"+request)
	//put request to upload the data
	req, err := http.NewRequest(http.MethodPut, request, &requestBody)
	if err != nil {
		log.Fatalln(err)
	}

	//set the content type -- to verify
	req.Header.Set("Content-Type", "application/json")

	//for write/read
	bearer := "cBqbac8aKYO570JE6CnT5J0uJvGn5kBvTNyCzZVC"
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

//implementation of the workers kv download
func downloadKV(dataKey string) string {

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
	bearer := "cBqbac8aKYO570JE6CnT5J0uJvGn5kBvTNyCzZVC"
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
	//	fmt.Println(string(body))

	return (string(body))

}


/*what do do when uploading data
create md5 sum

compare to the list of hashes
sort

compress
encrypt
metadata
move it to cloudflare


*/
//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup(list []string) {
	//channel to determine end of files
	c := make(chan string, 15)

	//separate the commands out
	for i := 0; i < len(list); i++ {
		//		println("Printing List loop: ", len(list), list[i])

		//TODO create a hashmap of the data, detect if the file has changed, upload if changed
		go upload(list[i], c)
	}
	println(<-c)
}

//upload the data
//file is the location of the data on the drive
func upload(file string, c chan string) {
	//check the size of the file and split it if necessary
	if getFileSize(file) > 30000000 {//30 million bytes ~= 30MB
		//compress
		//encrypt
		//split the file into multiple parts
		//check the md5 hash against the local datastore
		//upload if it's not in the datastore

		//make sure c is sent outside of any loop
		c <- file
	} else {
		//compress
		//encrypt
		//md5
		hash := md5file(file)
		//check against the local datastore
		//upload if not in the datastore
		uploadKV(file, hash)
			c <- file
	}
}

//creates the actual metadata structure for a file
func createMeta(fileName string) {
	fi, err := os.Lstat(fileName)
	if err != nil {
		log.Fatalln(err)
	}

fmt.Printf("permissions: %#o\n", fi.Mode().Perm()) // 0400, 0777, etc.
	switch mode := fi.Mode(); {
	case mode.IsRegular():
		fmt.Println("regular file")
	case mode.IsDir():
		fmt.Println("directory")
	case mode&os.ModeSymlink != 0:
		fmt.Println("symbolic link")
	case mode&os.ModeNamedPipe != 0:
		fmt.Println("named pipe")
	}
}


//read from a toml file
func readTOML(file string) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}

	doc := string(dat)
	dec := json.NewDecoder(toml.New(bytes.NewBufferString(doc)))

	enc := json.NewEncoder(os.Stdout)
	for {
		if err := dec.Decode(&cf); err != nil {
			log.Println(err)
			return
		}

		if err := enc.Encode(&cf); err != nil {
			log.Println(err)
		}
	}
}

//write a toml file
func writeTOML() {
}



//***************************************************

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/*
createEmptyFile := func(name string) {
        d := []byte("")
        check(ioutil.WriteFile(name, d, 0644))
    }
*/

//check if a file called name exists
func checkFileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		fmt.Println("file does not exist")
		return false
	}
	return true
}

//gets the size of a file called name
func getFileSize(name string) int64{
	fi, err := os.Stat(name)
	if err != nil {
	    	panic(err)
	}
	// return the size in bytes
	return fi.Size()
}

//create a new empty file named name
func createEmptyFile(name string) {
	if checkFileExist(name) {
		return
	}

	d := []byte("")
	check(ioutil.WriteFile(name, d, 0644))
}

//make a new directory called name
func mkdir(name string) {
	if checkFileExist(name) {
		return
	} else {
		fmt.Print("not exists")
		err := os.Mkdir(name, 0755)
		check(err)
	}
}

//parameters
//name is the drive path to a folder
//f is a slice that contains all of the accumulated files
//return []string
//the return []string is the slice f
func getFiles(name string, f []string) []string {
	//make sure name is valid
	if name == "" {
		name = "."
	}

	if checkFileExist(name) {
	
		stat, err := os.Stat(name)
		check(err)

		//if it's a regular file, append and return f
		if(stat.Mode().IsRegular()) {
			f = append(f, name)
			return f
		}
	}
			

	fmt.Printf("getFiles name=**%v**\n", name)
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		//remove .
		if path != "." && !info.IsDir() {
			f = append(f, path)
		}
		return nil
	})
	if err != nil {
		//    	log.Println(err)
		panic(err)
	}
	return f
}

//yes, email, Account, Data, Email, Namespace, Key, Token, Location string
//backup strategy, zip, encrypt, verbose, sync, list data, alt pref, no pref
func extractCommandLine() {
	//verbose
//	var yFlag = flag.Bool("y", false, "Always accept all prompts")
	var emailFlag = flag.String("email", "", "User email")
	var accountFlag = flag.String("account", "", "User Account")
	var nsFlag = flag.String("namespace", "", "User's Namespace")
	var keyFlag = flag.String("key", "", "Account Global Key")
	var tokenFlag = flag.String("token", "", "Configured KV Workers key")
	var addLocationFlag = flag.String("addLocation", "", "Add these locations to backup")
	var locationFlag = flag.String("location", "", "Use only these locations to backup")
	var backupFlag = flag.String("backup", "", "Backup strategy")
	var zipFlag = flag.String("zip", "", "zip")
/*	var encryptFlag = flag.String("crypt", "", "encrypt")
	var verboseFlag = flag.Bool("v", false, "More information")
	var syncFlag = flag.Bool("s", false, "Synchronize local data to the cloud")
	var listdataFlag = flag.Bool("list", false, "List the data from the local data")
*/	var altPrefFlag = flag.String("pref", "", "use a alternate preference file")

	flag.Parse()

	if *altPrefFlag != "" {
		readTOML(*altPrefFlag)
	}

	//overwrite over any preferences file
	if(*emailFlag != "") {
		cf.Email = *emailFlag
	}
	if(*accountFlag != "") {
		cf.Account = *accountFlag
	}
	if(*nsFlag != "") {
		cf.Namespace = *nsFlag
	}
	if(*keyFlag != "") {
		cf.Key = *keyFlag
	}
	if(*tokenFlag != "") {
		cf.Token = *tokenFlag
	}
	if(*locationFlag != "") { //replace the locations
		cf.Location = *locationFlag
	}
	if(*addLocationFlag != "") { //add to the locations
		cf.Location = cf.Location + "," + *addLocationFlag
	}
	if(*backupFlag != "") {
		cf.Backup = *backupFlag
	}
	if(*zipFlag != "") {
		cf.Zip = *zipFlag
	}




}

//this struct stores the metadata that will be uploaded with each file
type fileData struct {
	//f1o1 = file 1 of 1
	//f1o4 = file 1 of 4
	//f2o4 = file 2 of 4
	//ph:  = previous file hash
	//fh:  = following file hash
	//note: = notes
	//created timestamp
	//modified timestamp
	//permissions
	//folder structure
	//metadata filename:
	//
//metadata example test.txt:f2o4:ph#:fh#:
	hash, data, metadata string
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

func main() {

	readTOML("preferences.toml")
	fmt.Println(cf)
	fmt.Println("****************************\n")

	//get the command arguements
	//command line can overwrite the data from the preferences file
	extractCommandLine()

//	cf.Location = cf.Location+commandLineLoc
	fmt.Println(cf)
	fmt.Println("---"+cf.Email+"---")
	fmt.Println("-----------------------")	
/*
//	verifyKV()

		var fileList = make([]string, 0)

		backupLocations := strings.Split(cf.Location, ",")

		for i := 0; i < len(backupLocations); i++ {
			fileList = getFiles(backupLocations[i], fileList)
			//		fileList = getFiles("c:/testdir", fileList)
		}
		//	fmt.Println(fileList);

		//backup the data using
//		backup(fileList)

		a := fileList[0:4]

		fmt.Println("BUFFER upload")

		for i := 0; i < len(a); i++ {
			fmt.Println(a[i] + "  hash:" + md5file(a[i]))
			//upload test
			backup(a)
		}

		fmt.Println("BUFFER download")
		for i := 0; i < len(a); i++ {
			//download test
			fmt.Println(downloadKV(md5file(a[i])))
		}
*/
}