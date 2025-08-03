package main

import (
	// "fmt"
	"bufio"
	"bytes"
	"html/template"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

func GetOS() string {
	return runtime.GOOS
}

func ConnectCommand(srv Server, tp string) {
	raw, ok := getStructField(settings, tp)
	if !ok {
		log.Printf("You have not declared key: %s in settings for running %s command", tp)
		return
	}

	server := srv
	Password := srv.DecryptPassword()
	if server.User == "" {
		switch GetOS() {
		case "windows":
			server.User = "administrator"
		case "linux":
			server.User = "root"
		default:
			server.User = "root"
		}
	}
	sshkeytpl := struct {
		Home      string
		AppDir    string
		ConfigDir string
	}{
		Home:      env.homeDir,
		AppDir:    env.appPath,
		ConfigDir: env.configDir,
	}

	defaultsshkey := func(tpl string) string {
		tmpl, err := template.New("sshkey").
			Option("missingkey=default").
			Parse(tpl)
		if err != nil {
			log.Printf("Invalid defaultsshkey template in settings.ini (parse): %v — using literal", err)
			return tpl
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, sshkeytpl); err != nil {
			log.Printf("Invalid defaultsshkey template in settings.ini (exec): %v — using literal", err)
			return tpl
		}

		return buf.String()
	}

	// add any extra context your template needs:
	data := struct {
		Server
		Password   string
		Home       string
		AppDir     string
		ConfigDir  string
		DefaultKey string
	}{
		Server:     server,
		Password:   Password,
		Home:       env.homeDir,
		AppDir:     env.appPath,
		ConfigDir:  env.configDir,
		DefaultKey: defaultsshkey(settings.DefaultSSHKey),
	}

	cmdparams := func(tpl string) string {
		tmpl, err := template.New("cmd").
			Option("missingkey=default").
			Parse(tpl)
		if err != nil {
			log.Printf("Invalid Command line parameters template in settings.ini (parse): %v — using literal", err)
			return tpl
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("Invalid Command line parameters template in settings.ini (exec): %v — using literal", err)
			return tpl
		}
		return buf.String()
	}

	cmdline := cmdparams(raw)

	if Password != "" {
		redacted := strings.Replace(cmdline, Password, "<redacted>", 1)
		log.Printf("Executing command: %s\n", redacted)
	}
	cmdarr := strings.Split(cmdline, " ")

	cmd := exec.Command(cmdarr[0], cmdarr[1:]...)

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

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		log.Printf("Command finished with error: %v\n", err)
	} else {
		log.Printf("Command finished successfully")
	}

}

func ClientConnect(srv Server) {
	// enumarate between types, ssh, telnet etc...
	log.Printf("Connecting to server %s\n", srv.Host)
	switch srv.Type {
	case "SSH":
		if settings.SSHClient == "putty" {
			sshConnectPutty(srv)
		} else if settings.SSHClient == "iTerm" {
			sshConnectIterm(srv)
		} else {
			ConnectCommand(srv, "SSHCommand")
		}
	case "RDP":
		ConnectCommand(srv, "RDPCommand")
	case "WINBOX":
		ConnectCommand(srv, "WINBOXCommand")
	default:
		log.Printf("This type of server is not supported yet!\n")
		if GUIMODE {
			CallOnQtMain(func() {
				QTshowError(nil, "Error", "This type of server is not supported yet!")
			})
		}
		return
	}
}
