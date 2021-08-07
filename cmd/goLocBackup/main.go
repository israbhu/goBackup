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

//***************Info*************************
//	glog.V(1) represents verbose information, which is extra information non-essential to keep users updated
//	glog.V(2) represents debug information, which is extra information useful to debug the application

//***************global variables*************
//var cf gobackup.Account        //account credentials and preferences
var dat gobackup.DataContainer //local datastore tracking uploads and Metadata
var preferences string         //the location of preferences file
var homeDirectory string       //use this location to resolve pathing instead of the PWD

/* NOTES *
filepath == filepath same file
base filepath == folder (filepath minus filename)

downloading a folder = get all files with same base filepath
downloading a folder and all subfolders = get all files that start with the same base filepath
download a specific file (use the listAllFiles to find the hash?)


*/
//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup(p programParameters) {
	for _, list := range dat.TheMetadata {
		gobackup.UploadMultiPart(&p.Account, list)
	}
}

func readPreferencesFile(p programParameters) error {
	err := readTOML(p, p.preferencesFile, p.Account)

	//preferences file is not necessary, so only a warning given to user
	gobackup.NoErrorFound(err, "readTOML has encountered an error while attempting to open the preferences file. Program will continue to run but may be unstable if all the necessary command line options are not used.")

	return err
}

//read from a toml file
//check that the file is accessible since the function can be called from a commandline argument
// The content of iface is undefined when the returned error is not nil.
func readTOML(p programParameters, file string, iface interface{}) error {
	path, err := gobackup.MakeCanonicalPath(file)
	if err != nil {
		return fmt.Errorf("While getting the canonical path for '%s': %v", file, err)
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("While calling stat on file '%s': %v", path, err)
	}

	glog.Infof("Reading in the toml file, '%s'", path)
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("While reading file '%s': %v", path, err)
	}

	glog.V(1).Infof("Parsing preferences TOML: '%s'", path)
	if err := toml.Unmarshal(dat, iface); err != nil {
		return fmt.Errorf("While reading in the TOML file '%s': %v", path, err)
	}
	glog.V(1).Infof("Parsed '%s' to: %+v", path, p)

	return nil
}

//write to a toml file
func writeTOML(file string, data *gobackup.DataContainer) {
	doc, _ := toml.Marshal(data)

	err := ioutil.WriteFile(file, doc, 0644)
	if err != nil {
		glog.Fatalf("Error with writeTOML! Cannot write the file %v %v", file, err)
	}
	_, err = os.Lstat(file)
	if err != nil {
		glog.Fatalf("Error with writeTOML! Cannot stat the file: %v %v", file, err)
	}

}

//parameters
//name is the drive path to a folder
//f is a slice that contains all of the accumulated files
//the return []string is the slice f
// When error is not nil, f is not valid.
func getFiles(name string, f *[]string) error {
	//make sure name is valid
	if name == "" {
		glog.Infoln("getFiles name is blank, getting files from local directory")
		name = "."
	}

	path, err := gobackup.MakeCanonicalPath(name)
	if err != nil {
		return fmt.Errorf("While getting canonical path for '%s': %v", name, err)
	}
	if fi, err := os.Stat(path); err == nil {
		//if it's a regular file, append and return f
		if fi.Mode().IsRegular() {
			*f = append(*f, path)
			return nil
		}
	} else {
		return fmt.Errorf("getFiles has encountered a problem with os.Stat(%s): %v", path, err)
	}

	glog.V(2).Infof("getFiles name=**%v**\n", name)
	err = filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if !gobackup.NoErrorFound(err, "") {
			basedir, _ := os.Getwd()

			gobackup.DeleteLock()
			return fmt.Errorf("Cannot find %v. Closing the program!: %v", filepath.Join(basedir, path), err)
		}
		//remove .
		if path != "." && !info.IsDir() {
			*f = append(*f, path)
		}
		return nil
	})
	if !gobackup.NoErrorFound(err, "The filepath.Walk encountered an error. Returning without modifying the file list") {
		return err
	}
	return nil
}

// 	Namespace, // namespace is called the "namespace id" on the cloudflare website for Workers KV
// 	Account, // account is called "account id" on the cloudflare dashboard
// 	Key, // key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
// 	Token, // Token is used instead of the key (More secure than using Key) and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
// 	Email, // email is the email associated with your cloudflare account
// 	Data,
// 	Location, //locations of the comma deliminated file and folders to be backed up
// 	DownloadLocation, //default location to download data
// 	Zip, //string to determine zip type. Currently none, zstandard, and zip
// 	Backup string
type programParameters struct {
	addLocation     string
	dryRun          bool
	homeDirectory   string
	preferencesFile string
	gobackup.Account
}

