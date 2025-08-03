package main

// configStore provides a simple interface to read and write settings
// from an INI file. It supports both encrypted and unencrypted files.
// The settings file is loaded at startup and can be modified at runtime.
// (c) 2025 e1z0

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type configStore struct {
	cfg  *ini.File
	path string
}

// Global instance
var Store *configStore

func configInit(filepath string) error {
	cfg := &ini.File{}
	Store = &configStore{
		cfg:  cfg,
		path: filepath,
	}
	err := Store.Reload()
	if err != nil {
		return fmt.Errorf("failed to load settings file: %w", err)
	}

	return nil
}

func (s *configStore) isEncrypted() (bool, error) {
	raw, err := ioutil.ReadFile(s.path)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(string(raw), magic), nil
}

func (s *configStore) Reload() error {
	encrypted, err := s.isEncrypted()
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return err
	}
	if encrypted && settings.DecryptPassword == "" {
		return fmt.Errorf("settings file is encrypted, but no password provided")
	}

	diskCfg := &ini.File{}

	if encrypted {
		//diskCfg, err = LoadEncryptedINI(s.path, settings.DecryptPassword)
		raw, err := ioutil.ReadFile(s.path)
		if err != nil {
			return fmt.Errorf("read encrypted settings file error: %w", err)
		}
		content := string(raw)
		if strings.HasPrefix(content, magic) {
			// strip prefix and decrypt
			encB64 := strings.TrimPrefix(content, magic)
			dec, err := decryptAES(encB64, settings.DecryptPassword)
			if err != nil {
				return fmt.Errorf("decrypt ini error: %w", err)
			}
			content = dec
		}

		// now parse as plain INI text
		diskCfg, err = ini.LoadSources(ini.LoadOptions{}, []byte(content))
		if err != nil {
			return fmt.Errorf("parse ini error: %w", err)
		}
	} else {
		diskCfg, err = ini.LooseLoad(s.path)
	}
	if err != nil {
		return fmt.Errorf("failed to reload settings file: %w", err)
	}

	s.cfg = diskCfg
	return nil
}

func (s *configStore) save() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	encrypted, err := s.isEncrypted()
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return err
	}
	if encrypted && settings.DecryptPassword == "" {
		return fmt.Errorf("settings file is encrypted, but no password provided")
	}

	if encrypted {
		var buf bytes.Buffer
		if _, err := s.cfg.WriteTo(&buf); err != nil {
			return fmt.Errorf("serialize ini error: %w", err)
		}

		enc, err := encryptAES(buf.String(), settings.DecryptPassword)
		if err != nil {
			return fmt.Errorf("encrypt ini error: %w", err)
		}
		data := []byte(magic + enc)
		if err := ioutil.WriteFile(s.path, data, 0644); err != nil {
			return fmt.Errorf("write settings file error: %w", err)
		}
		return nil
	} else {
		return s.cfg.SaveTo(s.path)
	}
}

func (s *configStore) Set(section, key string, value interface{}) error {
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
Store.SetMany("Window-Notes", map[string]interface{}{
	"Width":     800,
	"Height":    600,
	"Splitter":  splitter.SaveState(),
	"DarkMode":  true,
	"ZoomLevel": 1.2,
})
*/
func (s *configStore) SetMany(section string, values map[string]interface{}) error {
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

// check if section exists
func (s *configStore) HasSection(section string) bool {
	_ = s.Reload()
	return s.cfg.HasSection(section)
}

// check if section and key exists
func (s *configStore) HasKey(section, key string) bool {
	_ = s.Reload()
	return s.cfg.Section(section).HasKey(key)
}

func (s *configStore) GetString(section, key string) (string, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return "", fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).String(), nil
}

func (s *configStore) GetInt(section, key string) (int, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Int()
}

func (s *configStore) GetBool(section, key string) (bool, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return false, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Bool()
}

func (s *configStore) GetFloat(section, key string) (float64, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return 0, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return sec.Key(key).Float64()
}

func (s *configStore) GetBytes(section, key string) ([]byte, error) {
	s.Reload()
	sec := s.cfg.Section(section)
	if !sec.HasKey(key) {
		return nil, fmt.Errorf("missing key: [%s]%s", section, key)
	}
	return base64.StdEncoding.DecodeString(sec.Key(key).String())
}
