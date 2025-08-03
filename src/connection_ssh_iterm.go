package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// connectionSSHITerm is a placeholder for iTerm2 specific SSH connection handling.
func sshConnectIterm(server Server) error {
	// iTerm2 specific SSH connection logic goes here.
	log.Printf("Connecting to server %s using iTerm2 specific logic", server.Host)

	// Build user@host
	target := server.IP
	if server.User != "" {
		target = fmt.Sprintf("%s@%s", server.User, server.IP)
	}

	// Start building SSH args
	args := []string{"ssh"}

	// Use sshpass if password is present
	password := server.DecryptPassword()
	if password != "" {
		//args = append([]string{"sshpass", "-p", server.Password, "ssh"}, args[1:]...)
		tmpFile, err := os.CreateTemp("", "sshpass")
		if err != nil {
			return err
		}
		tmpFile.WriteString(password)
		tmpFile.Close()

		args = append([]string{"sshpass", "-f", tmpFile.Name(), "ssh"}, args[1:]...)
		// Schedule secure delete after a few seconds
		go func(path string) {
			time.Sleep(5 * time.Second)
			os.Remove(path)
		}(tmpFile.Name())
	}

	// Add port if specified
	if server.Port != "" {
		args = append(args, "-p", server.Port)
	}

	// no strict host key checking
	// This is not recommended for production use, but useful for testing.
	args = append(args,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	)

	// Identity file
	key := ""
	if server.PrivateKey != "" {
		key = server.PrivateKey
	} else if settings.DefaultSSHKey != "" {
		key = settings.DefaultSSHKey
	}
	if key != "" {
		key = CmdParseTemplate(key)
		if _, err := os.Stat(key); err == nil {
			log.Printf("Found private key at: %s\n", key)
		} else {
			searchPaths := []string{env.appPath, filepath.Join(env.homeDir, ".ssh"), env.configDir}
			privatekey, err := FindFileInPaths(key, searchPaths)
			if err == nil {
				key = privatekey
			} else {
				log.Printf("Error finding private key: %s: %s\n", key, err)
			}
		}
		args = append(args, "-i", key)
	}

	// Final target
	args = append(args, target)

	wait := 2 // seconds to wait before closing the tab
	escapedCommand := strings.ReplaceAll(strings.Join(args, " "), `"`, `\"`)
	fullCommand := fmt.Sprintf(`clear && echo "Connecting to %s..." && %s ; sleep %d ; exit`, server.Host, escapedCommand, wait)

	// Prepare osascript command
	escaped := escapeAppleScriptString(fullCommand)
	// AppleScript to open iTerm and run command in new tab
	cmd := exec.Command("osascript",
		"-e", `tell application "iTerm" to activate`,
		"-e", `tell application "iTerm" to tell current window to create tab with default profile`,
		"-e", fmt.Sprintf(`tell application "iTerm" to tell current session of current window to write text "%s"`, escaped),
	)

	// Attach stdout/stderr to log
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	// Start the command asynchronously
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start osascript: %w", err)
	}

	// Log and return without waiting
	log.Printf("SSH command dispatched to iTerm: %s", fullCommand)

	return nil
}

func escapeAppleScriptString(s string) string {
	// Escape backslashes first
	s = strings.ReplaceAll(s, `\`, `\\`)
	// Escape double quotes
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
