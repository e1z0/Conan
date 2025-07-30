package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

func firstStart() {
	if _, err := os.Stat(env.themeDir); os.IsNotExist(err) {
		err := os.MkdirAll(env.themeDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
		// Check if file exists
		defTheme := filepath.Join(env.themeDir, "default.ini")
		if _, err := os.Stat(defTheme); os.IsNotExist(err) {
			// File doesn't exist, create it and write content
			err := os.WriteFile(defTheme, []byte(defaultTheme), 0644)
			if err != nil {
				fmt.Println("Error creating file:", err)
				return
			}
		}
	}
	if _, err := os.Stat(env.settingsFile); os.IsNotExist(err) {
		// no settings file found, maybe new installation ?
		welcome = true
		err = touchFile(env.settingsFile)
		if err != nil {
			fmt.Printf("Unable to create settings file: %s\n", err)
			return
		}
	}

	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return
	}
	if !encrypted {
		cfg, err := ini.Load(env.settingsFile)
		if err != nil {
			fmt.Println("Failed to read settings file:", err)
			return
		}

		section := cfg.Section("General")
		if section.HasKey("enckey") {
			settings.GlobEncryptKey = section.Key("enckey").String()
		} else {
			fmt.Printf("Encryption key not found, generating new one...\n")
			key, err := encNewKey()
			if err != nil {
				fmt.Printf("Unable to generate encryption key: %s\n", err)
				return
			}

			// when first start and user does not have any yml files, we create the default yml file
			if len(ymlfiles) == 0 {
				// creatiing the default yml file
				defaultYml := filepath.Join(env.configDir, defaultYmlFilename)
				err = touchFile(defaultYml)
				if err != nil {
					fmt.Printf("Unable to create default yml file: %s\n", err)
				} // else {
				//	ymlfiles = append(ymlfiles, defaultYml)
				//}
			}

			section.Key("enckey").SetValue(key)
			err = cfg.SaveTo(env.settingsFile)
			if err != nil {
				fmt.Println("Failed to save settings file:", err)
				return
			}
		}
	}
}
