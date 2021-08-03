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
	if fi, err := os.Stat("lock.pid"); err == nil {
		//add nuances here
		p := MustMakeCanonicalPath(fi.Name())
		glog.Fatalf("Data.dat has been locked for access. Please properly close the other program. If you wish to delete the lock manually, delete %s.", p)
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
	if _, err := os.Stat("lock.pid"); err == nil {
		err := os.Remove("lock.pid")
		NoErrorFound(err, "Error deleting lock.pid")
	}
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
