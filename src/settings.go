package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

var RepresentativeName = "Conan" // name for demonstration purposes, lol
var appName = "conan"            // app name that will be used in settings etc...
var servers []Server             // In-memory list of servers
var gists []GistConfig           // Gist configuration for syncing
var welcome bool = false         // Welcome window flag
var filteredServers []Server     // filtered servers
var ymlfiles []string
var sshConnectionClients = []string{"external"}
var defaultYmlFilename = "servers.yml" // default yml file name when no other files are found

// var enckey string
var settings Settings
var DEBUG bool = false
var env Environment

type Environment struct {
	configDir    string // configuration directory ~/.config/conan
	settingsFile string // configuration path ~/.config/conan/settings.ini
	homeDir      string // home directory ~/
	themeDir     string // theme directory for TUI theme ~/.config/conan/themes
	appPath      string // application directory where the binary lies
	tmpDir       string // OS Temp directory
	appDebugLog  string // app debug.log
	os           string // current operating system
}

type GistConfig struct {
	Name         string
	Path         string
	GistID       string
	GistSec      string
	EncKey       string
	EncryptNotes bool
}

type GuiServTable struct {
	HostColumnSize         int
	TypeColumnSize         int
	IpColumnSize           int
	UserColumnSize         int
	DescriptionColumnSize  int
	TagsColumnSize         int
	SourceColumnSize       int
	AvailabilityColumnSize int
	DisableTooltips        bool
	DisableRowTooltips     bool
}

type NoteSettings struct {
	AlwaysOnTop bool
}

func NewServTableColumnsSizes() *GuiServTable {
	return &GuiServTable{
		HostColumnSize:         130,
		TypeColumnSize:         80,
		IpColumnSize:           115,
		UserColumnSize:         80,
		DescriptionColumnSize:  120,
		TagsColumnSize:         150,
		SourceColumnSize:       100,
		AvailabilityColumnSize: 85,
	}
}

type Settings struct {
	GlobEncryptKey  string
	SSHClient       string
	SSHCommand      string
	RDPCommand      string
	WINBOXCommand   string
	DefaultSSHKey   string
	Sync            bool
	ServerTableGui  GuiServTable
	Ignore          string
	DecryptPassword string
	NotesSettings   NoteSettings
	//GistID        string
	//GistSecret    string
	//DefaultDB     string
}

func decryptSettings() {
	loadSettings(settings.DecryptPassword)
}

func InitializeEnvironment() {
	// gather all required directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Unable to determine the user home folder: %s\n", err)
	}
	configDir := filepath.Join(homeDir, ".config", appName)
	settingsFile := filepath.Join(configDir, "settings.ini")
	environ := Environment{
		configDir:    configDir,
		settingsFile: settingsFile,
		homeDir:      homeDir,
		themeDir:     filepath.Join(configDir, "themes"),
		appPath:      appPath(),
		tmpDir:       os.TempDir(),
		appDebugLog:  filepath.Join(configDir, "debug.log"),
		os:           GetOS(),
	}
	env = environ
	// collect and set possible server .yml files locations
	serverFilesPaths = []string{
		filepath.Join(env.configDir, "servers"),
		env.configDir,
		env.appPath,
	}
	switch env.os {
	case "windows":
		sshConnectionClients = append(sshConnectionClients, "putty")
	case "darwin":
		sshConnectionClients = append(sshConnectionClients, "iTerm")
	case "linux":
		// TBD
	default:
		break
	}

}

// return ini file as object
func IniLoadMemory() (*ini.File, error) {
	cfg := ini.Empty()
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		return cfg, err
	}
	if settings.DecryptPassword != "" {
		cfg, err = LoadEncryptedINI(env.settingsFile, settings.DecryptPassword)
		if err != nil {
			return cfg, err
		}
	} else {
		if encrypted {
			return cfg, fmt.Errorf("settings file is encrypted, please provide a passphrase")
		}
		cfg, err = ini.Load(env.settingsFile)
		if err != nil {
			log.Printf("Failed to read settings file: %s\n", err)
			return cfg, err
		}

	}
	return cfg, nil
}

