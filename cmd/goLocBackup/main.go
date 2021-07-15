package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/israbhu/goBackup/internal/pkg/gobackup"
	"github.com/pelletier/go-toml"
)

//***************global variables*************
var cf gobackup.Account        //account credentials and preferences
var dat gobackup.DataContainer //local datastore tracking uploads and Metadata

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

		glog.Infof("Reading in the toml file, %v", file)
		dat, err := ioutil.ReadFile(file)
		gobackup.CheckError(err, "")

		astring := string(dat)

		doc2 := []byte(astring)

		if file == "data.dat" {
			toml.Unmarshal(doc2, &dat)
			glog.V(1).Infoln("Reading in the data file:" + file)
			glog.V(1).Infoln(dat)

		} else {
			toml.Unmarshal(doc2, &cf)

			glog.V(1).Infoln("Reading in the preference file:" + file)
			glog.V(1).Infoln(cf)
		}
	} else { //the file does not exist
		workingDirectory, _ := os.Getwd()

		glog.Warningf("The file '%s' does not exist! We strongly recommend creating the file for higher efficiency.", filepath.Join(workingDirectory, file))
	} //else

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
		err := os.Mkdir(name, 0755)
		gobackup.CheckError(err, "")
	}
}

func resolvePath(file string) string {
	path, err := filepath.Abs(file)
	if err != nil {
		glog.Fatalf("resovePath has encountered an error: %v", err)
	}

	return path
}

