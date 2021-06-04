package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/israbhu/goBackup/internal/pkg/gobackup"
	"github.com/pelletier/go-toml"
)

//***************global variables*************
var cf gobackup.Account //account credentials and preferences
var dat gobackup.Data1  //local datastore tracking uploads and Metadata
var verbose bool        //flag for extra info output to console

//checks the errors********
func CheckError(err error, message string) {
	if err != nil {
		if message == "" {
			log.Fatalf("Error found! %v", err)
		} else {
			log.Fatalf(message+" %v", err)
		}
	}
}

//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup() {
	for _, list := range dat.TheMetadata {
		//		gobackup.UploadKV(&cf, list)
		gobackup.UploadMultiPart(&cf, list)
	}
}

func validatePreferences() {
	//account, namespace, email, key, token, location
	if cf.Account == "" {
		log.Fatalf("Account information is empty. Please edit your preferences.toml with valid info")
	} else if cf.Namespace == "" {
		log.Fatalf("Namespace information is empty. Please edit your preferences.toml with valid info")
	} else if cf.Email == "" {
		log.Fatalf("Email information is empty. Please edit your preferences.toml with the email associated with your cloudflare account")
	} else if cf.Key == "" && cf.Token == "" {
		log.Fatalf("Key and Token are empty. Please edit your preferences.toml with a valid key or token. It is best practice to access your account through a least priviledged token.")
	}

}

//read from a toml file
//check that the file exists since the function can be called from a commandline argument
func readTOML(file string) {
	dat, err := ioutil.ReadFile(file)
	CheckError(err, "")

	astring := string(dat)

	doc2 := []byte(astring)

	if file == "data.dat" {
		toml.Unmarshal(doc2, &dat)
		//verbose flag
		if verbose {
			fmt.Println("Reading in the preference file:" + file)
			fmt.Println(dat)
		}

	} else {
		toml.Unmarshal(doc2, &cf)

		//verbose flag
		if verbose {
			fmt.Println("Reading in the preference file:" + file)
			fmt.Println(cf)
		}
	}

}

//write a toml file
func writeTOML(file string) {
	//get []byte from type cf
	data, err := toml.Marshal(&cf)
	if err != nil {
		log.Fatalf("writeTOML has encountered an error: %v", err)
	}

	//open file to write
	writeFile, err := os.Open(file)
	if err != nil {
		log.Fatalf("writeTOML has encountered an error: %v", err)
	}

	//write to file
	io.WriteString(writeFile, string(data))

}

//check if a file called name exists
func checkFileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		fmt.Println("file does not exist")
		return false
	}
	return true
}

//gets the size of a file called name
func getFileSize(name string) int64 {
	fi, err := os.Stat(name)
	CheckError(err, "")
	// return the size in bytes
	return fi.Size()
}

//create a new empty file named name
func createEmptyFile(name string) {
	if checkFileExist(name) {
		return
	}

	d := []byte("")
	CheckError(ioutil.WriteFile(name, d, 0644), "")
}

//make a new directory called name
func mkdir(name string) {
	if checkFileExist(name) {
		return
	} else {
		fmt.Print("not exists")
		err := os.Mkdir(name, 0755)
		CheckError(err, "")
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
		fmt.Println("getFiles name is blank, getting files from local directory")
		name = "."
	}

	if checkFileExist(name) {

		stat, err := os.Stat(name)
		CheckError(err, "")

		//if it's a regular file, append and return f
		if stat.Mode().IsRegular() {
			f = append(f, name)
			return f
		}
	}

	fmt.Printf("getFiles name=**%v**\n", name)
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		CheckError(err, "")
		//remove .
		if path != "." && !info.IsDir() {
			f = append(f, path)
		}
		return nil
	})
	CheckError(err, "")
	return f
}