func SaveInitFile(cfg *ini.File) error {
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		return err
	}
	if encrypted {
		if err := SaveEncryptedINI(cfg, env.settingsFile, settings.DecryptPassword); err != nil {
			return err
		}
	} else {
		cfg.SaveTo(env.settingsFile)

	}
	return nil
}

func loadSettings(passphrase string) {
	var (
		cfg *ini.File
		err error
	)
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return
	}
	if passphrase != "" {
		cfg, err = LoadEncryptedINI(env.settingsFile, passphrase)
		if err != nil {
			log.Printf("Unable to load encrypted settings file %s\n", err)
		}
	} else {
		if encrypted {
			log.Printf("Settings file is encrypted, needs to request user's password\n")
			return
		}
		cfg, err = ini.Load(env.settingsFile)
		if err != nil {
			log.Printf("Failed to read settings file: %s\n", err)
			return
		}

	}

	section := cfg.Section("General")
	sshKey := fmt.Sprintf("%s_ssh", GetOS())
	rdpKey := fmt.Sprintf("%s_rdp", GetOS())
	winboxKey := fmt.Sprintf("%s_winbox", GetOS())
	if section.HasKey("ssh_client") {
		settings.SSHClient = section.Key("ssh_client").String()
	} else {
		settings.SSHClient = "builtin"
	}
	if section.HasKey("enckey") {
		settings.GlobEncryptKey = section.Key("enckey").String()
	}
	if section.HasKey(sshKey) {
		settings.SSHCommand = section.Key(sshKey).String()
	} else {
		log.Printf("You have not defined the ssh client command in your settings file\n")
		log.Printf("Please define %s in %s/settings.ini [General] section\n", sshKey, env.configDir)
	}
	if section.HasKey(rdpKey) {
		settings.RDPCommand = section.Key(rdpKey).String()
	} else {
		log.Printf("You have not defined the rdp client command in your settings file\n")
		log.Printf("Please define %s in %s/settings.ini [General] section\n", rdpKey, env.configDir)
	}
	if section.HasKey(winboxKey) {
		settings.WINBOXCommand = section.Key(winboxKey).String()
	} else {
		log.Printf("You have not defined the winbox client command in your settings file\n")
		log.Printf("Please define %s in %s/settings.ini [General] section\n", winboxKey, env.configDir)
	}
	if section.HasKey("sync") {
		settings.Sync = section.Key("sync").MustBool(false)
		//	settings.GistID = section.Key("gistid").MustString("")
		//	settings.GistSecret = section.Key("gistsecret").MustString("")
	}
	if section.HasKey("ignore") {
		settings.Ignore = section.Key("ignore").String()
	}
	if section.HasKey("defaultsshkey") {
		settings.DefaultSSHKey = section.Key("defaultsshkey").String()
	}
	settings.ServerTableGui = *NewServTableColumnsSizes()
	if cfg.HasSection("ServersTable") {
		section = cfg.Section("ServersTable")
		if section.HasKey("HostColumn") {
			settings.ServerTableGui.HostColumnSize = section.Key("HostColumn").MustInt()
		}
		if section.HasKey("TypeColumn") {
			settings.ServerTableGui.TypeColumnSize = section.Key("TypeColumn").MustInt()
		}
		if section.HasKey("IpColumn") {
			settings.ServerTableGui.IpColumnSize = section.Key("IpColumn").MustInt()
		}
		if section.HasKey("UserColumn") {
			settings.ServerTableGui.UserColumnSize = section.Key("UserColumn").MustInt()
		}
		if section.HasKey("DescriptionColumn") {
			settings.ServerTableGui.DescriptionColumnSize = section.Key("DescriptionColumn").MustInt()
		}
		if section.HasKey("TagsColumn") {
			settings.ServerTableGui.TagsColumnSize = section.Key("TagsColumn").MustInt()
		}
		if section.HasKey("SourceColumn") {
			settings.ServerTableGui.SourceColumnSize = section.Key("SourceColumn").MustInt()
		}
		if section.HasKey("AvailabilityColumn") {
			settings.ServerTableGui.AvailabilityColumnSize = section.Key("AvailabilityColumn").MustInt()
		}
		if section.HasKey("disabletooltips") {
			settings.ServerTableGui.DisableTooltips = section.Key("disabletooltips").MustBool()
		}
		if section.HasKey("disablerowtooltips") {
			settings.ServerTableGui.DisableRowTooltips = section.Key("disablerowtooltips").MustBool()
		}
	}
	if cfg.HasSection("notes") {
		section = cfg.Section("notes")
		if section.HasKey("alwaysontop") {
			settings.NotesSettings.AlwaysOnTop = section.Key("alwaysontop").MustBool()
		}
	}
	// should be initialized as nil because if we run loadsettings few times the gist array becomes huge... :D
	gists = nil
	ignored := strings.Split(settings.Ignore, ",")
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" || section.Name() == "General" || section.Name() == "ServersTable" || section.Name() == "Notes" {
			continue
		}

		if strings.Contains(section.Name(), "Sticky ") {
			continue
		}

		if strings.Contains(section.Name(), "Window-") {
			continue
		}

		if !section.HasKey("gistid") || !section.HasKey("gistsec") {
			continue
		}

		if FindInArray(ignored, section.Name()[5:]) {
			continue
		}

		name := section.Name()[5:] // remove "gist "
		gists = append(gists, GistConfig{
			Name:         name,
			GistID:       section.Key("gistid").String(),
			GistSec:      section.Key("gistsec").String(),
			EncKey:       section.Key("enckey").String(),
			EncryptNotes: section.Key("encrypt_notes").MustBool(),
		})
	}
	if encrypted {
		log.Printf("Decryption key loaded, settings loaded, re-checking yml files of the servers...\n")
		if dbFlag != "" {
			if err, _ := checkServYmlFiles(dbFlag); err != nil {
				log.Printf("Servers DB Not found at: %s: %w\n", dbFlag, err)
				os.Exit(1)
			}
		} else {
			findServerFiles()
		}
		//findServerFiles()
		fetchServersFromFiles()

	}
}

