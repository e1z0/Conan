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

func (s *SettingsStore) save() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	return s.cfg.SaveTo(s.path)
}

func (s *SettingsStore) Set(section, key string, value interface{}) error {
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

func (s *SettingsStore) GetString(section, key string) (string, error) {
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return "", fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).String(), nil
}

func (s *SettingsStore) GetInt(section, key string) (int, error) {
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Int()
}

func (s *SettingsStore) GetBool(section, key string) (bool, error) {
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return false, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Bool()
}

func (s *SettingsStore) GetFloat(section, key string) (float64, error) {
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Float64()
}

func (s *SettingsStore) GetBytes(section, key string) ([]byte, error) {
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return nil, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return base64.StdEncoding.DecodeString(sec.Key(key).String())
}
