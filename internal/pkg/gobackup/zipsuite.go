package gobackup

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
)

func ZipFile(filename string, zipname string) {
	fmt.Println("zipfile")

	//open the file to be zipped
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
	}

	defer file.Close()

	//get the fileInfo => will be transferred to zip
	fileInfo, _ := file.Stat()

	//create the zip file
	zipFile, err := os.Create(zipname)
	if err != nil {
		log.Println(err)
	}
	defer zipFile.Close()

	//create a zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	//create a zip file header
	fh, error := zip.FileInfoHeader(fileInfo)
	if error != nil {
		fmt.Println("File Info error")
	}

	//alter the name of the file, can use full path
	workingDir, _ := os.Getwd()
	fullPath := workingDir + filename
	fh.Name = fullPath

	//specify the method of zipping
	fh.Method = zip.Deflate

	//create the new zip header
	writer, _ := zipWriter.CreateHeader(fh)

	//copy from file to the writer
	_, err = io.Copy(writer, file)
	if err != nil {
		log.Println(err)
	}

}

func ZipPipe(pr *io.PipeReader, pw *io.PipeWriter, filename string) {

	//open the file to be zipped
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	//get the fileInfo => will be transferred to zip
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println(err)
	}

	//create a zip writer
	zipWriter := zip.NewWriter(pw)
	defer zipWriter.Close()

	//create a zip file header
	fh, error := zip.FileInfoHeader(fileInfo)
	if error != nil {
		fmt.Println("File Info error")
	}

	//alter the name of the file, can use full path
	workingDir, _ := os.Getwd()
	workingDir = workingDir[3:]

	fullPath := workingDir + string(os.PathSeparator) + filename
	fmt.Println("THE FULL PATH:" + fullPath)
	fh.Name = fullPath

	//specify the method of zipping
	fh.Method = zip.Deflate

	//create the new zip header
	writer, _ := zipWriter.CreateHeader(fh)

	//move data into the pipe
	_, _ = io.Copy(writer, file)

	//flush and close the writer
	zipWriter.Close()

	//close data pipe, signalling the end
	pw.Close()
	//	fmt.Printf("Wrote %v\n", n)
}

/*
func main() {

	fmt.Println("Start zip!")

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	go zipPipe(pr, pw, "README.txt")

	myzip, err := os.Create("readme.zip")
	if err != nil {
		log.Println(err)
	}

	io.Copy(myzip, pr)

	fmt.Println("next zip")
	//	piper, pipew := io.Pipe()

	zipFile("test.txt", "test.zip")
	fmt.Println("done zip")

}
*/
