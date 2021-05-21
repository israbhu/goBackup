package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml"

	"github.com/israbhu/goBackup/internal/pkg/gobackup"
)

//***************global variables*************
var cf gobackup.Account //account credentials and preferences
var dat gobackup.Data1  //local datastore tracking uploads and Metadata
var verbose bool        //flag for extra info output to console

//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup() {
	for _, list := range dat.TheMetadata {
		//		gobackup.UploadKV(&cf, list)
		gobackup.UploadMultiPart(&cf, list)
	}
}

//read from a toml file
//check that the file exists since the function can be called from a commandline argument
func readTOML(file string) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}
	astring := string(dat)

	fmt.Println("File as string: " + astring)

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

		fmt.Printf("Unmarshalling the preferences file:%v\n", file)
		fmt.Println("Contents of file:")
		fmt.Println(cf)
		fmt.Println("Contents of file end")

		//verbose flag
		if verbose {
			fmt.Println("Reading in the preference file:" + file)
			fmt.Println(cf)
		}
	}

}

//write a toml file
func writeTOML() {
}

func check(e error) {
	if e != nil {
		panic(e)
	}
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
		fmt.Println("getFiles name is blank, getting files from local directory")
		name = "."
	}

	if checkFileExist(name) {

		stat, err := os.Stat(name)
		check(err)

		//if it's a regular file, append and return f
		if stat.Mode().IsRegular() {
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

	if *altPrefFlag != "" {
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
		fmt.Println("Extracting flags from commandline:")
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
	if err != nil {
		log.Fatalf("searchData failed opening file:data.dat")
	}
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

//create a toml file from a struct
func dataFile() {
	doc, _ := toml.Marshal(&dat)

	fmt.Println(string(doc))
	fmt.Println(dat)
	err := ioutil.WriteFile("data.dat", doc, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = os.Lstat("data.dat")
	if err != nil {
		log.Fatalln(err)
	}
}

//******* This struct contains the data needed to access the cloudflare infrastructure. It is stored on drive in the file preferences.toml *****
type Account struct {
	//cloudflare account information
	Namespace, // namespace is called the "namespace id" on the cloudflare website for Workers KV
	Account, // account is called "account id" on the cloudflare dashboard
	Key, // key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Token, // Token is used instead of the key and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Email, // email is the email associated with your cloudflare account
	Data,
	Location,
	Zip,
	Backup string
}

// extract toml data for account and behaviour information
// extract out the files for backup
// backup the data
func main() {

	fmt.Println(gobackup.GetMetadata(gobackup.Metadata{}))
	readTOML("preferences.toml")
	fmt.Println(cf)
	fmt.Println("****************************")

	//	fmt.Println("zipping a file")
	//	gobackup.ZipFile("zipsuite.txt", "zipsuite.zip")
	//get the command arguments
	//command line can overwrite the data from the preferences file
	extractCommandLine()

	//get the filelist for backup
	var fileList []string

	fmt.Printf("CF LOCATION:%v", cf.Location)

	backupLocations := strings.Split(cf.Location, ",")

	for _, l := range backupLocations {
		fileList = getFiles(strings.TrimSpace(l), fileList)
	}

	sort.Strings(fileList)

	//fill in the Metadata
	for _, f := range fileList {
		hash := gobackup.Md5file(f)

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

	if len(dat.TheMetadata) == 0 {
		fmt.Println("All files are up to date! Exiting!")
		os.Exit(0)
	}

	fmt.Printf("Original File: %s, Data Size: %v, Data Count: %v", dat.TheMetadata[0].FileName, dat.DataSize, dat.Count)
	//split the work and backup
	backup()

	fmt.Println(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

	//update the local data file
	gobackup.DataFile2("data.dat", &dat)

} //main
