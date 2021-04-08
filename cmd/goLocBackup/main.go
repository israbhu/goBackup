package main

import (
	"goLocBackup/internal/pkg/gobackup"
	"sort"

	//test
	"bufio"

	//flags
	"flag"
	"fmt"
	"strings"

	//read and write files
	"io/ioutil"
	"os"
	"path/filepath"

	//http
	"log"
	//toml
	//	"github.com/komkom/toml"
	"github.com/pelletier/go-toml"
)

//***************global variables*************
var cf gobackup.Account //account credentials and preferences
var dat gobackup.Data1  //local datastore tracking uploads and Metadata
var verbose bool        //flag for extra info output to console

/*what do do when uploading data
done create md5 sum

done compare to the list of hashes
sort

compress
encrypt
done Metadata
done move it to cloudflare
*/

//backs up the list of files
//uploading the data should be the most time consuming portion of the program, so it will pushed into a go routine
func backup(list []string) {
	gobackup.UploadKV(&cf, &dat)
}

//read from a toml file
//check that the file exists since the function can be called from a commandline argument
func readTOML(file string) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}
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
	//verbose
	//	var yFlag = flag.Bool("y", false, "Always accept all prompts")
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
	/*	var encryptFlag = flag.String("crypt", "", "encrypt")
		var syncFlag = flag.Bool("s", false, "Synchronize local data to the cloud")
		var listdataFlag = flag.Bool("list", false, "List the data from the local data")
	*/var altPrefFlag = flag.String("pref", "", "use an alternate preference file")

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
		//		locations := *zipFlag
		fmt.Println("Extracting downloads:" + cf.Location)
		//		getFiles(cf.Locations)

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

		//		fmt.Println(a)
		if b[0] == hash && fileName == b[1] {
			return true
		}
	}
	return false
}

//create a toml file from a struct
func dataFile() {

	//	var doc []byte
	doc, _ := toml.Marshal(&dat)

	fmt.Println(string(doc))
	fmt.Println(dat)
	err := ioutil.WriteFile("data.dat", doc, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	fi, err := os.Lstat("data.dat")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("permissions: %#o\n", fi.Mode().Perm()) // 0400, 0777, etc.

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

/*
extract toml data for account and behaviour information
extract out the files for backup
backup the data
*/
func main() {

	fmt.Println(gobackup.GetMetadata(gobackup.Metadata{}))
	readTOML("preferences.toml")
	fmt.Println(cf)
	fmt.Println("****************************\n")

	fmt.Println("zipping a file")
	gobackup.ZipFile("zipsuite.txt", "zipsuite.zip")
	//get the command arguments
	//command line can overwrite the data from the preferences file
	extractCommandLine()

	//get the filelist for backup
	var fileList = make([]string, 0)

	backupLocations := strings.Split(cf.Location, ",")

	for i := 0; i < len(backupLocations); i++ {
		fileList = getFiles(strings.TrimSpace(backupLocations[i]), fileList)
		//		fileList = getFiles("c:/testdir", fileList)
	}

	//	fileList = append(fileList, "am.txt")
	//fmt.Println("UNSORTED")
	//fmt.Println(fileList)
	sort.Strings(fileList)
	//fmt.Println("SORTED")
	//fmt.Println(fileList)

	var a []string

	//fill in the Metadata
	for i := 0; i < len(fileList); i++ {

		hash := gobackup.Md5file(fileList[i])
		a = append(a, hash)
	}
	//	fmt.Println("UNSORTED")
	//	fmt.Println(a)
	sort.Strings(a)
	//	fmt.Println("SORTED")
	//	fmt.Println(a)

	//fill in the Metadata
	for i := 0; i < len(fileList); i++ {

		hash := gobackup.Md5file(fileList[i])

		//if not found
		if !searchData(hash, fileList[i]) {
			meta := gobackup.CreateMeta(fileList[i])
			fmt.Println("NOT FOUND AND INCLUDING! " + hash + "-" + gobackup.GetMetadata(meta))

			//update the data struct
			//			dat.Hash = append(dat.Hash, hash)
			dat.TheMetadata = append(dat.TheMetadata, meta)

			//TODO: sort the data struct

			dat.DataSize += meta.Size
			dat.Count += 1
		} else {
			fmt.Println("FOUND AND EXCLUDING! " + hash)
		}
	} //for

	if len(dat.TheMetadata) == 0 {
		fmt.Println("All files are up to date! Exiting!")
		os.Exit(0)
	}

	fmt.Printf("Data Size: %v, Data Count: %v", dat.DataSize, dat.Count)
	//split the work and backup
	backup(fileList)
	fmt.Println(gobackup.DownloadKV(&cf, dat.TheMetadata[0].Hash, "test.txt"))

	//update the local data file
	//TODO in the future, I will sort the data file, at the moment, it appends the new data to the end of the file
	gobackup.DataFile2("data.dat", &dat)

	//	getKVkeys()

} //main

/*

Bulk upload example

curl -X PUT "https://api.cloudflare.com/client/v4/accounts/01a7362d577a6c3019a474fd6f485823/storage/kv/namespaces/0f2ac74b498b48028cb68387c421e279/bulk" \
     -H "X-Auth-Email: user@example.com" \
     -H "X-Auth-Key: c2547eb745079dac9320b638f5e225cf483cc5cfdda41" \
     -H "Content-Type: application/json" \
     --data '[{"key":"My-Key","value":"Some string","expiration":1578435000,"expiration_ttl":300,"Metadata":{"someMetadataKey":"someMetadataValue"},"base64":false}]'

Writing data in bulk
You can write more than one key-value pair at a time with wrangler or via the API, up to 10,000 KV pairs. A key and value are required for each KV pair. The entire request size must be less than 100 megabytes. We do not support this from within a Worker script at this time.

You can choose one of two ways to specify when a key should expire:

Set its "expiration", using an absolute time specified in a number of seconds since the UNIX epoch. For example, if you wanted a key to expire at 12:00AM UTC on April 1, 2019, you would set the key’s expiration to 1554076800.

Set its "expiration TTL" (time to live), using a relative number of seconds from the current time. For example, if you wanted a key to expire 10 minutes after creating it, you would set its expiration TTL to 600.

Creating expiring keys
We talked about the basic form of the put method above, but this call also has an optional third parameter. It accepts an object with optional fields that allow you to customize the behavior of the put. In particular, you can set either expiration or expirationTtl, depending on how you would like to specify the key’s expiration time. In other words, you’d run one of the two commands below to set an expiration when writing a key from within a Worker:

NAMESPACE.put(key, value, {expiration: secondsSinceEpoch})

NAMESPACE.put(key, value, {expirationTtl: secondsFromNow})

These assume that secondsSinceEpoch and secondsFromNow are variables defined elsewhere in your Worker code.

You can also write with an expiration on the command line via Wrangler or via the API

Additionally, if list_complete is false, there are more keys to fetch. You’ll use the cursor property to get more keys. See the Pagination section below for more details.



my datafile should be:
Hash:Metadata \n


Then we can open the file and insert data in the correct spot
If we rebuild the file we can simply download the keys and append key name and Metadata, then sort
*/
/*
   https://stackoverflow.com/questions/45429210/how-do-i-check-a-files-permissions-in-linux-using-go
   //Check 'others' permission
   m := info.Mode()
   if m&(1<<2) != 0 {
       //other users have read permission
   } else {
       //other users don't have read permission
   }
*/
