package main

import (
	"bufio"
	"log"

	//	"encoding/csv"
	//	"bytes"

	"fmt"

	"os"
)

var version string
var build string
var debugging = "false"
var lines string

func main() {
	fmt.Printf("\n%s v%s (build: %s).\n\nCopyright (c) 2025 by Justinas K (e1z0@icloud.com)\n\n", RepresentativeName, version, build)

	if env.os == "windows" {
		gui, err := isWindowsGUI()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error detecting subsystem: %v\n", err)
			os.Exit(1)
		}
		if gui {
			fmt.Println("I was built as a GUI app!")
		} else {
			fmt.Println("I was built as a console app.")
		}
	}
	if env.os == "darwin" {
		//runtime.LockOSThread() // lock THIS goroutine to main thread
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runTUIv1() {
	log.Printf("to be implemented")
}

func runTUI() {
	log.Printf("Running TUI...\n")
	initTUI()
}

func runGUI() {
	log.Printf("Running GUI...\n")
	trayIcon()
	os.Exit(0)
}

func init() {
	if debugging == "true" {
		DEBUG = true
	}
	initlog()
}

// Pause function that waits for user to press Enter
func pause() {
	fmt.Print("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// Dummy jumpserver function
func jumpserver(srv Server) {
	ShowMessageBox("Error", "This feature is not supported yet!")
}
