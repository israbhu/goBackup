package gobackup

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//determine if a path is relative or not
func isRelative(file string) bool {
	workingDirectory, _ := os.Getwd()

	volumeName := filepath.VolumeName(workingDirectory)

	//if there's a volumename, it should not be relative
	if strings.Contains(file, volumeName) {
		return false
	} else {
		return true
	}
}

//convert hex bytes into a string
func hashToString(in []byte) string {
	return hex.EncodeToString(in)
}

/*
//run md5 hash on a string
func md5string(a string) string {
	data := md5.Sum([]byte(a))
	return hashToString(data[:])
}
*/

//run md5 hash on a file
func Md5file(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		Logger.Fatalf("md5 failed")
	}
	data := md5.Sum(dat)
	return hashToString(data[:])
}

//run md5 hash on a file contents and metadata
func Md5fileAndMeta(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		Logger.Fatalf("while generating hash for file and metadata: %v", err)
	}
	result := fmt.Sprintf("%v%v", string(dat), CreateMeta(in))
	Logger.Printf("CONTENT AND META******%v******\n\n", result)
	data := md5.Sum([]byte(result))
	return hashToString(data[:])
}

//write data in a stream-like fashion
func BuildData2(a *Data1) ([]byte, error) {
	var result bytes.Buffer

	for i := 0; i < len(a.TheMetadata); i++ {
		aMetadata := a.TheMetadata[i]
		part, err := json.Marshal(aMetadata)
		if err != nil {
			Logger.Println(err)
		}

		result.WriteString(string(part))
	}
	return result.Bytes(), nil
}

