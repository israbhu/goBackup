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
	"github.com/pelletier/go-toml"

	"github.com/israbhu/goBackup/internal/pkg/cf"
	"github.com/israbhu/goBackup/internal/pkg/gobackup"
)

//***************Info*************************
//	glog.V(1) represents verbose information, which is extra information non-essential to keep users updated
//	glog.V(2) represents debug information, which is extra information useful to debug the application

//***************types*************
type commandEntry struct {
	help string
	fn   func(p *programParameters) error
}

//***************global variables*************
var (
	commands = map[string]commandEntry{
		"keys":            {help: "Get the keys and metadata from cloudflare", fn: doGetKeys},
		"sync":            {help: "Download the keys and metadata from cloud and overwrite the local database", fn: doSync},
		"search":          {help: "Search the local database and print to screen"},
		"listAllFiles":    {help: "List all files in the local database", fn: doListAll},
		"listRecentFiles": {help: "List most recent files in the local database", fn: doListRecent},
		"upload":          {help: "Uploads the locations indicated via preferences or commandline", fn: doUpload},
	}
	dat           gobackup.DataContainer //local datastore tracking uploads and Metadata
	preferences   string                 //the location of preferences file
	homeDirectory string                 //use this location to resolve pathing instead of the PWD
)

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
		cf.UploadMultiPart(&p.Account, list)
	}
}

// Reads the preferences file at the given path.
// Returns an account, which may be empty if there was a problem reading the
// file.
func readPreferencesFile(path string) cf.Account {

	var acct cf.Account
	err := readTOML(path, &acct)

	glog.V(1).Infof("Finished reading the preferences file, showing results: %v6", &acct)

	if err != nil {
		glog.Info("While reading the preferences file, an error was encountered:%s", err)
	}

	return acct
}

//read from a toml file
//check that the file is accessible since the function can be called from a commandline argument
// The content of iface is undefined when the returned error is not nil.
func readTOML(file string, iface interface{}) error {
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

	glog.Infof("Passed log 1 point, '%s'", path)
	glog.Infof("Data at log 1 point, '%s'", string(dat))

	glog.V(1).Infof("Parsing preferences TOML: '%s'", path)
	if err := toml.Unmarshal(dat, iface); err != nil {
		return fmt.Errorf("While reading in the TOML file '%s': %v", path, err)
	}
	glog.Infof("Passed readTOML log 2 point, '%s'", path)

	glog.V(1).Infof("Parsed '%s' to: %+v", path, iface)

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

type programParameters struct {
	addLocation     string
	command         string
	datPath         string
	dryRun          bool
	preferencesFile string
	cf.Account
}

// Returns an account information given all of the options given on the command
// line.
func (p *programParameters) makeAccount() *cf.Account {
	var acct cf.Account
	glog.Infof("Passed makeAccount log 0 point part 1 of 2")
	glog.Infof("Passed makeAccount log 0 point: preferencesFile:%s", p.preferencesFile)

	// Command line options override settings in preferences file.
	// Read the preferences file first, then set any command line options.
	if p.preferencesFile != "" {
		// read the preferences file and populate fields.
		// Ignore any error.
		glog.Infof("Passed makeAccount log 1 point")
		acct = readPreferencesFile(p.preferencesFile)
	}

	// Override fields with any flags that were present.
	if p.Email != "" {
		acct.Email = p.Email
	}
	if p.AccountID != "" {
		acct.AccountID = p.AccountID
	}
	if p.Namespace != "" {
		acct.Namespace = p.Namespace
	}
	if p.Key != "" {
		acct.Key = p.Key
	}
	if p.Token != "" {
		acct.Token = p.Token
	}
	if p.Location != "" {
		acct.Location = p.Location
	}
	if p.addLocation != "" {
		acct.Location = acct.Location + "," + p.addLocation
	}
	if p.Zip != "" {
		acct.Zip = p.Zip
	}
	if p.HomeDirectory != "" {
		acct.HomeDirectory = p.HomeDirectory
	}
	glog.Infof("Passed makeAccount log 2 point")

	acct.LocalOnly = p.dryRun

	if err := acct.Validate(); err != nil {
		glog.Errorf("Could not make an account from arguments: %v", err)
		return nil
	}
	glog.Infof("Passed makeAccount log 3 point")

	return &acct
}

//process the command line commands
//yes, email, Account, Data, Email, Namespace, Key, Token, Location string
//backup strategy, zip, encrypt, verbose, sync, list data, alt pref, no pref
func extractCommandLine() programParameters {
	flag.Usage = func() {
		var args []string
		var desc string
		for k, v := range commands {
			args = append(args, k)
			desc += fmt.Sprintf("  %s\n\t%s\n", k, v.help)
		}
		fmt.Fprintf(flag.CommandLine.Output(), "USAGE: goLocBackup [options] <%s>\n\n", strings.Join(args, "|"))
		fmt.Fprintf(flag.CommandLine.Output(), "commands:\n%s\noptions:\n", desc)
		flag.PrintDefaults()
	}

	var p programParameters

	flag.StringVar(&p.preferencesFile, "pref", "", "use an alternate preference file")
	flag.BoolVar(&p.dryRun, "dryrun", false, "Dry run. Goes through all the steps, but it makes no changes on disk or network")

	// Account Overrides
	flag.StringVar(&p.Email, "email", "", "Set the User email instead of using any preferences file")
	flag.StringVar(&p.AccountID, "account", "", "Set the User Account instead of using any preferences file")
	flag.StringVar(&p.Namespace, "namespace", "", "Set the User's Namespace instead of using any preferences file")
	flag.StringVar(&p.Key, "key", "", "Set the Account Global Key instead of using any preferences file")
	flag.StringVar(&p.Token, "token", "", "Set the Configured KV Workers key instead of using any preferences file")
	flag.StringVar(&p.addLocation, "addLocation", "", "Add these locations/files to backup in addition to those set in the preferences file")
	flag.StringVar(&p.Location, "location", "", "Use only these locations to backup")
	// FIXME Duplicate? flag.String(&p.Location, "download", "", "Download files. By default use the preferences location. Use -location and -addLocation to modify the files downloaded.")
	flag.StringVar(&p.Zip, "zip", "", "Set the zip compression to 'none', 'zstandard', or 'zip'")
	flag.StringVar(&p.HomeDirectory, "home", "", "Change your home directory. All relative paths based on home directory")
	flag.StringVar(&p.datPath, "save", "", "save the current data queue to the save file")

	// Force logs to stderr, as this is a command line program.
	flag.Set("logtostderr", "true")
	flag.Parse()
	glog.V(1).Infoln("Verbose information is true, opening the flood gates!")
	if p.dryRun {
		glog.Infoln("Dry Run is active! No changes will be made!")
	}

	if l := len(flag.Args()); l != 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "ERROR: Expecting exactly one command, but %d arguments were given\n\n", l)
		flag.Usage()
		os.Exit(1) // print usage and exit
	}

	if _, ok := commands[flag.Arg(0)]; !ok {
		fmt.Fprintf(flag.CommandLine.Output(), "Unknown command '%s'\n\n", flag.Arg(0))
		flag.Usage()
		os.Exit(1) // print usage and exit
	}

	p.command = flag.Arg(0)

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

