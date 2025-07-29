package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func sshConnectPutty(srv Server) {
	log.Printf("Putty handling function reached\n")
	args := []string{}
	var privkey string
	var user string
	var password string
	requirePass := false
	var passFullPath string
	if srv.PrivateKey != "" {
		privkey = srv.PrivateKey
	} else {
		privkey = settings.DefaultSSHKey
	}

	if srv.User == "" {
		user = "root"
	} else {
		user = srv.User
	}

	if privkey != "" {
		privkey = CmdParseTemplate(privkey)
		if _, err := os.Stat(privkey); err == nil {
			log.Printf("Found private key at: %s\n", privkey)
		} else {
			searchPaths := []string{env.appPath, filepath.Join(env.homeDir, ".ssh"), env.configDir}
			privatekey, err := FindFileInPaths(privkey, searchPaths)
			if err == nil {
				privkey = privatekey
			} else {
				log.Printf("Error finding private key: %s: %s\n", privkey, err)
			}
		}
	}

	password = srv.DecryptPassword()

	if password != "" {
		requirePass = true
		baseTempDir := filepath.Join(env.tmpDir, "conan")
		passfname := srv.ID + uuid.New().String() + ".tmp"
		// Step 2: Create the temp/conan directory if it doesn't exist
		err := os.MkdirAll(baseTempDir, 0o755)
		if err != nil {
			fmt.Println("Failed to create temp directory:", err)
			return
		}
		passFullPath = filepath.Join(baseTempDir, passfname)
		err = os.WriteFile(passFullPath, []byte(password), 0o644)
		if err != nil {
			fmt.Println("Failed to write to file:", err)
			return
		}

	}

	puttyPaths := []string{env.appPath,
		env.configDir,
		filepath.Join(env.appPath, "bundle"),
		filepath.Join(env.appPath,
			"..",
			"bundle",
			"windows")}
	putty, err := FindFileInPaths("putty.exe", puttyPaths)
	if err != nil {
		log.Printf("Unable to determine where the putty.exe exists... %s\n", err)
	}

	args = append(args, "-ssh")
	args = append(args, fmt.Sprintf("%s@%s", user, srv.IP))

	if srv.Port != "" {
		args = append(args, "-P")
		args = append(args, srv.Port)
	}

	if requirePass {
		args = append(args, "-pwfile")
		args = append(args, passFullPath)
	}
	if privkey != "" {
		args = append(args, "-i")
		args = append(args, privkey)
	}

	cmdline := strings.Join(args, " ")

	if requirePass {
		cmdline = strings.Replace(cmdline, password, "<redacted>", 1)
	}

	log.Printf("Executing command: %s %s\n", putty, cmdline)

	cmd := exec.Command(putty, args...)
	// Create pipes for stdout and stderr
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start: %v\n", err)
		return
	}

	// Log stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("[stdout] %s\n", scanner.Text())
		}
	}()

	// Log stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[stderr] %s\n", scanner.Text())
		}
	}()

	if requirePass {
		go func() {
			time.Sleep(2 * time.Second)
			err = os.Remove(passFullPath)
			if err != nil {
				log.Printf("Error removing file: %s\n", err)
				return
			}
			log.Printf("Temp file deleted")
		}()
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		log.Printf("Command finished with error: %v\n", err)
	} else {
		log.Printf("Command finished successfully")
	}
}
