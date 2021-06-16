package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
var Verbose bool        //flag for extra info output to console

//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup() {
	for _, list := range dat.TheMetadata {
		//		gobackup.UploadKV(&cf, list)
		gobackup.UploadMultiPart(&cf, list)
	}
}

/*
func validatePreferences() {
	//account, namespace, email, key, token, location
	if cf.Account == "" {
		gobackup.Logger.Fatalf("Account information is empty. Please edit your preferences.toml with valid info")
	} else if cf.Namespace == "" {
		gobackup.Logger.Fatalf("Namespace information is empty. Please edit your preferences.toml with valid info")
	} else if cf.Email == "" {
		gobackup.Logger.Fatalf("Email information is empty. Please edit your preferences.toml with the email associated with your cloudflare account")
	} else if cf.Key == "" && cf.Token == "" {
		gobackup.Logger.Fatalf("Key and Token are empty. Please edit your preferences.toml with a valid key or token. It is best practice to access your account through a least priviledged token.")
	}

}
*/

//read from a toml file
//check that the file exists since the function can be called from a commandline argument
func readTOML(file string) {
	dat, err := ioutil.ReadFile(file)
	gobackup.CheckError(err, "")

	astring := string(dat)

	doc2 := []byte(astring)

	if file == "data.dat" {
		toml.Unmarshal(doc2, &dat)
		//verbose flag
		if Verbose {
			fmt.Println("Reading in the data file:" + file)
			fmt.Println(dat)
		}

	} else {
		toml.Unmarshal(doc2, &cf)

		//verbose flag
		if Verbose {
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
		gobackup.Logger.Fatalf("writeTOML has encountered an error: %v", err)
	}

	//open file to write
	writeFile, err := os.Open(file)
	if err != nil {
		gobackup.Logger.Fatalf("writeTOML has encountered an error: %v", err)
	}

	//write to file
	io.WriteString(writeFile, string(data))
	writeFile.Close()
}

//gets the size of a file called name
func getFileSize(name string) int64 {
	fi, err := os.Stat(name)
	gobackup.CheckError(err, "")
	// return the size in bytes
	return fi.Size()
}

//create a new empty file named name
func createEmptyFile(name string) {
	if gobackup.FileExist(name) {
		return
	}

	d := []byte("")
	gobackup.CheckError(ioutil.WriteFile(name, d, 0644), "")
}

//make a new directory called name
func mkdir(name string) {
	if gobackup.FileExist(name) {
		return
	} else {
		fmt.Print("not exists")
		err := os.Mkdir(name, 0755)
		gobackup.CheckError(err, "")
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

	if gobackup.FileExist(name) {

		stat, err := os.Stat(name)
		gobackup.CheckError(err, "")

		//if it's a regular file, append and return f
		if stat.Mode().IsRegular() {
			f = append(f, name)
			return f
		}
	}

	fmt.Printf("getFiles name=**%v**\n", name)
	//	fmt.Printf("my cf email is %v", cf.Email)
	//	log.Fatalln("Stopping here for checkup")
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		gobackup.CheckError(err, "")
		//remove .
		if path != "." && !info.IsDir() {
			f = append(f, path)
		}
		return nil
	})
	gobackup.CheckError(err, "")
	return f
}

