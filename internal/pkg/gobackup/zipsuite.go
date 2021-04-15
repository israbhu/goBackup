package gobackup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

/* Sample use case

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

//create a zip using pipes
//must use in a go routine
func zipInit(filename string, pr *io.PipeReader, pw *io.PipeWriter, errCh chan error) {
	defer close(errCh)
	defer pw.Close()
	//open the file to be zipped
	file, err := os.Open(filename)
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

/*
func main() {
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
}
*/
