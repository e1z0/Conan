package main

// SettingsStore provides a simple interface to read and write settings
// from an INI file. It supports both encrypted and unencrypted files.
// The settings file is loaded at startup and can be modified at runtime.
// (c) 2025 e1z0

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type SettingsStore struct {
	cfg  *ini.File
	path string
}

// Global instance
var Store *SettingsStore

func configInit() error {
	cfg := &ini.File{}
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return err
	}
	if encrypted {
		if settings.DecryptPassword == "" {
			return fmt.Errorf("settings file is encrypted, but no password provided")
		}
		cfg, err = LoadEncryptedINI(env.settingsFile, settings.DecryptPassword)
	}

	cfg, err = ini.LooseLoad(env.settingsFile)
	if err != nil {
		return err
	}
	Store = &SettingsStore{
		cfg:  cfg,
		path: env.settingsFile,
	}
	return nil
}

func (s *SettingsStore) Reload() error {
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return err
	}
	if encrypted && settings.DecryptPassword == "" {
		return fmt.Errorf("settings file is encrypted, but no password provided")
	}

	diskCfg := &ini.File{}

	if encrypted {
		diskCfg, err = LoadEncryptedINI(env.settingsFile, settings.DecryptPassword)
	} else {
		diskCfg, err = ini.LooseLoad(env.settingsFile)
	}
	if err != nil {
		return fmt.Errorf("failed to reload settings file: %w", err)
	}

	s.cfg = diskCfg
	return nil
}

func (s *SettingsStore) save() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return err
	}
	if encrypted && settings.DecryptPassword == "" {
		return fmt.Errorf("settings file is encrypted, but no password provided")
	}

	if encrypted {
		return SaveEncryptedINI(s.cfg, s.path, settings.DecryptPassword)
	} else {
		return s.cfg.SaveTo(s.path)
	}
}

func (s *SettingsStore) Set(section, key string, value interface{}) error {
	err := s.Reload()
	if err != nil {
		log.Printf("failed to reload settings file: %w", err)
	}
	sec := s.cfg.Section(section)
	var val string
	switch v := value.(type) {
	case string:
		val = v
	case bool:
		val = fmt.Sprintf("%v", v)
	case int, int64, int32:
		val = fmt.Sprintf("%d", v)
	case float64:
		val = fmt.Sprintf("%f", v)
	case []byte:
		val = base64.StdEncoding.EncodeToString(v)
	default:
		val = fmt.Sprintf("%v", v)
	}
	sec.Key(key).SetValue(val)
	return s.save()
}

// setMany sets multiple keys in a section.
/*
config.Store.SetMany("Window-Notes", map[string]interface{}{
	"Width":     800,
	"Height":    600,
	"Splitter":  splitter.SaveState(),
	"DarkMode":  true,
	"ZoomLevel": 1.2,
})
*/
func (s *SettingsStore) SetMany(section string, values map[string]interface{}) error {
	err := s.Reload()
	if err != nil {
		log.Printf("failed to reload settings file: %w", err)
	}
	sec := s.cfg.Section(section)
	for key, val := range values {
		switch v := val.(type) {
		case string:
			sec.Key(key).SetValue(v)
		case bool:
			sec.Key(key).SetValue(fmt.Sprintf("%v", v))
		case int, int64, int32:
			sec.Key(key).SetValue(fmt.Sprintf("%d", v))
		case float64:
			sec.Key(key).SetValue(fmt.Sprintf("%f", v))
		case []byte:
			sec.Key(key).SetValue(base64.StdEncoding.EncodeToString(v))
		default:
			sec.Key(key).SetValue(fmt.Sprintf("%v", v))
		}
	}
	return s.save()
}

func (s *SettingsStore) GetString(section, key string) (string, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return "", fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).String(), nil
}

func (s *SettingsStore) GetInt(section, key string) (int, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Int()
}

func (s *SettingsStore) GetBool(section, key string) (bool, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return false, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Bool()
}

func (s *SettingsStore) GetFloat(section, key string) (float64, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Float64()
}

func (s *SettingsStore) GetBytes(section, key string) ([]byte, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return nil, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return base64.StdEncoding.DecodeString(sec.Key(key).String())
}