func doSearch(p *programParameters) error {
	d := gobackup.Data{ReadOnly: p.dryRun}

	d.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

	glog.Info("Displaying sorted dat")
	sort.Sort(gobackup.ByFilepath(dat.TheMetadata))

	glog.Infof("%+v", dat.TheMetadata)

	//save the data container to the file indicated by *saveFlag
	writeTOML(p.datPath, &dat)

	return nil
}
func doListAll(p *programParameters) error {
	d := gobackup.Data{ReadOnly: p.dryRun}
	d.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

	glog.Info("Displaying sorted dat")
	sort.Sort(gobackup.ByFilepath(dat.TheMetadata))

	glog.Info("Listing Files by key, filename, and last modified time")
	for i := 0; i < len(dat.TheMetadata); i++ {
		glog.Info(dat.TheMetadata[i].Hash + " " + dat.TheMetadata[i].FileName + " " + dat.TheMetadata[i].Filepath + " " + dat.TheMetadata[i].Mtime.String())
	}

	//if the saveFlag was set, save, otherwise only list
	if p.datPath != "" {
		//save the data container to the file indicated by *saveFlag
		writeTOML(p.datPath, &dat)
	}

	return nil
}
func doListRecent(p *programParameters) error {
	d := gobackup.Data{ReadOnly: p.dryRun}
	d.SearchLocalDatabase(&dat, "data.dat", "method", "key", "asc", "result")

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
	writeTOML(p.datPath, &dat)

	return nil
}

func doGetKeys(p *programParameters) error {
	acct := p.makeAccount()
	if acct == nil {
		return fmt.Errorf("Could not get keys from invalid account")
	}
	//get keys
	glog.Infoln("Getting the keys and metadata!")
	glog.Infoln(string(acct.GetKVkeys()))
	return nil
}

func doSync(p *programParameters) error {
	acct := p.makeAccount()
	if acct == nil {
		return fmt.Errorf("Could not get keys from invalid account")
	}
	//get keys
	glog.Infoln("Getting the keys and metadata!")
	jsonKeys := acct.GetKVkeys()
	glog.Infof("jsonKeys:%s", jsonKeys)

	var extractedData cf.CloudflareResponse

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

		d := gobackup.Data{ReadOnly: p.dryRun}
		d.DataFile2("data.dat", &dat) // FIXME respect the -save flag?
	} else {
		glog.Infoln("Empty Result")
	}
	return nil
}
func doUpload(p *programParameters) error {
	//the filelist for backup
	var fileList []string

	glog.V(1).Infof("CF LOCATION:%v", p.Location)

	backupLocations := strings.Split(p.Location, ",")
	// TODO canonicalize each and check they are under home directory.

	for _, l := range backupLocations {
		if err := getFiles(strings.TrimSpace(l), &fileList); err != nil {
			glog.Warningf("While getting files to back up: %v", err)
			continue
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
		return nil
	}

	//information for user
	glog.Infof("Backing up %v Files, Data Size: %v", dat.Count, dat.DataSize)

	//split the work and backup
	backup(*p)

	d := gobackup.Data{ReadOnly: p.dryRun}
	//update the local data file
	d.DataFile2("data.dat", &dat)

	return nil
}

func (p *programParameters) doCommand() error {
	if p == nil {
		return fmt.Errorf("BUG: Empty program parameters")
	}
	gobackup.AddLock()
	defer gobackup.DeleteLock()

	e, ok := commands[p.command]
	if !ok {
		return fmt.Errorf("BUG: unknown command '%s'", p.command)
	}
	return e.fn(p)
}

func main() {
	//command line can overwrite the data from the preferences file
	p := extractCommandLine()
	cf := p.makeAccount()
	if cf == nil {
		glog.Fatalf("Could not determine account information")
	}

	if err := os.Chdir(cf.HomeDirectory); err != nil {
		glog.Fatalf("Could not change working directory to home directory '%s': %v", cf.HomeDirectory, err)
	}

	if err := p.doCommand(); err != nil {
		os.Exit(1)
	}

} //main
