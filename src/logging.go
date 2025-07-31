package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

func initlog() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Unable to retrieve home directory: %s\n", err)
		return
	}
	// create directory if it does not exist
	dir := filepath.Join(home, ".config", "conan")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	// Open the log file
	file, err := os.OpenFile(dir+"/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
		return
	}
	//    (we always write to file; if DEBUG=true we could also write to stdout)
	if debugging == "true" {
		log.SetOutput(io.MultiWriter(file, os.Stdout))
	} else {
		log.SetOutput(io.MultiWriter(file))
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
