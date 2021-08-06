package gobackup

import (
	"fmt"
	"io"
	"os"

	"github.com/golang/glog"
)

//creates a lock file for data.dat
func AddLock() {
	lockPath, err := MakeCanonicalPath("lock.pid")
	if err != nil {
		glog.Fatalf("While trying to create lock file 'lock.pid': %v", err)
	}
	//if exist
	if _, err := os.Stat(lockPath); err == nil {
		//add nuances here
		glog.Fatalf("Data.dat has been locked for access. Please properly close the other program. If you wish to delete the lock manually, delete 'lock.pid'.")
	}
	lockfile, err := os.Create(lockPath)
	if err != nil {
		glog.Fatalf("There was an error creating '%s': %v", lockPath, err)
	}
	defer lockfile.Close()

	//write the process id into the file
	_, err = io.WriteString(lockfile, fmt.Sprintf("%v", os.Getpid()))
	if err != nil {
		glog.Errorf("Error writing to the lock file '%s': %v", lockPath, err)
		lockfile.Close()
		if err := os.Remove(lockPath); err != nil {
			glog.Fatalf("While cleaning up lock file '%s': %v", lockPath, err)
		}
	}
}

//creates a lock file for data.dat
func DeleteLock() {
	lockPath, err := MakeCanonicalPath("lock.pid")
	if err != nil {
		glog.Fatalf("While trying to delete lock file 'lock.pid': %v", err)
	}
	glog.Infof("Attempting to delete '%s'", lockPath)
	if err := os.Remove("lock.pid"); err != nil {
		glog.Fatalf("While deleting lock file '%s': %v", lockPath, err)
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