func gistExists(name string) (bool, GistConfig) {
	gist := GistConfig{}
	for _, g := range gists {
		//log.Printf("Does gist %s exist? in: %s\n", g.Name, name)
		if g.Name == name {
			//log.Printf("yes it exists: %s\n", g.Name)
			return true, g
		}
	}
	return false, gist
}

// IsEncryptedINI reads the given filename and returns true if it
// detects the magic prefix (i.e. we wrote it with SaveEncryptedINI).
func IsEncryptedINI(filename string) (bool, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(string(raw), magic), nil
}

// SaveEncryptedINI serializes cfg to INI in memory, encrypts it with passphrase,
// and writes "<magic><base64…>" to filename.
func SaveEncryptedINI(cfg *ini.File, filename, passphrase string) error {
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return fmt.Errorf("serialize ini: %w", err)
	}

	enc, err := encryptAES(buf.String(), passphrase)
	if err != nil {
		return fmt.Errorf("encrypt ini: %w", err)
	}

	data := []byte(magic + enc)
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// LoadEncryptedINI reads filename, checks for magic prefix,
// decrypts if needed (using passphrase), then parses the INI
// and returns the *ini.File.
func LoadEncryptedINI(filename, passphrase string) (*ini.File, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	content := string(raw)
	if strings.HasPrefix(content, magic) {
		// strip prefix and decrypt
		encB64 := strings.TrimPrefix(content, magic)
		dec, err := decryptAES(encB64, passphrase)
		if err != nil {
			return nil, fmt.Errorf("decrypt ini: %w", err)
		}
		content = dec
	}

	// now parse as plain INI text
	cfg, err := ini.LoadSources(ini.LoadOptions{
		// your options here
	}, []byte(content))
	if err != nil {
		return nil, fmt.Errorf("parse ini: %w", err)
	}
	//log.Printf("cfg: %s\n", cfg)
	return cfg, nil
}