//process the command line commands
//yes, email, Account, Data, Email, Namespace, Key, Token, Location string
//backup strategy, zip, encrypt, verbose, sync, list data, alt pref, no pref
func extractCommandLine() {
	var emailFlag = flag.String("email", "", "User email")
	var accountFlag = flag.String("account", "", "User Account")
	var nsFlag = flag.String("namespace", "", "User's Namespace")
	var keyFlag = flag.String("key", "", "Account Global Key")
	var tokenFlag = flag.String("token", "", "Configured KV Workers key")
	var addLocationFlag = flag.String("addLocation", "", "Add these locations/files to backup")
	var locationFlag = flag.String("location", "", "Use only these locations to backup")
	var backupFlag = flag.String("backup", "", "Backup strategy")
	var downloadFlag = flag.String("download", "", "Folder/files to download")
	var zipFlag = flag.String("zip", "", "zip")
	var verboseFlag = flag.Bool("v", false, "More information")
	var altPrefFlag = flag.String("pref", "", "use an alternate preference file")

	flag.Parse()
	fmt.Println("Checking flags!")

	if *altPrefFlag != "" {
		fmt.Println("Alernate Preferences file detected, checking:")
		readTOML(*altPrefFlag)
		if !gobackup.ValidateCF(&cf) {
			fmt.Printf("%v has errors that need to be fixed!", *altPrefFlag)
		}
	}

	//overwrite over any preferences file
	if *emailFlag != "" {
		cf.Email = *emailFlag
	}
	if *accountFlag != "" {
		cf.Account = *accountFlag
	}
	if *nsFlag != "" {
		cf.Namespace = *nsFlag
	}
	if *keyFlag != "" {
		cf.Key = *keyFlag
	}
	if *tokenFlag != "" {
		cf.Token = *tokenFlag
	}
	if *locationFlag != "" { //replace the locations
		cf.Location = *locationFlag
	}
	if *addLocationFlag != "" { //add to the locations
		cf.Location = cf.Location + "," + *addLocationFlag
	}
	if *backupFlag != "" {
		cf.Backup = *backupFlag
	}
	if *zipFlag != "" {
		cf.Zip = *zipFlag
	}
	if *verboseFlag {
		verbose = true
		fmt.Println("Verbose information is true, opening the flood gates!")
	}
	if *downloadFlag != "" {
		cf.Location = *downloadFlag //hash
		gobackup.DownloadKV(&cf, *downloadFlag, "download.file")
		fmt.Println("Downloaded a file!")
		os.Exit(0)
	}
}

//search the data file for hash, line by line
//hash is the hash from a file, fileName is a file
//in the future, may introduce a binary search or a simple index for a pre-sorted data file
func searchData(hash string, fileName string) bool {

	file, err := os.Open("data.dat")
	CheckError(err, "searchData failed opening file:data.dat")
	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		a := scan.Text()           //get lines of data.dat
		b := strings.Split(a, ":") //split out the data
		if len(b[0]) != 32 {
			b[0] = b[0][:32] //get the base hash
		}

		if b[0] == hash && fileName == b[1] {
			return true
		}
	}
	return false
}

/*
//create a toml file from a struct
?? redundant, same as writeTOML
func dataFile() {
	doc, _ := toml.Marshal(&dat)

	fmt.Println(string(doc))
	fmt.Println(dat)
	err := ioutil.WriteFile("data.dat", doc, 0644)
	CheckError(err, "")
	_, err = os.Lstat("data.dat")
	CheckError(err, "")
}
*/

//******* This struct contains the data needed to access the cloudflare infrastructure. It is stored on drive in the file preferences.toml *****
type Account struct {
	//cloudflare account information
	Namespace, // namespace is called the "namespace id" on the cloudflare website for Workers KV
	Account, // account is called "account id" on the cloudflare dashboard
	Key, // key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Token, // Token is used instead of the key and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Email, // email is the email associated with your cloudflare account
	Data,
	Location, //locations of the comma deliminated file and folders to be backed up
	Zip, //string to determine zip type. Currently none, zstandard, and zip
	Backup string
}

func main() {
	//command line can overwrite the data from the preferences file
	extractCommandLine()

	//if no alternate prefences were in the command line, extract the default
	if cf.Token == "" {
		readTOML("preferences.toml")
	}

	//make sure the preferences are valid
	validatePreferences()
	gobackup.ValidateCF(&cf)

	//the filelist for backup
	var fileList []string

	if verbose {
		fmt.Printf("CF LOCATION:%v", cf.Location)
	}

	backupLocations := strings.Split(cf.Location, ",")

	for _, l := range backupLocations {
		fileList = getFiles(strings.TrimSpace(l), fileList)
	}

	sort.Strings(fileList)

	//fill in the Metadata
	for _, f := range fileList {
		hash := gobackup.Md5file(f)
		openFile, _ := os.Open(f)
		contents, _ := ioutil.ReadAll(openFile)

		fmt.Printf("For Loop: (f:%v)(hash:%v)(contents:%v)", f, hash, string(contents))

		//if not found
		if !searchData(hash, f) {
			meta := gobackup.CreateMeta(f)
			fmt.Println("NOT FOUND AND INCLUDING! " + hash + "-" + gobackup.GetMetadata(meta))

			//update the data struct
			dat.TheMetadata = append(dat.TheMetadata, meta)

			sort.Sort(gobackup.ByHash(dat.TheMetadata))

			dat.DataSize += meta.Size
			dat.Count += 1
		} else {
			fmt.Println("FOUND AND EXCLUDING! " + hash)
		}
	} //for

	//get keys
	fmt.Println("Getting the keys and metadata!")
	gobackup.GetKVkeys(&cf)

	//TheMetadata is empty, then there is no work left to be done. Exit program
	if len(dat.TheMetadata) == 0 {
		fmt.Println("All files are up to date! Exiting!")
		os.Exit(0)
	}

	//information for user
	fmt.Printf("Original File: %s, Data Size: %v, Data Count: %v", dat.TheMetadata[0].FileName, dat.DataSize, dat.Count)

	//split the work and backup
	backup()

	//download the data
	fmt.Println(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

	//update the local data file
	gobackup.DataFile2("data.dat", &dat)

} //main