//parameters
//name is the drive path to a folder
//f is a slice that contains all of the accumulated files
//return []string
//the return []string is the slice f
func getFiles(name string, f []string) []string {
	//make sure name is valid
	if name == "" {
		glog.Infoln("getFiles name is blank, getting files from local directory")
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

	glog.V(2).Infof("getFiles name=**%v**\n", name)
	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if !gobackup.CheckError(err, "") {
			basedir, _ := os.Getwd()

			glog.Fatalf("Cannot find %v. Closing the program!", filepath.Join(basedir, path))
		}
		//remove .
		if path != "." && !info.IsDir() {
			f = append(f, path)
		}
		return nil
	})
	if !gobackup.CheckError(err, "") {
		glog.Fatalf("Error found: %v", err)
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
	var altPrefFlag = flag.String("pref", "", "use an alternate preference file")
	var dryRunFlag = flag.Bool("dryrun", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var dryFlag = flag.Bool("dry", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	var getKeysFlag = flag.Bool("keys", false, "Get the keys and metadata from cloudflare")
	var syncFlag = flag.Bool("sync", false, "Download the keys and metadata from cloud and overwrite the local database")
	//	var forcePrefFlag = flag.String("f", "", "ignore the lock")

	flag.Parse()
	glog.Infoln("Checking flags!")

	if *altPrefFlag != "" {
		glog.Infoln("Alernate Preferences file detected, checking:")
		readTOML(*altPrefFlag)
		if !gobackup.ValidateCF(&cf) {
			glog.Infof("%v has errors that need to be fixed!", *altPrefFlag)
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
	//	if *backupFlag != "" {
	//		cf.Backup = *backupFlag
	//	}
	if *zipFlag != "" {
		cf.Zip = *zipFlag
	}
	glog.V(1).Infoln("Verbose information is true, opening the flood gates!")
	if *dryRunFlag || *dryFlag {
		gobackup.DryRun = true
		glog.Infoln("Dry Run is active! No changes will be made!")
	}
	if *downloadFlag != "" {
		cf.Location = *downloadFlag //hash

		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}

		//**** NEEEDS WORK!!****
		//download the data
		glog.Infoln(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

		gobackup.DownloadKV(&cf, *downloadFlag, "download.file")
		glog.Infoln("Downloaded a file!")
		os.Exit(0)
	}
	if *getKeysFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		glog.Infoln("Getting the keys and metadata!")
		glog.Infoln(string(gobackup.GetKVkeys(&cf)))
		os.Exit(0)
	}

	if *syncFlag {
		if !gobackup.DryRun {
			//check account
			gobackup.ValidateCF(&cf)
		}
		//get keys
		glog.Infoln("Getting the keys and metadata!")
		jsonKeys := gobackup.GetKVkeys(&cf)
		glog.Infof("jsonKeys:%s", jsonKeys)

		var extractedData gobackup.CloudflareResponse

		json.Unmarshal(jsonKeys, &extractedData)

		glog.Infof("Data extracted******: %v \n******", extractedData)
		glog.Infof("success: %v\n", extractedData.Success)
		if len(extractedData.Result) != 0 {
			glog.V(1).Infof("size of result:%v\n", len(extractedData.Result))
			glog.V(1).Infof("result: %v \n\n", extractedData.Result)

			glog.Infoln("Adding extracted keys and metadata to data1 struct")
			for i := 0; i < len(extractedData.Result); i++ {
				//update the data struct
				dat.TheMetadata = append(dat.TheMetadata, extractedData.Result[i].TheMetadata)
				glog.Infoln("Added " + extractedData.Result[i].TheMetadata.FileName)
			}

			glog.Infof("Size of the data1 metadata array: %v\n", len(dat.TheMetadata))
			glog.Infof("dat: %v\n\n\n", dat.TheMetadata)
			//			sort.Sort(gobackup.ByHash(dat.TheMetadata))

			gobackup.DataFile2("data.dat", &dat)

		} else {
			glog.Infoln("Empty Result")
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
			glog.V(1).Info("data.dat file Does Not Exist. Creating a new data.dat file!")
			file, err = os.Create("data.dat")
			if !gobackup.CheckError(err, "searchData failed to create data.dat!") {
				glog.Fatal("Closing program!")
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

//******* This struct contains the data needed to access the cloudflare infrastructure. It is stored on drive in the file preferences.toml *****
type Account struct {
	//cloudflare account information
	Namespace, // namespace is called the "namespace id" on the cloudflare website for Workers KV
	Account, // account is called "account id" on the cloudflare dashboard
	Key, // key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Token, // Token is used instead of the key (More secure than using Key) and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	Email, // email is the email associated with your cloudflare account
	Data,
	Location, //locations of the comma deliminated file and folders to be backed up
	DownloadLocation, //default location to download data
	Zip, //string to determine zip type. Currently none, zstandard, and zip
	Backup string
}

//add a 0byte data entry to cloudflare workersKV.
//The metadata of the file is uploaded to the hashContentAndMeta key.
//It retains the correct metadata extracted from the file
//The foreign key in the metadata contains the key to the content entry
func populateFK(dat *gobackup.DataContainer, meta *gobackup.Metadata, hashContentAndMeta string) {

	metaFK := *meta
	metaFK.Hash = hashContentAndMeta
	metaFK.ForeignKey = meta.Hash

	glog.Infoln("CONTENT HASH FOUND, FOREIGN KEY NOT FOUND! Adding key:" + hashContentAndMeta + "-" + gobackup.GetMetadata(metaFK))

	//update the data struct with the foreign key
	dat.TheMetadata = append(dat.TheMetadata, metaFK)
	dat.Count++

	sort.Sort(gobackup.ByHash(dat.TheMetadata))
}

//add a the content data entry to cloudflare workersKV. Also add a 0 byte data entry
//The metadata of the file is uploaded to the hashContentAndMeta key.
//It retains the correct metadata extracted from the file
//The foreign key in the metadata contains the key to the content entry
func populatePayloadAndMeta(dat *gobackup.DataContainer, meta *gobackup.Metadata, hashContentAndMeta string) {

	//TODO check on CF_MAX_UPLOAD and CF_MAX_DATA_UPLOAD, CF_MAX_DATA_FILE

	metaFK := *meta
	metaFK.ForeignKey = meta.Hash
	metaFK.Hash = hashContentAndMeta

	glog.V(2).Infoln("NOT FOUND AND INCLUDING! " + meta.Hash + "-fkhash " + metaFK.ForeignKey + " metadata " + gobackup.GetMetadata(metaFK))

	//update the data struct with the content
	dat.TheMetadata = append(dat.TheMetadata, *meta)

	//update the data struct with the foreign key
	dat.TheMetadata = append(dat.TheMetadata, metaFK)

	sort.Sort(gobackup.ByHash(dat.TheMetadata))

	glog.V(2).Infof("Checking For foreign key: %v\n*******end FK check", dat.TheMetadata)
	//update the dat information (need to check if we exceed the capacities of the account, see TODO above)
	dat.DataSize += meta.Size
	dat.Count += 2
}

func main() {

	//command line can overwrite the data from the preferences file
	extractCommandLine()

	//if no alternate prefences were in the command line, extract the default
	if cf.Token == "" {
		readTOML("preferences.toml")
		fmt.Printf("preferences resolved to:%v", resolvePath("preferences.toml"))
	}

	//make sure the preferences are valid
	//	validatePreferences()
	gobackup.ValidateCF(&cf)

	//prevent other local gobackup instances from altering critical files
	gobackup.AddLock()
	defer gobackup.DeleteLock()

	//the filelist for backup
	var fileList []string

	glog.V(1).Infof("CF LOCATION:%v", cf.Location)

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
		if glog.V(2) {
			openFile, _ := os.Open(f)
			contents, _ := ioutil.ReadAll(openFile)
			glog.Infof("For Loop: (f:%v)(hash:%v)(contents:%v)", f, hash, string(contents))
		}
		//if content hash not found, need two entries 1, content hash, 2, content and meta hash
		if !searchData(hash) {
			meta := gobackup.CreateMeta(f)
			populatePayloadAndMeta(&dat, &meta, hashContentAndMeta)
		} else if !searchData(hashContentAndMeta) { //content hash was found, so now check for content and meta hash
			meta := gobackup.CreateMeta(f)
			populateFK(&dat, &meta, hashContentAndMeta)

		} else {
			glog.V(1).Infoln("FOUND AND EXCLUDING! " + f + " " + hash)
		}
	} //for

	//TheMetadata is empty, then there is no work left to be done. Exit program
	if len(dat.TheMetadata) == 0 {
		glog.Infoln("All files are up to date! Exiting!")
		gobackup.DeleteLock()
		os.Exit(0)
	}

	//information for user
	glog.Infof("Backing up %v Files, Data Size: %v", dat.Count, dat.DataSize)

	//split the work and backup
	backup()

	//update the local data file
	if gobackup.DryRun {
		glog.Infoln("Dry Run dataFile2 is running!")
		gobackup.DataFile2("", &dat)
	} else {
		gobackup.DataFile2("data.dat", &dat)
	}
} //main
