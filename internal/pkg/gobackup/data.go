package gobackup

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/glog"
)

//convert hex bytes into a string
func hashToString(in []byte) string {
	return hex.EncodeToString(in)
}

//run md5 hash on a file
func Md5file(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		DeleteLock()
		glog.Fatalf("md5 failed")
	}
	data := md5.Sum(dat)
	return hashToString(data[:])
}

//run md5 hash on a file contents and metadata
func Md5fileAndMeta(in string) string {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		DeleteLock()
		glog.Fatalf("while generating hash for file and metadata: %v", err)
	}

	result := fmt.Sprintf("%v%v", string(dat), CreateMeta(in))
	glog.V(2).Infof("CONTENT AND META******%v******\n\n", result)
	data := md5.Sum([]byte(result))
	return hashToString(data[:])
}

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

//create a data file from data struct
func DataFile2(file string, dat *DataContainer) {
	var theFile io.ReadWriter
	var err error

	if DryRun { //dryRun
		glog.Infoln("Dry run, setting output to standard out!")
		theFile = os.Stdout
	} else {
		glog.Infoln("Opening the data.dat file!")
		theFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			DeleteLock()
			glog.Fatalf("problem opening file '%s': %v", file, err)
		}
	}

	datawriter := bufio.NewWriter(theFile)
	for i, data := range dat.TheMetadata {
		//added a spacer between the hash and filename
		_, err := datawriter.WriteString(dat.TheMetadata[i].Hash + ":" + GetMetadata(data) + "\n")
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
	temp.File = "f1o1"
	temp.Mtime = fi.ModTime()
	temp.Permissions = fi.Mode().Perm().String()
	temp.Size = fi.Size()
	return temp
}

func GetMetadata(d Metadata) string {
	return fmt.Sprintf("%s:%d:%s:%d", d.FileName, d.FileNum, d.Notes, d.Mtime.Unix())
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
	Filepath    string    `json:"filepath"` //relative path to pref
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
