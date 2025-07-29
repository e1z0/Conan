package main

import (
	"debug/pe"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Try to open the file (create if doesn't exist)
func touchFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	// Update the modification and access time
	now := time.Now()
	return os.Chtimes(path, now, now)
}

// gets struct field/checks if it does exist
func getStructField(obj interface{}, field string) (string, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName(field)
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String(), true
	}
	return "", false
}

// return app path
func appPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}

	// Resolve any symlinks and clean path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return ""
	}
	return filepath.Dir(realPath)
}

// helper to find index by name
func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1 // not found
}

// baseNames returns the file name (with extension) for each full path.
func baseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return names
}

// fullPathFor returns the first full path in files whose base name equals basename.
// e.g. fullPathFor("file1.yml", []string{"/path/file1.yml", "/other/file2.yml"})
// returns "/path/file1.yml", nil.
func fullPathFor(basename string, files []string) (string, error) {
	for _, f := range files {
		if filepath.Base(f) == basename {
			return f, nil
		}
	}
	return "", fmt.Errorf("no YAML file named %q in slice", basename)
}

// fileExists returns true if the given path exists (and is not a directory).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir()
	}
	if os.IsNotExist(err) {
		return false
	}
	// some other error (e.g. permissions) — assume it “exists” so the caller can decide
	return true
}

// isRunningInAppBundle returns true if the executable lives inside an .app bundle.
func isRunningInAppBundle() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	// go up three levels: YourApp.app/Contents/MacOS/your_binary
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	return filepath.Ext(bundle) == ".app"
}

// isWinExecutable returns true if it's windows .exe file
func isWinExecutable() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	realPath, err := filepath.EvalSymlinks(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving symlink: %v\n", err)
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Ext(realPath), ".exe")
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return false
	}
	mode := info.Mode().Perm()
	return mode&0111 != 0
}

func doRestart() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	args := os.Args[1:]
	cmd := exec.Command(exe, args...)
	cmd.Start()
	os.Exit(0)
}

func FindInArray(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func trimYML(s string) string {
	if strings.HasSuffix(s, ".yml") {
		return strings.TrimSuffix(s, ".yml")
	}
	return s
}

func StringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// FindFileInPaths searches for a file in multiple directories.
// Returns the full path if found, or an empty string.
func FindFileInPaths(filename string, paths []string) (string, error) {
	for _, dir := range paths {
		fullPath := filepath.Join(dir, filename)
		log.Printf("Searching for %s in %s\n", filename, fullPath)
		if _, err := os.Stat(fullPath); err == nil {
			log.Printf("I have found %s in %s\n", filename, fullPath)
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("file %q not found in provided paths", filename)
}

func isWindowsGUI() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	f, err := pe.Open(exe)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var sub uint16
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		sub = oh.Subsystem
	case *pe.OptionalHeader64:
		sub = oh.Subsystem
	default:
		return false, fmt.Errorf("unknown OptionalHeader type")
	}

	const (
		IMAGE_SUBSYSTEM_WINDOWS_GUI = 2
		IMAGE_SUBSYSTEM_WINDOWS_CUI = 3
	)
	return sub == IMAGE_SUBSYSTEM_WINDOWS_GUI, nil
}

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		// Not enough space even for "..."
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func printThreadID(where string) {
	fmt.Println(where, "on thread", runtime.LockOSThread)
}

// ToLocalTime converts a time.Time (usually in UTC) to the local computer's timezone.
func ToLocalTime(t time.Time) time.Time {
	return t.In(time.Now().Location())
}
