package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var ()

const (
	GistAPI = "https://api.github.com/gists"
)

// GistRequest represents the JSON payload for updating a Gist
type GistRequest struct {
	Description string              `json:"description"`
	Files       map[string]GistFile `json:"files"`
}

type GistFile struct {
	Content string `json:"content"`
}

func findGist(name string) GistConfig {
	gist := GistConfig{}
	for _, v := range gists {
		if v.Name == name {
			return v
		}
	}
	return gist
}

func UploadGists() error {
	for i, v := range gists {
		if v.EncKey == "" {
			return fmt.Errorf("Gist %d (%s) has no encryption key set", i+1, v.Name)
		}
		if v.GistID == "" {
			return fmt.Errorf("Gist %d (%s) has no ID set", i+1, v.Name)
		}
		if v.GistSec == "" {
			return fmt.Errorf("Gist %d (%s) has no Secret ID set", i+1, v.Name)
		}
		err := uploadToGist(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func DownloadGists() error {
	for i, v := range gists {
		if v.EncKey == "" {
			return fmt.Errorf("Gist %d (%s) has no encryption key set", i+1, v.Name)
		}
		if v.GistID == "" {
			return fmt.Errorf("Gist %d (%s) has no ID set", i+1, v.Name)
		}
		if v.GistSec == "" {
			return fmt.Errorf("Gist %d (%s) has no Secret ID set", i+1, v.Name)
		}
		err := downloadFromGist(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Upload servers list to GitHub Gist
func uploadToGist(gist GistConfig) error {

	ymldata, err := os.ReadFile(gist.Path)
	if err != nil {
		return err
	}

	encrypted, err := encryptString(string(ymldata), gist.EncKey)

	if err != nil {
		return fmt.Errorf("error encrypting servers data for file %s: %s", gist.Path, err)
	}

	// Prepare JSON payload
	payload := GistRequest{
		Description: gist.Name,
		Files: map[string]GistFile{
			gist.Name: {Content: string(encrypted)},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/%s", GistAPI, gist.GistID)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+gist.GistSec)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %s", string(body))
	}

	fmt.Printf("✅ Servers list %s pushed to GitHub Gist successfully!\n", gist.Name)
	return nil
}

func downloadFromGist(gist GistConfig) error {
	// Request Gist
	url := fmt.Sprintf("%s/%s", GistAPI, gist.GistID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+gist.GistSec)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Parse JSON
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Extract content
	files := result["files"].(map[string]interface{})
	serversData := files[gist.Name].(map[string]interface{})["content"].(string)
	// Decrypt data
	decrypted, err := decryptString(serversData, gist.EncKey)
	if err != nil {
		return err
	}

	if gist.Path == "" {
		return fmt.Errorf("Unable to determine output yml file for gist output\n")
	}

	if !fileExists(gist.Path) {
		return fmt.Errorf("File does not exist %s\n", gist.Path)
	}

	// Open the file in append mode, create if it doesn't exist
	file, err := os.OpenFile(gist.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	// Write string to the end of the file
	_, err = file.WriteString(decrypted)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Servers list %s pulled from GitHub gist successfully!\n", gist.Name)
	return nil
}