//process the command line commands
//yes, email, Account, Data, Email, Namespace, Key, Token, Location string
//backup strategy, zip, encrypt, verbose, sync, list data, alt pref, no pref
func extractCommandLine() programParameters {
	var p programParameters
	flag.StringVar(&p.Email, "email", "", "Set the User email instead of using any preferences file")
	flag.StringVar(&p.AccountID, "account", "", "Set the User Account instead of using any preferences file")
	flag.StringVar(&p.Namespace, "namespace", "", "Set the User's Namespace instead of using any preferences file")
	flag.StringVar(&p.Key, "key", "", "Set the Account Global Key instead of using any preferences file")
	flag.StringVar(&p.Token, "token", "", "Set the Configured KV Workers key instead of using any preferences file")
	flag.StringVar(&p.addLocation, "addLocation", "", "Add these locations/files to backup in addition to those set in the preferences file")
	flag.StringVar(&p.Location, "location", "", "Use only these locations to backup")
	// FIXME Duplicate? flag.String(&p.Location, "download", "", "Download files. By default use the preferences location. Use -location and -addLocation to modify the files downloaded.")
	flag.StringVar(&p.Zip, "zip", "", "Set the zip compression to 'none', 'zstandard', or 'zip'")
	flag.StringVar(&p.preferencesFile, "pref", "", "use an alternate preference file")
	flag.BoolVar(&p.dryRun, "dryrun", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")
	flag.StringVar(&p.homeDirectory, "home", "", "Change your home directory. All relative paths based on home directory")

	// TODO Make these command arguments instead of separate options to the program.
	var getKeysFlag = flag.Bool("keys", false, "Get the keys and metadata from cloudflare")
	var syncFlag = flag.Bool("sync", false, "Download the keys and metadata from cloud and overwrite the local database")
	var searchFlag = flag.String("search", "", "Search the local database and print to screen")
	var listAllFlag = flag.String("listAllFiles", "", "List all files in the local database")
	var listRecentFlag = flag.String("listRecentFiles", "", "List most recent files in the local database")
	var saveFlag = flag.String("save", "", "save the current data queue to the save file")

	flag.Set("logtostderr", "true")
	flag.Parse()
	glog.Infoln("Checking flags!")

	if p.preferencesFile != "" {
		glog.Infoln("Alternate Preferences file detected, checking:")

		readPreferencesFile(p)
		// 		if err := gobackup.ValidateCF(&p.Account); err != nil {
		// 			gobackup.DeleteLock()
		// 			glog.Fatalf("The preferences file at '%s' has errors that need to be fixed!: %v", gobackup.MustMakeCanonicalPath(*altPrefFlag), err)
		// 		}
	}

	if *searchFlag != "" {
		gobackup.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

		glog.Info("Displaying sorted dat")
		sort.Sort(gobackup.ByFilepath(dat.TheMetadata))

		glog.Infof("%+v", dat.TheMetadata)

		//save the data container to the file indicated by *saveFlag
		writeTOML(*saveFlag, &dat)

		os.Exit(0)
	}

	if *listAllFlag != "" {
		gobackup.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

		glog.Info("Displaying sorted dat")
		sort.Sort(gobackup.ByFilepath(dat.TheMetadata))

		glog.Info("Listing Files by key, filename, and last modified time")
		for i := 0; i < len(dat.TheMetadata); i++ {
			//			glog.Info(dat.TheMetadata[i].Hash + " " + dat.TheMetadata[i].FileName + " " + dat.TheMetadata[i].Mtime.String())
			glog.Info(dat.TheMetadata[i].Hash + " " + dat.TheMetadata[i].FileName + " " + dat.TheMetadata[i].Filepath + " " + dat.TheMetadata[i].Mtime.String())
		}

		//save the data container to the file indicated by *saveFlag
		writeTOML(*saveFlag, &dat)

		//		fmt.Println(dat)
		gobackup.DeleteLock()
		os.Exit(0)
	}

	if *listRecentFlag != "" {
		gobackup.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

		glog.Info("Displaying sorted dat")
		sort.Sort(gobackup.ByFilepath(dat.TheMetadata))

		glog.Infof("%v metadata in the container after sorting\n", len(dat.TheMetadata))

		var prune []gobackup.Metadata

		//prune the old files
		for i := 0; i < len(dat.TheMetadata)-1; i++ {
			prune = append(prune, dat.TheMetadata[i])

			test := dat.TheMetadata[i].FileName

			//TODO decide if we need filePath, fileName, or something else
			//prune out any similar filepaths
			for test == dat.TheMetadata[i+1].FileName && i < len(dat.TheMetadata)-2 {
				glog.Infof("test%v:%v vs test%v:%v", i, test, i+1, dat.TheMetadata[i+1].FileName)
				i++
			}
		}

		dat.TheMetadata = prune

		glog.Infof("%v metadata in the container after pruning\n", len(dat.TheMetadata))

		glog.Info("Listing Files by key, filename, and last modified time")
		for i := 0; i < len(dat.TheMetadata); i++ {
			glog.Info(dat.TheMetadata[i].Hash + " " + dat.TheMetadata[i].FileName + " " + dat.TheMetadata[i].Filepath + " " + dat.TheMetadata[i].Mtime.String())
		}

		//save the data container to the file indicated by *saveFlag
		writeTOML(*saveFlag, &dat)

		//		fmt.Println(dat)
		gobackup.DeleteLock()
		os.Exit(0)
	}

	if p.addLocation != "" { //add to the locations
		p.Location = p.Location + "," + p.addLocation
	}
	glog.V(1).Infoln("Verbose information is true, opening the flood gates!")
	if p.dryRun {
		gobackup.DryRun = true
		glog.Infoln("Dry Run is active! No changes will be made!")
	}
	if p.Location != "" {
		if !gobackup.DryRun {
			//check account
			if err := gobackup.ValidateCF(&p.Account); err != nil {
				gobackup.DeleteLock()
				glog.Fatalf("%v", err)
			}
		}

		//**** NEEEDS WORK!!****
		//download the data
		glog.Infoln(gobackup.DownloadKV(&p.Account, dat.TheMetadata[0].Hash, "test.txt"))

		gobackup.DownloadKV(&p.Account, p.Location, "download.file")
		glog.Infoln("Downloaded a file!")
		os.Exit(0)
	}
	if *getKeysFlag {
		if !gobackup.DryRun {
			//check account
			if err := gobackup.ValidateCF(&p.Account); err != nil {
				gobackup.DeleteLock()
				glog.Fatalf("%v", err)
			}
		}
		//get keys
		glog.Infoln("Getting the keys and metadata!")
		glog.Infoln(string(gobackup.GetKVkeys(&p.Account)))
		os.Exit(0)
	}

	if *syncFlag {
		if !gobackup.DryRun {
			//check account
			if err := gobackup.ValidateCF(&p.Account); err != nil {
				gobackup.DeleteLock()
				glog.Fatalf("%v", err)
			}
		}
		//get keys
		glog.Infoln("Getting the keys and metadata!")
		jsonKeys := gobackup.GetKVkeys(&p.Account)
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
	return p
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
			if !gobackup.NoErrorFound(err, "searchData failed to create data.dat!") {
				glog.Fatal("Closing program!")
			}
		} else {
			gobackup.NoErrorFound(err, "searchData failed opening file:data.dat")
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

//add a 0byte data entry to cloudflare workersKV.
//The metadata of the file is uploaded to the hashContentAndMeta key.
//It retains the correct metadata extracted from the file
//The foreign key in the metadata contains the key to the content entry
func populateFK(dat *gobackup.DataContainer, meta *gobackup.Metadata, hashContentAndMeta string) {

	metaFK := *meta
	metaFK.Hash = hashContentAndMeta
	metaFK.ForeignKey = meta.Hash

	glog.Infoln("CONTENT HASH FOUND, FOREIGN KEY NOT FOUND! Adding key:" + hashContentAndMeta + "-" + gobackup.MetadataToString(metaFK))

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

	glog.V(2).Infoln("NOT FOUND AND INCLUDING! " + meta.Hash + "-fkhash " + metaFK.ForeignKey + " metadata " + gobackup.MetadataToString(metaFK))

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
	p := extractCommandLine()

	// lock()
	// defer unlock()
	// switch cmd {
	// case "search":
	// 	search()
	// case "listMostRecent":
	// 	listMostRecent()
	// 	//....
	// default:
	// 	glog.Fatalf("You messed up. Here is the usage of this program")

	// }

	if err := gobackup.ValidateCF(&p.Account); err != nil {
		glog.Fatalf("Account information is not valid: %v", err)
	}

	pref, _ := filepath.Abs(p.preferencesFile)

	gobackup.ChangeHomeDirectory(&p.Account)
	fmt.Printf("Checking that directory is below home! Directory:%v result:%t", pref, gobackup.CheckPath(pref, &p.Account))

	//prevent other local gobackup instances from altering critical files
	gobackup.AddLock()
	defer gobackup.DeleteLock()

	//the filelist for backup
	var fileList []string

	glog.V(1).Infof("CF LOCATION:%v", p.Location)

	backupLocations := strings.Split(p.Location, ",")

	for _, l := range backupLocations {
		if err := getFiles(strings.TrimSpace(l), &fileList); err != nil {
			glog.Errorf("Fatal error while getting files to back up: %v", err)
			return
		}
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

		} else { //all Hashes for the file were in the local database. Exclude from uploading
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
	backup(p)

	//update the local data file
	if gobackup.DryRun {
		glog.Infoln("Dry Run dataFile2 is running!")
		gobackup.DataFile2("", &dat)
	} else {
		gobackup.DataFile2("data.dat", &dat)
	}
} //main