//write data to disk for a maximum of 100 MB
func BuildData(a *Data1) string {
	var sb strings.Builder

	//start the array
	sb.WriteString("[")

	//if files are less than 100MB or less than 10000
	buf := bytes.Buffer{}

	//for loop
	for i := 0; i < (len(a.TheMetadata) - 1); i++ {
		d := a.TheMetadata[i]

		file, err := os.Open(string(d.FileName))
		if err != nil {
			Logger.Fatalf("searchData failed opening file:" + string(d.FileName))
		}
		defer file.Close()

		body, err := ioutil.ReadAll(file)
		if err != nil {
			Logger.Fatalln(err)
		}

		sb.WriteString("{\"key\":\"")
		sb.WriteString(d.Hash)
		sb.WriteString("\",\"value\":\"")
		json.HTMLEscape(&buf, body)
		sb.Write(buf.Bytes()) //compress and encrypt?
		sb.WriteString("\",\"expiration_ttl\":")
		sb.WriteString("60000")
		sb.WriteString(",\"Metadata\":{\"")
		sb.WriteString("The Metadata Key")
		sb.WriteString("\":\"")
		sb.WriteString(GetMetadata(d))
		sb.WriteString("\"},\"base64\":false},")
	}

	d := a.TheMetadata[len(a.TheMetadata)-1]

	file, err := os.Open(string(d.FileName))
	if err != nil {
		Logger.Fatalf("searchData failed opening file:" + string(d.FileName))
	}
	defer file.Close()

	body, err := ioutil.ReadAll(file)
	if err != nil {
		Logger.Fatalln(err)
	}

	sb.WriteString("{\"key\":\"")
	sb.WriteString(d.Hash)
	sb.WriteString("\",\"value\":\"")
	escaped := strings.ReplaceAll(string(body), `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	sb.WriteString(escaped)
	sb.WriteString("\",\"expiration_ttl\":")
	sb.WriteString("6000")
	sb.WriteString(",\"Metadata\":{\"")
	sb.WriteString("The Metadata Key")
	sb.WriteString("\":\"")
	sb.WriteString(GetMetadata(d))
	sb.WriteString("\"},\"base64\":false}")

	//end the array
	sb.WriteString("]")

	Logger.Println("SB =" + sb.String())
	return sb.String()
}

//create a data file from data struct
func DataFile2(file string, dat *Data1) {
	var theFile io.ReadWriter
	var err error

	if DryRun { //dryRun
		Logger.Println("Dry run, setting output to standard out!")
		theFile = os.Stdout
	} else {
		Logger.Println("Opening the data.dat file!")
		theFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			Logger.Fatalf("problem opening file '%s': %v", file, err)
		}
	}

	datawriter := bufio.NewWriter(theFile)
	for i, data := range dat.TheMetadata {
		//added a spacer between the hash and filename
		_, err := datawriter.WriteString(dat.TheMetadata[i].Hash + ":" + GetMetadata(data) + "\n")
		CheckError(err, "Error in Datafile2!")

		if Verbose {
			Logger.Printf("Adding item:%v and data:%v\n", i, data)
		}
	}
	datawriter.Flush()

}

//extracts the Metadata from a file
func CreateMeta(file string) Metadata {
	fi, err := os.Lstat(file)
	if err != nil {
		Logger.Fatalln(err)
	}

	var temp Metadata

	Logger.Printf("permissions: %#o\n", fi.Mode().Perm()) // 0400, 0777, etc.

	temp.FileName = file
	temp.Hash = Md5file(file)
	/*
	   	//check if the path is relative
	   	if isRelative(file) {
	   		workingDirectory, _ := os.Getwd()
	   //		volumeName := filepath.VolumeName(workingDirectory)

	   		//if the path has a ., then need to modify the path
	   		// you can do a .\ or .\dir or a ..\ or a ..\dir1\dir2
	   		if strings.Contains(file, ".") {

	   		} else {
	   			temp.Filepath = workingDirectory+string(os.PathSeparator)+file
	   		}
	   	} else {
	   		temp.Filepath = file
	   	}
	*/
	temp.ForeignKey = ""
	temp.FileNum = 0
	temp.File = "f1o1"
	temp.Mtime = fi.ModTime()
	temp.Permissions = fi.Mode().Perm().String()
	temp.Size = fi.Size()
	return temp
}

func GetMetadata(d Metadata) string {
	return fmt.Sprintf("%v:%v:%v:%v", d.FileName, d.FileNum, d.Notes, d.Mtime)
	//	return d.FileName + ":" + d.FileNum + ":" + d.Notes + ":" + d.Atime.String()
}

func (a Stream) MarshalJSON() ([]byte, error) {
	//start of marshal
	Logger.Println("Marshal is working!")
	//convert from Stream type to string
	filename := string(a)

	file, err := os.Open(filename)
	if err != nil {
		Logger.Println(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		Logger.Println(err)
	}

	//escape html
	base64 := base64.StdEncoding.EncodeToString(fileContents)
	base64 = "\"" + base64 + "\""

	//dest, source
	Logger.Printf("Length:%v", len(base64))
	return []byte(base64), nil
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
	File        string    `json:"file"`
	Notes       string    `json:"notes"`
	Permissions string    `json:"permissions"`
	Filepath    string    `json:"filepath"`
	Hash        string    `json:"hash"`
	ForeignKey  string    `json:"foreignkey"`
	FileNum     int       `json:"file_num"`
	FileName    string    `json:"filename"`
	Mtime       time.Time `json:"mtime"`
	Size        int64
	pr          *io.PipeReader
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

//******* This struct contains the data tracking uploads*****
//Redo: Datasize will be size of files
//      remove hash, use only Metadata => add hash to Metadata
//
type Data1 struct {
	DataSize           int64 //keeps track of the byte size of the uploads
	Count              int   //keeps track of the number of files
	CF_MAX_UPLOAD      int   //max number of files for upload
	CF_MAX_DATA_UPLOAD int64 //max data uploaded at a time
	CF_MAX_DATA_FILE   int64 //max data per file

	TheMetadata []Metadata
}

//sync to online => pull Metadata from cloud and rebuild
//sync to drive  => pull Metadata from cloud and check against the drive database, reupload anything missing
