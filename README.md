goLocBackup is a program designed for two audiences. The first audience is a technically savvy user who wishes to backup their data. The second audience is for developers who wish to build upon the base that I have started.

What is this program designed to do?
This program was optimized to be used in a chron job. It can be used on the commandline as well, but I have optimized the program to be used with a preferences file. Every option that can be set via preferences file can be overwritten by the commandline options.

What are its limitations?
The program is currently limited to a hard cap of 25MB of stored data due to the nature of KV. The program currently supports zcompression and deflate, however, you must be sure the compressed data will be smaller than 25 MB or face truncation. In the future, I plan to upgrade the capabilities of the program to split a file into multiple 25MB and lower chunks.

The download capabilities are currently limited to a single download at a time via commandline with the knowledge of the key name assigned to the file. I plan to upgrade this using a download search option in the commandline which will allow you to search for your desired files.

What do you need to run this program?
This program requires a cloudflare free or higher tier account. It is preferred, but not necessary to have a configured workerskv token created on the cloudflare website.

You need a computer (virtual is good as well) and some data that you need backed up.

How do you install the program?
You can acquire the source code from: https://github.com/israbhu/goBackup

Run "go build cmd\goLockBackup\main.go" in the directory. On a Windows system, you can rename main.exe to gobackup.exe or another name of your choosing.

The commandline options are as follows:

	-email: "Set the User email instead of using any preferences file"
	-account", "", "Set the User Account instead of using any preferences file"
	-namespace", "", "Set the User's Namespace instead of using any preferences file"
	-key", "", "Set the Account Global Key instead of using any preferences file"
	-token", "", "Set the Configured KV Workers key instead of using any preferences file"
	-addLocation: "Add these locations/files to backup in addition to those set in the preferences file"
	-location: "Use only these locations to backup"
	-download: "Download files using the file's key. This option assumes you know how to find the file key via the data.dat file or the -keys option"

	-zip: "Set the zip compression to 'none', 'zstandard', or 'zip'"
	-pref: "use an alternate preference file"
	-dryrun: "Dry run. Goes through all the steps, but it makes no changes on disk or network"
	-dry:    "Dry run. Goes through all the steps, but it makes no changes on disk or network"
	-keys: "Get the keys and metadata from cloudflare"
	-sync: "Download the keys and metadata from cloud and overwrite the local database"

    Examples for use via go run:
    //upload files using the preferences file, do not use compression
    go run cmd\goLocBackup\main.go -zip none

    //upload files using the preferences.toml file one level below the current directory
    go run cmd\goLocBackup\main.go -pref ../preferences.toml

    Windows examples after following instructions for "How do you install the program?"
    //upload files using the preferences file, do not use compression
    gobackup.exe -zip none

    //upload files using the preferences.toml file one level below the current directory
    gobackup.exe -pref ../preferences.toml

Supported Platforms?
This program should be able to be compiled anywhere you can install the Go language.

More info
The project lives at github.com/israbhu/goBackup
You may contact the author at gobackup@israauthor.com