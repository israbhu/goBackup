package gobackup

import (
	"fmt"
	"io"
	"os"

	"github.com/golang/glog"
)

//creates a lock file for data.dat
func AddLock() {
	//if exist
	if FileExist("lock.pid") {

		workingDirectory, _ := os.Getwd()
		//add nuances here
		glog.Fatalf("Data.dat has been locked for access. Please properly close the other program. If you wish to delete the lock manually, delete the %v file in the gobackup directory.", workingDirectory+string(os.PathSeparator)+"lock.pid")
	} else {
		lockfile, err := os.Create("lock.pid")
		NoErrorFound(err, "There was an error creating lock.pid")

		defer lockfile.Close()

		//write the process id into the file
		_, err = io.WriteString(lockfile, fmt.Sprintf("%v", os.Getpid()))
		NoErrorFound(err, "Error writing the processid to the lock file!")
	}
}

//creates a lock file for data.dat
func DeleteLock() {
	glog.Infoln("Attempting to delete lock!")
	if FileExist("lock.pid") {
		err := os.Remove("lock.pid")
		NoErrorFound(err, "Error deleting lock.pid")
	}
}

//check if a file called name exists
func FileExist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		//		glog.Fatalf("The file at %v does not exist", name)
		return false
	}
	return true
}

//checks for errors
//true means no errors were found
//false means an error was found
func NoErrorFound(err error, message string) bool {
	if err != nil {
		if message == "" {
			glog.Infof("Error found! %v", err)
			return false
		} else {
			glog.Infof(message+" %v", err)
			return false
		}
	}
	return true
}
