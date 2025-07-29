package main

/* Notes backend processing
(c) e1z0 2025
*/

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// Snapshot represents a saved revision
type Snapshot struct {
	Timestamp time.Time
	Path      string
}

// Note holds a single markdown note and its metadata
type Note struct {
	ID      string     // relative path under NotesDir
	Path    string     // full file path
	Meta    NoteMeta   // parsed front-matter
	Body    []byte     // markdown content, no front-matter
	Raw     []byte     // full content on disk (decrypted if needed)
	History []Snapshot // list of previous snapshots
}

// NoteService handles file I/O, history, and encryption
type NoteService struct {
	NotesDir   string
	HistoryDir string
	Gist       GistConfig
}

// note metadata
type NoteMeta struct {
	Title     string    `yaml:"title"`
	Created   time.Time `yaml:"created"`
	Updated   time.Time `yaml:"updated"`
	Source    string    `yaml:"source"`
	Author    string    `yaml:"author"`
	Latitude  float64   `yaml:"latitude"`
	Longitude float64   `yaml:"longitude"`
	Altitude  float64   `yaml:"altitude"`
	Completed bool      `yaml:"completed"`
	Due       time.Time `yaml:"due"`
	Tags      []string  `yaml:"tags"`
	Sticky    bool      `yaml:"sticky"`
	Color     string    `yaml:"color"`
}

// Load reads, decrypts (if enabled), parses front-matter & history
func (s *NoteService) Load(relID string) (*Note, error) {
	full := filepath.Join(s.NotesDir, relID)
	raw, err := ioutil.ReadFile(full)
	if err != nil {
		return nil, err
	}
	raw = s.maybeDecrypt(raw)
	meta, body := stripYAMLFrontMatter(raw)

	// load history snapshots
	hDir := filepath.Join(s.NotesDir, s.HistoryDir, relID)
	snaps := []Snapshot{}
	files, _ := ioutil.ReadDir(hDir)
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if ts, err := time.Parse("20060102-150405", strings.TrimSuffix(name, ".md")); err == nil {
			snaps = append(snaps, Snapshot{Timestamp: ts, Path: filepath.Join(hDir, name)})
		}
	}
	// sort snapshots by time ascending
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.Before(snaps[j].Timestamp)
	})

	return &Note{ID: relID, Path: full, Meta: meta, Body: body, Raw: raw, History: snaps}, nil
}

func (s *NoteService) DisableSticky(n *Note) {
	n.Meta.Sticky = false
	s.Save(n)
}

