package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

//read from a toml file
//check that the file exists since the function can be called from a commandline argument
func readTOML(file string) {

	if gobackup.FileExist(file) {

		gobackup.Logger.Printf("Reading in the toml file, %v", file)
		dat, err := ioutil.ReadFile(file)
		gobackup.CheckError(err, "")

		astring := string(dat)

		doc2 := []byte(astring)

		if file == "data.dat" {
			toml.Unmarshal(doc2, &dat)
			//verbose flag
			if Verbose {
				gobackup.Logger.Println("Reading in the data file:" + file)
				gobackup.Logger.Println(dat)
			}

		} else {
			toml.Unmarshal(doc2, &cf)

			//verbose flag
			if Verbose {
				gobackup.Logger.Println("Reading in the preference file:" + file)
				gobackup.Logger.Println(cf)
			}
		}
	} else { //the file does not exist
		workingDirectory, _ := os.Getwd()

		gobackup.Logger.Output(1, "The file, "+workingDirectory+string(os.PathSeparator)+file+" does not exist! We strongly recommend creating the file for higher efficiency.")
	} //else

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
		gobackup.Logger.Print("not exists")
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
		gobackup.Logger.Println("getFiles name is blank, getting files from local directory")
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

	gobackup.Logger.Printf("getFiles name=**%v**\n", name)
	//	fmt.Printf("my cf email is %v", cf.Email)
	//	log.Fatalln("Stopping here for checkup")
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if !gobackup.CheckError(err, "") {
			basedir, _ := os.Getwd()

			gobackup.Logger.Fatalf("Cannot find %v. Closing the program!", basedir+string(os.PathSeparator)+path)
		}
		//remove .
		if path != "." && !info.IsDir() {
			f = append(f, path)
		}
		return nil
	})
	if !gobackup.CheckError(err, "") {
		gobackup.Logger.Fatalf("Error found: %v", err)
	}
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
	var downloadFlag = flag.String("download", "", "Download files. By default use the preferences location. Use -location and -addLocation to modify the files downloaded.")
	var zipFlag = flag.String("zip", "", "Set the zip compression to 'none', 'zstandard', or 'zip'")
	var verboseFlag = flag.Bool("v", false, "More information")
	var altPrefFlag = flag.String("pref", "", "use an alternate preference file")
	var dryRunFlag = flag.Bool("dryrun", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var dryFlag = flag.Bool("dry", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var getKeysFlag = flag.Bool("keys", false, "Get the keys and metadata from cloudflare")
	var debugFlag = flag.Bool("debug", false, "Debugging information is shown")
	var syncFlag = flag.Bool("sync", false, "Download the keys and metadata from cloud and overwrite the local database")
	//	var forcePrefFlag = flag.String("f", "", "ignore the lock")

	flag.Parse()
	gobackup.Logger.Println("Checking flags!")

	if *altPrefFlag != "" {
		gobackup.Logger.Println("Alernate Preferences file detected, checking:")
		readTOML(*altPrefFlag)
		if !gobackup.ValidateCF(&cf) {
			gobackup.Logger.Printf("%v has errors that need to be fixed!", *altPrefFlag)
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
		gobackup.Logger.Println("Verbose information is true, opening the flood gates!")
	}
	if *dryRunFlag || *dryFlag {
		gobackup.DryRun = true
		gobackup.Logger.Println("Dry Run is active! No changes will be made!")
	}
	if *downloadFlag != "" {
		cf.Location = *downloadFlag //hash

		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}

		//**** NEEEDS WORK!!****
		//download the data
		gobackup.Logger.Println(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

		gobackup.DownloadKV(&cf, *downloadFlag, "download.file")
		gobackup.Logger.Println("Downloaded a file!")
		os.Exit(0)
	}
	if *getKeysFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		gobackup.Logger.Println("Getting the keys and metadata!")
		gobackup.Logger.Println(string(gobackup.GetKVkeys(&cf)))
		os.Exit(0)
	}

	if *syncFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		gobackup.Logger.Println("Getting the keys and metadata!")
		jsonKeys := gobackup.GetKVkeys(&cf)
		gobackup.Logger.Printf("jsonKeys:%s", jsonKeys)

		var extractedData gobackup.CloudflareResponse

		json.Unmarshal(jsonKeys, &extractedData)

		gobackup.Logger.Printf("Data extracted******: %v \n******", extractedData)
		gobackup.Logger.Printf("success: %v\n", extractedData.Success)
		if len(extractedData.Result) != 0 {
			if Verbose {
				gobackup.Logger.Printf("size of result:%v\n", len(extractedData.Result))
				gobackup.Logger.Printf("result: %v \n\n", extractedData.Result)
			}

			gobackup.Logger.Println("Adding extracted keys and metadata to data1 struct")
			for i := 0; i < len(extractedData.Result); i++ {
				//update the data struct
				dat.TheMetadata = append(dat.TheMetadata, extractedData.Result[i].TheMetadata)
				gobackup.Logger.Println("Added " + extractedData.Result[i].TheMetadata.FileName)
			}

			gobackup.Logger.Printf("Size of the data1 metadata array: %v\n", len(dat.TheMetadata))
			gobackup.Logger.Printf("dat: %v\n\n\n", dat.TheMetadata)
			//			sort.Sort(gobackup.ByHash(dat.TheMetadata))

			gobackup.DataFile2("data.dat", &dat)

		} else {
			gobackup.Logger.Println("Empty Result")
		}
		os.Exit(0)
	}
}

