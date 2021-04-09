package gobackup

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

//convert hex bytes into a string
func hashToString(in []byte) string {
	return hex.EncodeToString(in)
}

//run md5 hash on a string
func md5string(a string) string {
	data := md5.Sum([]byte(a))
	return hashToString(data[:])
}

//run md5 hash on a file
func Md5file(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		log.Fatalf("md5 failed")
	}
	data := md5.Sum(dat)
	return hashToString(data[:])
}

//write data in a stream-like fashion
func BuildData2(a *Data1) ([]byte, error) {
	var result bytes.Buffer

	for i := 0; i < len(a.TheMetadata); i++ {
		aMetadata := a.TheMetadata[i]
		part, err := json.Marshal(aMetadata)
		if err != nil {
			fmt.Println(err)
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
			log.Fatalf("searchData failed opening file:" + string(d.FileName))
		}
		defer file.Close()

		body, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalln(err)
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
		log.Fatalf("searchData failed opening file:" + string(d.FileName))
	}
	defer file.Close()

	body, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalln(err)
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

	fmt.Println("SB =" + sb.String())
	return sb.String()
}

//create a data file from data struct
func DataFile2(file string, dat *Data1) {

	theFile, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("problem opening file '%s': %v", file, err)
	}

	defer theFile.Close()

	datawriter := bufio.NewWriter(theFile)
	for i, data := range dat.TheMetadata {
		//added a spacer between the hash and filename
		_, _ = datawriter.WriteString(dat.TheMetadata[i].Hash + ":" + GetMetadata(data) + "\n")
	}
	datawriter.Flush()

}

//extracts the Metadata from a file
func CreateMeta(file string) Metadata {
	fi, err := os.Lstat(file)
	if err != nil {
		log.Fatalln(err)
	}

	var temp Metadata

	fmt.Printf("permissions: %#o\n", fi.Mode().Perm()) // 0400, 0777, etc.

	temp.FileName = Stream(file)
	temp.Hash = Md5file(file)
	temp.FileNum = "f1o1"
	temp.Atime = fi.ModTime()
	temp.Permissions = fi.Mode().Perm().String()
	temp.Size = fi.Size()
	return temp
}

func GetMetadata(d Metadata) string {
	return string(d.FileName) + ":" + d.FileNum + ":" + d.Notes + ":" + d.Atime.String()
}

func (a Stream) MarshalJSON() ([]byte, error) {
	//start of marshal
	fmt.Println("Marshal is working!")
	//convert from Stream type to string
	filename := string(a)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	//escape html
	base64 := base64.StdEncoding.EncodeToString(fileContents)
	base64 = "\"" + base64 + "\""

	//dest, source
	fmt.Printf("Length:%v", len(base64))
	return []byte(base64), nil
}

type Stream string

//this struct stores the Metadata that will be uploaded with each file
type Metadata struct {
	//f1o1 = file 1 of 1
	//note: = notes
	//modified timestamp
	//permissions
	//folder structure
	//Metadata filename:
	//
	//Metadata example test.txt:f2o4:ph#:fh#:
	FileNum, Notes, Permissions, Filepath, Hash string
	FileName                                    Stream
	Atime                                       time.Time
	Size                                        int64
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
	Count              int   //keeps track of the number of uploads
	CF_MAX_UPLOAD      int   //max number of files for upload
	CF_MAX_DATA_UPLOAD int64 //max data uploaded at a time
	CF_MAX_DATA_FILE   int64 //max data per file

	TheMetadata []Metadata
}

//sync to online => pull Metadata from cloud and rebuild
//sync to drive  => pull Metadata from cloud and check against the drive database, reupload anything missing
