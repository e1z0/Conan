package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// path's where we should look for server list files
var serverFilesPaths []string

type Server struct {
	ID           string `yaml:"-"` // new unique identifier
	SourcePath   string `yaml:"-"` // full path, not marshalled
	SourceName   string `yaml:"-"` // basename, not marshalled
	Host         string `yaml:"host"`
	IP           string `yaml:"ip"`
	User         string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	PrivateKey   string `yaml:"privatekey,omitempty"`
	Port         string `yaml:"port,omitempty"`
	Description  string `yaml:"description,omitempty"`
	Type         string `yaml:"type"`
	Tags         string `yaml:"tags,omitempty"` // Comma-separated
	Availability string `yaml:"-"`              // e.g., "available", "unavailable"
}

func (s *Server) DecryptPassword() string {
	key := ""
	exists, gist := gistExists(s.SourceName)
	if exists {
		//log.Printf("Exists: %s %s\n", s.SourceName, gist.EncKey)
		key = gist.EncKey
	}
	pass, err := decryptString(s.Password, key)
	if err != nil {
		log.Printf("Decryption failed for server %s: %s\n", s.Host, err)
		return "" // Return empty string if decryption fails
	}
	return pass
}

func (s *Server) EncryptPassword(pass string) string {
	key := ""
	exists, gist := gistExists(s.SourceName)
	if exists {
		key = gist.EncKey
	}
	if pass == "" {
		return ""
	}
	encrypted, err := encryptString(pass, key)
	if err != nil {
		log.Printf("Error encrypting password for server %s: %s\n", s.Host, err)
		return pass // Return original if encryption fails
	}
	return encrypted
}

var ServerTypes = []string{"SSH", "RDP", "VNC", "Telnet", "Serial", "WINBOX"}

// TagsList returns the tags as a []string or nil if empty
func (s Server) TagsList() []string {
	if strings.TrimSpace(s.Tags) == "" {
		return nil
	}
	tags := strings.Split(s.Tags, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	return tags
}

// change the encryption key in the database file
func changeEncryptionKey(db, newkey string) error {
	if _, err := os.Stat(db); os.IsNotExist(err) {
		//log.Printf("Database file does not exist: %s\n", db)
		return err
	}
	data, err := os.ReadFile(db)
	if err != nil {
		//log.Printf("Error reading database file: %s\n", err)
		return err
	}
	var servers []Server
	if err := yaml.Unmarshal(data, &servers); err != nil {
		//log.Printf("Error unmarshalling database file: %s\n", err)
		return err
	}
	for i := range servers {
		//serv := servers[i]
		if servers[i].Password != "" {
			log.Printf("before pass: %#v\n", servers[i])
			decrypted, err := decryptString(servers[i].Password, "")
			if err != nil {
				log.Printf("Error decrypting password for server %s: %s\n", servers[i].Host, err)
				continue
			}
			newk, err := encryptString(decrypted, newkey)
			if err != nil {
				log.Printf("Error encrypting password for server %s: %s\n", servers[i].Host, err)
				continue
			}
			servers[i].Password = newk
			log.Printf("after pass: %#v\n", servers[i])
		}
	}
	data, err = yaml.Marshal(servers)
	if err != nil {
		return err
	}
	if err := os.WriteFile(db, data, 0644); err != nil {
		return err
	}
	return nil
}

// find available server configuration files
func findServerFiles() {
	ymlfiles = nil
	ignored := strings.Split(settings.Ignore, ",")
	log.Printf("Searching for server list files...\n")
	for _, dir := range serverFilesPaths {
		//log.Printf("searching in path: %s\n",dir)
		files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
		if err != nil {
			log.Printf("some error: %s\n", err)
			continue
		}
		for _, f := range files {
			//if !f.IsDir() {
			// Double-check if it's a file (not a dir)
			//if info, err := fs.Stat(os.DirFS("/"), f); err == nil && !info.IsDir() {
			log.Printf("Found possible servers list yml file: %s\n", f)
			if FindInArray(ignored, filepath.Base(f)) {
				log.Printf("Skipping file %s because it's defined as ignored\n", f)
				continue
			}
			for i, g := range gists {
				if g.Name == filepath.Base(f) {
					gists[i].Path = f // Update the Gist path
				}
			}
			ymlfiles = append(ymlfiles, f)

			//}
		}
	}
}

// check if servers list file exist in the specified path
func checkServYmlFiles(filename string) (error, string) {
	if filename == "" {
		return errors.New("File not specified"), ""
	}
	var err error
	if !strings.Contains(filename, ".yml") {
		filename = filename + ".yml"
	}
	ymlfiles = nil
	ignored := strings.Split(settings.Ignore, ",")
	for _, dir := range serverFilesPaths {
		fullPath := filepath.Join(dir, filename)
		log.Printf("checking %s for servers list file\n", fullPath)
		if FindInArray(ignored, filepath.Base(fullPath)) {
			log.Printf("Skipping file %s because it's defined as ignored\n", fullPath)
			continue
		}
		if _, err := os.Stat(fullPath); err == nil {
			log.Printf("Found in: %s\n", fullPath)
			ymlfiles = make([]string, 1)
			ymlfiles[0] = fullPath
			for i, g := range gists {
				if g.Name == filepath.Base(fullPath) {
					gists[i].Path = fullPath // Update the Gist path
				}
			}
			return nil, fullPath
		}
	}
	return err, ""
}

func fetchServersFromFiles() {
	tmpservs := make([]Server, 0)
	for _, file := range ymlfiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Printf("the servers list file does not exist: %s\n", err)
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %s\n", file, err)
			continue
		}
		var serversFromFile []Server
		if err := yaml.Unmarshal(data, &serversFromFile); err != nil {
			log.Printf("Error unmarshalling file %s: %s\n", file, err)
			continue
		}
		baseName := filepath.Base(file)
		for i, _ := range serversFromFile {
			serversFromFile[i].ID = uuid.NewString()
			serversFromFile[i].SourcePath = file     // Store the full path in each server struct
			serversFromFile[i].SourceName = baseName // Store the file path in each server struct
			//srv.File = baseName // Store the file path in the server struct
		}
		tmpservs = append(tmpservs, serversFromFile...)
	}
	servers = tmpservs
	filteredServers = servers // Initially show all servers
}

func pushServersToFile() {
	// 1) Group servers by their SourcePath
	byPath := make(map[string][]Server)
	for _, srv := range servers {
		if srv.SourcePath == "" {
			log.Printf("skip server %s: no SourcePath\n", srv.Host)
			continue
		}
		byPath[srv.SourcePath] = append(byPath[srv.SourcePath], srv)
	}

	// 2) For each path, marshal & write
	for path, list := range byPath {
		// marshal just that slice
		data, err := yaml.Marshal(list)
		if err != nil {
			log.Printf("Error marshalling %d servers for %s: %v\n", len(list), path, err)
			continue
		}

		// ensure the directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Error creating dir %s: %v\n", dir, err)
			continue
		}

		// atomic write: write to tmp then rename
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0600); err != nil {
			log.Printf("Error writing temp file %s: %v\n", tmp, err)
			continue
		}
		if err := os.Rename(tmp, path); err != nil {
			log.Printf("Error renaming %s → %s: %v\n", tmp, path, err)
			continue
		}

		log.Printf("Saved %d servers to %s\n", len(list), path)
	}
}