//search the data file for hash, line by line
//hash is the hash from a file, fileName is a file
//in the future, may introduce a binary search or a simple index for a pre-sorted data file
func searchData(hash string) bool {

	file, err := os.Open("data.dat")
	if err != nil {
		if os.IsNotExist(err) {
			gobackup.Logger.Print("data.dat file Does Not Exist. Creating a new data.dat file!")
			file, err = os.Create("data.dat")
			if !gobackup.CheckError(err, "searchData failed to create data.dat!") {
				gobackup.Logger.Fatalln("Closing program!")
			}
		} else {
			gobackup.CheckError(err, "searchData failed opening file:data.dat")
		}
	}

	defer file.Close()

	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		line := scan.Text()                //get lines of data.dat
		column := strings.Split(line, ":") //split out the data

		if column[0] == hash {
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

func populatePayloadAndMeta(dat *gobackup.Data1, meta *gobackup.Metadata, hashContentAndMeta string) {
	metaFK := *meta
	metaFK.ForeignKey = meta.Hash
	metaFK.Hash = hashContentAndMeta

	gobackup.Logger.Println("NOT FOUND AND INCLUDING! " + meta.Hash + "-fkhash " + metaFK.ForeignKey + " metadata " + gobackup.GetMetadata(metaFK))

	//update the data struct with the content
	dat.TheMetadata = append(dat.TheMetadata, *meta)

	//update the data struct with the foreign key
	dat.TheMetadata = append(dat.TheMetadata, metaFK)

	sort.Sort(gobackup.ByHash(dat.TheMetadata))

	gobackup.Logger.Printf("Checking For foreign key: %v\n*******end FK check", dat.TheMetadata)

	dat.DataSize += meta.Size
	dat.Count += 1
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
	defer gobackup.DeleteLock()

	//the filelist for backup
	var fileList []string

	if Verbose {
		gobackup.Logger.Printf("CF LOCATION:%v", cf.Location)
	}

	backupLocations := strings.Split(cf.Location, ",")

	for _, l := range backupLocations {
		fileList = getFiles(strings.TrimSpace(l), fileList)
	}

	sort.Strings(fileList)

	//fill in the Metadata
	for _, f := range fileList {
		hash := gobackup.Md5file(f)
		hashContentAndMeta := gobackup.Md5fileAndMeta(f)

		//debug info
		if gobackup.Debug {
			openFile, _ := os.Open(f)
			contents, _ := ioutil.ReadAll(openFile)
			gobackup.Logger.Printf("For Loop: (f:%v)(hash:%v)(contents:%v)", f, hash, string(contents))
		}
		//if content hash not found, need two entries 1, content hash, 2, content and meta hash
		if !searchData(hash) {
			meta := gobackup.CreateMeta(f)
			populatePayloadAndMeta(&dat, &meta, hashContentAndMeta)
		} else if !searchData(hashContentAndMeta) { //content hash was found, so now check for content and meta hash
			metaFK := gobackup.CreateMeta(f)
			metaFK.Hash = hashContentAndMeta
			metaFK.ForeignKey = hash
			gobackup.Logger.Println("CONTENT HASH FOUND, FOREIGN KEY NOT FOUND! Adding key:" + hashContentAndMeta + "-" + gobackup.GetMetadata(metaFK))

			//update the data struct with the foreign key
			dat.TheMetadata = append(dat.TheMetadata, metaFK)

			sort.Sort(gobackup.ByHash(dat.TheMetadata))

		} else {
			gobackup.Logger.Println("FOUND AND EXCLUDING! " + hash)
		}
	} //for

	//TheMetadata is empty, then there is no work left to be done. Exit program
	if len(dat.TheMetadata) == 0 {
		gobackup.Logger.Println("All files are up to date! Exiting!")
		gobackup.DeleteLock()
		os.Exit(0)
	}

	//information for user
	gobackup.Logger.Printf("Original File: %s, Data Size: %v, Data Count: %v", dat.TheMetadata[0].FileName, dat.DataSize, dat.Count)

	//split the work and backup
	backup()

	//download the data
	//	fmt.Println(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

	//update the local data file

	if gobackup.DryRun {
		gobackup.Logger.Println("Dry Run dataFile2 is running!")
		gobackup.DataFile2("", &dat)
	} else {
		gobackup.DataFile2("data.dat", &dat)
	}
} //main