// Save writes note with front-matter, snapshots old version, and encrypts if needed
func (s *NoteService) Save(n *Note) error {
	// create snapshot
	snapDir := filepath.Join(s.NotesDir, s.HistoryDir, n.ID)
	if err := os.MkdirAll(filepath.Dir(snapDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return err
	}
	ts := time.Now().Format("20060102-150405") + ".md"
	ioutil.WriteFile(filepath.Join(snapDir, ts), n.Raw, 0644)

	// update metadata timestamp
	n.Meta.Updated = time.Now().UTC()

	// render new content
	out, err := renderYAMLFrontMatter(n.Meta, n.Body)
	if err != nil {
		return err
	}
	out = s.maybeEncrypt(out)
	if err := ioutil.WriteFile(n.Path, out, 0644); err != nil {
		return err
	}
	// update raw
	n.Raw = out
	return nil
}

// ListTree returns maps for tree population
func (s *NoteService) ListTree() (map[string][]string, map[string]bool, error) {
	treeData := make(map[string][]string)
	isBranch := make(map[string]bool)
	treeData[""] = []string{}
	isBranch[""] = true

	err := filepath.WalkDir(s.NotesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(s.NotesDir, path)
		if err != nil || rel == "." {
			return nil
		}
		// skip history
		if rel == s.HistoryDir || strings.HasPrefix(rel, s.HistoryDir+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		parent := filepath.Dir(rel)
		if parent == "." {
			parent = ""
		}
		treeData[parent] = append(treeData[parent], rel)
		isBranch[rel] = d.IsDir()
		return nil
	})
	return treeData, isBranch, err
}

// Create a new note file under parentRel
func (s *NoteService) NewNote(parentRel, name string) error {
	parent := s.NotesDir
	if parentRel != "" {
		parent = filepath.Join(s.NotesDir, parentRel)
	}
	path := filepath.Join(parent, name+".md")
	log.Printf("Doing note: %s\n", path)
	now := time.Now().UTC()
	meta := NoteMeta{Title: name, Created: now, Updated: now}
	body := []byte(fmt.Sprintf("# %s\n", name))
	raw, err := renderYAMLFrontMatter(meta, body)
	if err != nil {
		return err
	}
	raw = s.maybeEncrypt(raw)
	return ioutil.WriteFile(path, raw, 0644)
}

// Create a new folder under parentRel
func (s *NoteService) NewFolder(parentRel, name string) error {
	parent := s.NotesDir
	if parentRel != "" {
		parent = filepath.Join(s.NotesDir, parentRel)
	}
	dir := filepath.Join(parent, name)
	return os.MkdirAll(dir, 0755)
}

// Delete removes note and history, and optionally from gist
func (s *NoteService) DeleteNote(relID string) error {
	// remove file
	if err := os.Remove(filepath.Join(s.NotesDir, relID)); err != nil {
		return err
	}
	// remove history
	hDir := filepath.Join(s.NotesDir, s.HistoryDir, relID)
	os.RemoveAll(hDir)

	return nil
}

// DeleteFromGist removes a single file (note) from the specified Gist.
// - relID: relative path to note.
// Returns an error if anything goes wrong.
func (s *NoteService) DeleteFromGist(relID string) error {
	log.Printf("delete from gist: %s\n", relID)
	ctx := context.Background()

	// 1) Create an authenticated GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.Gist.GistSec})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	safe := strings.ReplaceAll(relID, string(os.PathSeparator), "__")

	// 2) Build a raw body that tells GitHub to delete our target file.
	//    Notice filename → nil, which becomes JSON "filename": null
	body := map[string]interface{}{
		"files": map[string]interface{}{
			safe: nil,
		},
	}

	// 3) Create and send the PATCH request ourselves
	path := fmt.Sprintf("gists/%s", s.Gist.GistID)
	req, err := client.NewRequest("PATCH", path, body)
	if err != nil {
		return fmt.Errorf("creating PATCH request: %s", err)
	}
	resp, err := client.Do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("sending PATCH request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	return nil
}

// PushSync pushes all markdown notes under path to a Gist, preserving folder structure as encoded filenames.
// Since Gists only support a flat file list, directory separators are encoded as "__".
func (s *NoteService) PushSync() error {
	ctx := context.Background()
	// OAuth2 token source
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.Gist.GistSec})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	// collect files
	filesMap := make(map[github.GistFilename]github.GistFile)
	err := filepath.Walk(s.NotesDir, func(full string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(s.NotesDir, full)
		if err != nil {
			return err
		}

		// FIXME skip .history this time.. just for testing...
		// If any part of the path contains .history, skip it
		if strings.Contains(rel, string(os.PathSeparator)+s.HistoryDir+string(os.PathSeparator)) ||
			strings.HasPrefix(rel, ".history"+string(os.PathSeparator)) ||
			rel == ".history" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// only .md files
		if !strings.HasSuffix(rel, ".md") {
			return err
		}
		data, err := ioutil.ReadFile(full)
		if err != nil {
			return err
		}
		contentStr := string(data)
		// encrypt if key provided
		if s.Gist.EncKey != "" {
			enc, err := encryptAES(contentStr, s.Gist.EncKey)
			if err != nil {
				log.Printf("Unable to encrypt note: %s err: %s\n", full, err)
			}
			if err == nil {
				contentStr = enc
			}
		}
		// encode path separators so filenames remain unique
		safe := strings.ReplaceAll(rel, string(os.PathSeparator), "__")
		filesMap[github.GistFilename(safe)] = github.GistFile{
			Content: github.String(string(contentStr)),
		}
		return nil
	})
	if err != nil {
		return err
	}

	opt := &github.Gist{
		Files:       filesMap,
		Public:      github.Bool(false),
		Description: github.String("Notes sync for " + s.NotesDir),
	}

	var gist *github.Gist
	if s.Gist.GistID == "" {
		created, _, err := client.Gists.Create(ctx, opt)
		if err != nil {
			return err
		}
		gist = created
	} else {
		updated, _, err := client.Gists.Edit(ctx, s.Gist.GistID, opt)
		if err != nil {
			return err
		}
		gist = updated
	}
	// optionally store new ID back into cfg
	log.Println("Push Synced Gist ID:", gist.GetID())
	return nil
}

// PullSync fetches all markdown files from the given Gist and writes them into path, decoding folder structure.
func (s *NoteService) PullSync() error {
	fmt.Printf("Syncing pull notes: %s using %s\n", s.NotesDir, s.Gist)
	ctx := context.Background()
	// set up GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.Gist.GistSec})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	// retrieve the gist
	gist, _, err := client.Gists.Get(ctx, s.Gist.GistID)
	if err != nil {
		return err
	}
	// iterate through files
	for name, gf := range gist.Files {
		safe := string(name)
		// skip non-.md files
		if !strings.HasSuffix(safe, ".md") {
			log.Printf("skipping invalid file in gist notes: %s\n", safe)
			continue
		}
		rel := strings.ReplaceAll(safe, "__", string(os.PathSeparator))
		fullPath := filepath.Join(s.NotesDir, rel)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		content := gf.GetContent()
		if s.Gist.EncKey != "" {
			decr, err := decryptAES(content, s.Gist.EncKey)
			if err != nil {
				log.Printf("Error decrypting note: %s err: %s\n", safe, err)
				continue
			}
			content = decr
		}

		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	log.Println("Pull Synced Gist ID:", gist.GetID())
	return nil
}

func (s *NoteService) maybeDecrypt(data []byte) []byte {
	if s.Gist.EncKey != "" && isEncrypted(string(data)) {
		if dec, err := decryptWithMagic(string(data), s.Gist.EncKey); err == nil {
			return []byte(dec)
		}
	}
	return data
}

func (s *NoteService) maybeEncrypt(data []byte) []byte {
	if s.Gist.EncryptNotes && s.Gist.EncKey != "" {
		if enc, err := encryptWithMagic(string(data), s.Gist.EncKey); err == nil {
			return []byte(enc)
		}
	}
	return data
}

// stripYAMLFrontMatter and renderYAMLFrontMatter as before
func stripYAMLFrontMatter(raw []byte) (meta NoteMeta, body []byte) {
	raw = bytes.TrimLeft(raw, "\ufeff")
	if bytes.HasPrefix(raw, []byte("---\n")) {
		parts := bytes.SplitN(raw, []byte("---\n"), 3)
		if len(parts) == 3 {
			yaml.Unmarshal(parts[1], &meta)
			body = parts[2]
			return
		}
	}
	body = raw
	return
}

func renderYAMLFrontMatter(meta NoteMeta, body []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(meta); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.Write(body)
	return buf.Bytes(), nil
}
