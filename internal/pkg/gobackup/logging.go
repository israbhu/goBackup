package gobackup

import (
	"fmt"
	"io"
	"log"
	"os"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
var Verbose bool = false
var Debug bool = false

//creates a lock file for data.dat
func AddLock() {
	//if exist
	if FileExist("lock.pid") {

		workingDirectory, _ := os.Getwd()
		//add nuances here
		Logger.Fatalf("Data.dat has been locked for access. Please properly close the other program. If you wish to delete the lock manually, delete the %v file in the gobackup directory.", workingDirectory+string(os.PathSeparator)+"lock.pid")
	} else {
		lockfile, err := os.Create("lock.pid")
		CheckError(err, "There was an error creating lock.pid")

		defer lockfile.Close()

		//write the process id into the file
		_, err = io.WriteString(lockfile, fmt.Sprintf("%v", os.Getpid()))
		CheckError(err, "Error writing the processid to the lock file!")
	}
}

//creates a lock file for data.dat
func DeleteLock() {
	Logger.Println("Attempting to delete lock!")
	if FileExist("lock.pid") {
		err := os.Remove("lock.pid")
		CheckError(err, "Error deleting lock.pid")
	}
}

//check if a file called name exists
func FileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		//		Logger.Fatalf("The file at %v does not exist", name)
		return false
	}
	return true
}

//checks the errors and delete lock********
func CheckError(err error, message string) bool {
	if err != nil {
		if message == "" {
			Logger.Printf("Error found! %v", err)
			//			DeleteLock()
			return false
		} else {
			Logger.Printf(message+" %v", err)
			//			DeleteLock()
			return false
		}
	}
	return true
}
