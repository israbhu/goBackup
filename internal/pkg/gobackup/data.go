package gobackup

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

type Data struct {
	ReadOnly bool
}

type MyData struct {
	Name        string   `json:"name"`
	TheMetadata Metadata `json:"metadata"`
}

type CanonicalPathString string

//convert hex bytes into a string
func hashToString(in []byte) string {
	return hex.EncodeToString(in)
}

//run md5 hash on a file
func Md5file(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		// FIXME Do not fatal
		glog.Fatalf("md5 failed")
	}
	data := md5.Sum(dat)
	return hashToString(data[:])
}

//run md5 hash on a file contents and metadata
func Md5fileAndMeta(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		// FIXME Do not fatal
		glog.Fatalf("while generating hash for file and metadata: %v", err)
	}

	result := fmt.Sprintf("%v%v", string(dat), CreateMeta(in))
	glog.V(2).Infof("CONTENT AND META******%v******\n\n", result)
	data := md5.Sum([]byte(result))
	return hashToString(data[:])
}

/*
//write data in a stream-like fashion
func BuildData2(a *DataContainer) ([]byte, error) {
	var result bytes.Buffer

	for i := 0; i < len(a.TheMetadata); i++ {
		aMetadata := a.TheMetadata[i]
		part, err := json.Marshal(aMetadata)
		if err != nil {
			glog.Warningf("While marshaling metadata %+v: %v", aMetadata, err)
			continue
		}

		result.WriteString(string(part))
	}
	return result.Bytes(), nil
}
*/
//create a data file from data struct
func (d *Data) DataFile2(file string, dat *DataContainer) {
	var theFile io.ReadWriter
	var err error

	if d.ReadOnly { //dryRun
		glog.Infoln("Dry run, setting output to standard out!")
		theFile = os.Stdout
	} else {
		glog.Infoln("Opening the data.dat file!")
		theFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			// FIXME Do not fatal
			glog.Fatalf("problem opening file '%s': %v", file, err)
		}
	}

	datawriter := bufio.NewWriter(theFile)
	for i, data := range dat.TheMetadata {
		//added a spacer between the hash and filename
		_, err := datawriter.WriteString(MetadataToString(data) + "\n")
		NoErrorFound(err, "Error in Datafile2!")

		glog.V(1).Infof("Adding item:%v and data:%v", i, data)
	}
	datawriter.Flush()

}

//extracts the Metadata from a file
func CreateMeta(file string) Metadata {
	fi, err := os.Lstat(file)
	if err != nil {
		glog.Fatalln(err)
	}

	var temp Metadata

	glog.Infof("permissions: %#o\n", fi.Mode().Perm()) // 0400, 0777, etc.

	temp.FileName = file
	temp.Hash = Md5file(file)
	temp.ForeignKey = ""
	temp.FileNum = 0
	temp.FileInfo = "f1o1"
	temp.Mtime = fi.ModTime()
	temp.Permissions = fi.Mode().Perm().String()
	temp.Size = fi.Size()
	return temp
}

//convert from Metadata to a string
/*
	FileInfo    string    `json:"fileInfo"`
	Filepath    string    `json:"filepath"` //relative path to pref
	Notes       string    `json:"notes"`
	Permissions string    `json:"permissions"`
	Hash        string    `json:"hash"`
	ForeignKey  string    `json:"foreignkey"`
	FileNum     int       `json:"file_num"`
	FileName    string    `json:"filename"`
	Mtime       time.Time `json:"mtime"`
	Size        int64
	pr          *io.PipeReader
*/
func MetadataToString(d Metadata) string {
	return fmt.Sprintf("%v:%d:%s:%v:%v:%d:%v:%s:%d:%v", d.Hash, d.Size, d.FileName, d.Filepath, d.FileInfo, d.FileNum, d.ForeignKey, d.Notes, d.Mtime.Unix(), d.Permissions)
}

//reverse the MetadataToString function
func StringToMetadata(stringMeta string) Metadata {
	var containsc bool

	//remove colon
	if strings.Contains(stringMeta, "C:\\") {
		stringMeta = strings.Replace(stringMeta, "C:\\", "C;\\", 1)
		containsc = true
	} else {
		containsc = false
	}

	items := strings.Split(stringMeta, ":")

	//add colon back in
	if containsc {
		items[2] = strings.Replace(items[2], "C;\\", "C:\\", 1)
	}

	//trim out whitespace
	items[8] = strings.TrimSpace(items[8])

	//list the items
	//	glog.Infof("Item0:%v Item1:%v Item2:%v Item3:%v Item4:%v Item5:%v Item6:%v Item7:%v Item8:%v ", items[0], items[1], items[2], items[3], items[4], items[5], items[6], items[7], items[8])

	//convert from string to base 10, 64bit int
	i, err := strconv.ParseInt(items[8], 10, 64)
	if err != nil {
		glog.Error("Error converting string mtime to int64! %v", err)
	}
	size, err := strconv.ParseInt(items[1], 10, 64)
	if err != nil {
		glog.Error("Error converting string size to int64! %v", err)
	}
	num, err := strconv.ParseInt(items[5], 10, 32)
	if err != nil {
		glog.Error("Error converting string filenum to int64! %v", err)
	}

	meta := Metadata{
		Permissions: items[9],
		Filepath:    items[3],
		FileInfo:    items[4],
		FileNum:     int(num),
		FileName:    items[2],
		Hash:        items[0],
		ForeignKey:  items[6],
		Notes:       items[7],
		Mtime:       time.Unix(i, 0),
		Size:        size,
	}

	return meta
}