//process the command line commands
//yes, email, Account, Data, Email, Namespace, Key, Token, Location string
//backup strategy, zip, encrypt, verbose, sync, list data, alt pref, no pref
func extractCommandLine() {
	var emailFlag = flag.String("email", "", "Set the User email instead of using any preferences file")
	var accountFlag = flag.String("account", "", "Set the User Account instead of using any preferences file")
	var nsFlag = flag.String("namespace", "", "Set the User's Namespace instead of using any preferences file")
	var keyFlag = flag.String("key", "", "Set the Account Global Key instead of using any preferences file")
	var tokenFlag = flag.String("token", "", "Set the Configured KV Workers key instead of using any preferences file")
	var addLocationFlag = flag.String("addLocation", "", "Add these locations/files to backup in addition to those set in the preferences file")
	var locationFlag = flag.String("location", "", "Use only these locations to backup")
	//	var backupFlag = flag.String("backup", "", "Backup strategy")
	var downloadFlag = flag.String("download", "", "Folder/files to download")
	var zipFlag = flag.String("zip", "", "Set the zip compression to 'none', 'zstandard', or 'zip'")
	var verboseFlag = flag.Bool("v", false, "More information")
	var altPrefFlag = flag.String("pref", "", "use an alternate preference file")
	var dryRunFlag = flag.Bool("dryrun", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var dryFlag = flag.Bool("dry", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var getKeysFlag = flag.Bool("keys", false, "Get the keys and metadata from cloudflare")
	var debugFlag = flag.Bool("debug", false, "Debugging information is shown")
	var synchFlag = flag.Bool("synch", false, "Download the keys and metadata from cloud and overwrite the local database")
	//	var forcePrefFlag = flag.String("f", "", "ignore the lock")

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
	if *debugFlag {
		gobackup.Debug = true
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
	//	if *backupFlag != "" {
	//		cf.Backup = *backupFlag
	//	}
	if *zipFlag != "" {
		cf.Zip = *zipFlag
	}
	if *verboseFlag {
		Verbose = true
		gobackup.Verbose = true
		fmt.Println("Verbose information is true, opening the flood gates!")
	}
	if *dryRunFlag || *dryFlag {
		gobackup.DryRun = true
		fmt.Println("Dry Run is active! No changes will be made!")
	}
	if *downloadFlag != "" {
		cf.Location = *downloadFlag //hash

		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}

		gobackup.DownloadKV(&cf, *downloadFlag, "download.file")
		fmt.Println("Downloaded a file!")
		os.Exit(0)
	}
	if *getKeysFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		fmt.Println("Getting the keys and metadata!")
		gobackup.GetKVkeys(&cf)
		os.Exit(0)
	}

	if *synchFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		fmt.Println("Getting the keys and metadata!")
		jsonKeys := gobackup.GetKVkeys(&cf)
		fmt.Printf("jsonKeys:%s", jsonKeys)

		var extractedData gobackup.CloudflareResponse

		json.Unmarshal(jsonKeys, &extractedData)

		fmt.Printf("Data extracted******: %v \n******", extractedData)
		fmt.Printf("success: %v\n", extractedData.Success)
		if len(extractedData.Result) != 0 {
			if Verbose {
				fmt.Printf("size of result:%v\n", len(extractedData.Result))
				fmt.Printf("result: %v \n\n", extractedData.Result)
			}

			fmt.Println("Adding extracted keys and metadata to data1 struct")
			for i := 0; i < len(extractedData.Result); i++ {
				//update the data struct
				dat.TheMetadata = append(dat.TheMetadata, extractedData.Result[i].TheMetadata)
				fmt.Println("Added " + extractedData.Result[i].TheMetadata.FileName)
			}

			fmt.Printf("Size of the data1 metadata array: %v\n", len(dat.TheMetadata))
			fmt.Printf("dat: %v\n\n\n", dat.TheMetadata)
			//			sort.Sort(gobackup.ByHash(dat.TheMetadata))

			gobackup.DataFile2("data.dat", &dat)

		} else {
			fmt.Println("Empty Result")
		}
		os.Exit(0)
	}
}

//search the data file for hash, line by line
//hash is the hash from a file, fileName is a file
//in the future, may introduce a binary search or a simple index for a pre-sorted data file
func searchData(hash string, fileName string) bool {

	file, err := os.Open("data.dat")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Print("data.dat file Does Not Exist. Creating a new data.dat file!")
			file, err = os.Create("data.dat")
			gobackup.CheckError(err, "searchData failed to create data.dat!")
		} else {
			gobackup.CheckError(err, "searchData failed opening file:data.dat")
		}
	}

	defer file.Close()

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
	//	validatePreferences()
	gobackup.ValidateCF(&cf)

	//prevent other local gobackup instances from altering critical files
	gobackup.AddLock()

	//the filelist for backup
	var fileList []string

	if Verbose {
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

		if gobackup.Debug {
			openFile, _ := os.Open(f)
			contents, _ := ioutil.ReadAll(openFile)
			fmt.Printf("For Loop: (f:%v)(hash:%v)(contents:%v)", f, hash, string(contents))
		}
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

	if gobackup.DryRun {
		fmt.Println("Dry Run dataFile2 is running!")
		gobackup.DataFile2("", &dat)
	} else {
		gobackup.DataFile2("data.dat", &dat)
	}

	//remove locks
	gobackup.DeleteLock()

} //main
