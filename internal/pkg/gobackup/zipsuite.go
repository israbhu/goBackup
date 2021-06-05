package gobackup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/klauspost/compress/zstd"
)

/* Sample use case
0
//open a pipe
pr, pw := io.Pipe()
errCh := make(chan error, 1)
go zipInit(pr, pw, errCh)

//copy from file to the writer
zipFile, _ := os.Create("file.zip")
// Do stuff with pr here.
_, _ = io.Copy(zipFile, pr)
//create the zip file
zipFile.Close()
//file.Close()
var exitCode int

if err, ok := <-errCh; ok {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	exitCode = 1
}

os.Exit(exitCode)



*/

// Change permissions Linux.
func filePermissions(filename string, filemode uint32) {
	//default filemode is all read write and execute
	if filemode <= 0 {
		filemode = 0777
	}

	err := os.Chmod(filename, os.FileMode(filemode))
	if err != nil {
		Logger.Println(err)
	}
}

func fileOwner(filename string) {
	// Change file ownership.
	err := os.Chown(filename, os.Getuid(), os.Getgid())
	if err != nil {
		Logger.Println(err)
	}
}
func fileAccess(filename string, lastAccess time.Time, lastModify time.Time) {
	// Change file timestamps.
	//		addOneDayFromNow := time.Now().Add(24 * time.Hour)
	//		lastAccessTime := addOneDayFromNow
	//		lastModifyTime := addOneDayFromNow
	err := os.Chtimes(filename, lastAccess, lastModify)
	if err != nil {
		Logger.Println(err)
	}
}

//create a zstandard compressed file
func zStandardInit(filename string, pw *io.PipeWriter) {
	defer pw.Close()

	enc, err := zstd.NewWriter(pw)
	if err != nil {
		fmt.Printf("newWriter error:%v", err)
	}
	defer enc.Close()

	//open the file to be zipped
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error in zStandardInit: %v", err)
	}
	defer file.Close()

	written, err := io.Copy(enc, file)
	if err != nil {
		fmt.Printf("io.Copy error:%v", err)
		enc.Close()
	}
	fmt.Printf("Successfully encoded bytes: %d\n", written)
	//	return enc.Close()

}

//no compression
func copyFile(filename string, pr *io.PipeReader, pw *io.PipeWriter) {
	defer pw.Close()
	//open the file to be zipped
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error in copyFile: %v", err)
	}
	defer file.Close()

	written, err := io.Copy(pw, file)
	if err != nil {
		fmt.Printf("io.Copy error:%v", err)
	}
	fmt.Printf("Successfully written:%v", written)

}

//decompress a zstandard compressed file
func zStandardDecompress(filename string, pr *io.PipeReader, pw *io.PipeWriter) {
	defer pw.Close()
	//open the file to be zipped
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error in zStandardDecompress: %v", err)
	}
	defer file.Close()

	dec, err := zstd.NewReader(pr)
	if err != nil {
		fmt.Printf("newReader error:%v", err)
	}
	written, err := io.Copy(pw, dec)
	if err != nil {
		fmt.Printf("io.Copy error:%v", err)
		dec.Close()
	}
	fmt.Printf("Successfully written:%v", written)
}

//create a zip using pipes
//must use in a go routine
func zipInit(filename string, pr *io.PipeReader, pw *io.PipeWriter, errCh chan error) {
	defer close(errCh)
	defer pw.Close()
	//open the file to be zipped
	file, err := os.Open(filename)

	//TODO: change file and time permissions
	if err != nil {
		errCh <- err
		return
	}
	defer file.Close()

	//get the fileInfo => will be transferred to zip
	fileInfo, err := file.Stat()
	if err != nil {
		errCh <- err
		return
	}

	//create a zip writer
	zipWriter := zip.NewWriter(pw)
	defer zipWriter.Close()

	//create a zip file header
	fh, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		errCh <- err
		return
	}

	//alter the name of the file, can use full path
	fh.Name = "theGo.mod"

	//specify the method of zipping
	fh.Method = zip.Deflate

	//create the new zip header
	writer, err := zipWriter.CreateHeader(fh)
	if err != nil {
		errCh <- err
		return
	}

	n, err := io.Copy(writer, file)
	if err != nil {
		errCh <- err
		return
	}
	fmt.Printf("Wrote %d Bytes\n", n)
}