//searches the database and returns the results as a string
//method describes the search method, by name, by filepath, etc
//key describes what the search looks for
//order will reorder the results asc (ascending), desc (descending), none (as entered)
//result allows you to add formatting to the result string
//the result string should be able to be outputted to standard out or a file to be used in combination with the download option
func (d *Data) SearchLocalDatabase(dat *DataContainer, file string, method string, key string, order string, result string) string {
	d.openLocalDatabase(file, "byHash", dat)
	//	fmt.Println(dat)
	return ""
}

//open the local database and sort it
//file should be a string pointing to the file
//sort is the preferred sort method run on the data
func (d *Data) openLocalDatabase(file string, sort string, dat *DataContainer) {

	var theFile io.ReadWriter
	var err error

	if d.ReadOnly { //dryRun
		glog.Infoln("Dry run, setting output to standard out!")
		theFile = os.Stdout
	} else {
		glog.Infoln("Opening the " + file + " file!")
		theFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			// FIXME Do not fatal
			glog.Fatalf("problem opening file '%s': %v", file, err)
		}
	}

	datareader := bufio.NewReader(theFile)

	//preload for loop
	line, err := datareader.ReadString('\n')

	//	fmt.Println("First Loop " + line)
	//loop over the dat file until io.EOF
	for err != io.EOF {

		//convert from string to Metadata
		meta := StringToMetadata(line)

		glog.Infof("Appending data to dat: %v", meta)

		//append it to the data container
		dat.TheMetadata = append(dat.TheMetadata, meta)

		//preload for next loop
		line, err = datareader.ReadString('\n')
	}
	glog.Infof("%v meta was loaded into the container\n", len(dat.TheMetadata))
}

type Stream string

//this struct stores the Metadata that will be uploaded with each file
type Metadata struct {
	//f1o1 = file 1 of 1
	//note: = notes
	//modified timestamp
	//permissions
	//hash is the hash of the contents
	//foreign key is the hash of the contents and metadata (blank if source data)
	//folder structure
	//Metadata filename:
	//FileNum is the current file number (starting from 0) in a file that has been split
	//Metadata example test.txt:f2o4:ph#:fh#:
	FileInfo    string    `json:"fileInfo"`
	Filepath    string    `json:"filepath"` //relative path to pref
	Notes       string    `json:"notes"`
	Permissions string    `json:"permissions"`
	Hash        string    `json:"hash"`
	ForeignKey  string    `json:"foreignkey"`
	FileNum     int       `json:"file_num"`
	FileName    string    `json:"filename"`
	Mtime       time.Time `json:"mtime"`
	Size        int64
	PR          *io.PipeReader
}

// ByHash Implements sort.Interface for []Metadata based on the Hash field.
type ByHash []Metadata

func (h ByHash) Len() int {
	return len(h)
}

func (h ByHash) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h ByHash) Less(i, j int) bool {
	return h[i].Hash < h[j].Hash
}

// ByFilepath Implements sort.Interface for []Metadata based on the Hash field.
type ByFilepath []Metadata

func (h ByFilepath) Len() int {
	return len(h)
}

func (h ByFilepath) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h ByFilepath) Less(i, j int) bool {
	//tie breaker by time
	if h[i].Filepath == h[j].Filepath {
		return h[i].Mtime.Unix() < h[j].Mtime.Unix()
	} else {
		return h[i].Filepath < h[j].Filepath
	}
}

//******* This struct contains the data tracking uploads*****
//Redo: Datasize will be size of files
//      remove hash, use only Metadata => add hash to Metadata
//
type DataContainer struct {
	DataSize           int64 //keeps track of the byte size of the uploads
	Count              int   //keeps track of the number of files
	CF_MAX_UPLOAD      int   //max number of files for upload
	CF_MAX_DATA_UPLOAD int64 //max data uploaded at a time
	CF_MAX_DATA_FILE   int64 //max data per file

	TheMetadata []Metadata
}

//sync to online => pull Metadata from cloud and rebuild
//sync to drive  => pull Metadata from cloud and check against the drive database, reupload anything missing

// MakeCanonicalPath returns an absolute path for the given path, where
// symlinks are evaluated and the path is cleaned according to filepath.Clean.
// The returned path is empty when error is not nil.
func MakeCanonicalPath(path string) (CanonicalPathString, error) {
	// abs
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("While getting absolute path for '%s': %v", path, err)
	}

	// eval symlink
	ret, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("While evaluating symlinks for '%s': %v", abs, err)
	}

	return CanonicalPathString(ret), nil
}

//check that the path is not above the home directory
func CheckPath(path, homePath CanonicalPathString) bool {
	glog.V(1).Infof("checking path for: %s", path)
	return strings.HasPrefix(string(path), string(homePath))
}
