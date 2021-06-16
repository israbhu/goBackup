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
		//add nuances here
		Logger.Fatalf("Data.dat has been locked for access. Please properly close the other program. If you wish to delete the lock manually, delete the lock.pid file in the gobackup directory.")
	} else {
		lockfile, err := os.Create("lock.pid")
		CheckError(err, "There was an error creating lock.pid")

		//write the process id into the file
		io.WriteString(lockfile, fmt.Sprintf("%v", os.Getpid()))

		//close the file
		lockfile.Close()
	}
}

//creates a lock file for data.dat
func DeleteLock() {
	//if exist
	if FileExist("lock.pid") {
		err := os.Remove("lock.pid")
		CheckError(err, "Error deleting lock.pid")
	}
}

//check if a file called name exists
func FileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		//		fmt.Println("file does not exist")
		return false
	}
	return true
}

//checks the errors and delete lock********
func CheckError(err error, message string) {
	if err != nil {
		if message == "" {
			Logger.Fatalf("Error found! %v", err)
			DeleteLock()
		} else {
			Logger.Fatalf(message+" %v", err)
			DeleteLock()
		}
	}
}
