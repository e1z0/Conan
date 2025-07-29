package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mappu/miqt/qt"
	"golang.org/x/crypto/scrypt"
)

// exportConfig prompts user for a password, compresses configDir, encrypts it, and saves to file.
func exportConfig(parent *qt.QWidget) {
	// 1) Ask for password & confirmation using QDialog
	dlg := qt.NewQDialog(parent)
	dlg.SetWindowTitle("Export Settings – Set Password")
	layout := qt.NewQFormLayout(dlg.QWidget)

	pw1 := qt.NewQLineEdit(nil)
	pw1.SetEchoMode(qt.QLineEdit__Password)
	pw2 := qt.NewQLineEdit(nil)
	pw2.SetEchoMode(qt.QLineEdit__Password)

	layout.AddRow3("Password", pw1.QWidget)
	layout.AddRow3("Confirm", pw2.QWidget)

	btnBox := qt.NewQDialogButtonBox5(
		qt.QDialogButtonBox__Ok|qt.QDialogButtonBox__Cancel,
		qt.Horizontal,
	)

	layout.AddWidget(btnBox.QWidget)
	dlg.SetLayout(layout.QLayout)

	// OK handler
	btnBox.OnAccepted(func() {
		if pw1.Text() == "" || pw1.Text() != pw2.Text() {
			QTshowWarn(dlg.QWidget, "Error", "Passwords do not match or are empty.")
			return
		}
		password := pw1.Text()
		dlg.Hide()

		// 2) ZIP configDir into memory
		zipData, err := compressDirToBuffer(env.configDir)
		if err != nil {
			QTshowWarn(parent, "Error", err.Error())
			return
		}
		// 3) Encrypt with AES-GCM / scrypt key
		sealed, err := encrypt(zipData, password)
		if err != nil {
			QTshowWarn(parent, "Error", err.Error())
			return
		}

		// 4) Ask where to save
		ts := time.Now().Format("20060102-150405")
		defaultName := "conan-settings-" + ts + ".cnn"
		fileDlg := qt.NewQFileDialog6(parent, "Select Encrypted Conan Settings File", "", "Conan Files (*.cnn)")
		fileDlg.SetAcceptMode(qt.QFileDialog__AcceptSave)
		fileDlg.SelectFile(defaultName)
		if fileDlg.Exec() == int(qt.QDialog__Accepted) {
			files := fileDlg.SelectedFiles()
			if len(files) == 0 {
				return
			}
			filename := files[0]
			if err := os.WriteFile(filename, sealed, 0o644); err != nil {
				QTshowWarn(parent, "Error", "Could not write file: "+err.Error())
				return
			}
			QTshowInfo(parent, "Export Complete", "Settings exported successfully.")
		}
	})

	btnBox.OnRejected(func() {
		dlg.Reject()
	})

	dlg.SetModal(true)
	dlg.Resize(320, 140)
	// Optionally center on parent/screen here
	dlg.Exec()
}

// compressDirToBuffer walks `dir`, zips all files, closes the writer, then returns buf.Bytes().
func compressDirToBuffer(srcDir string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Walk the directory tree
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Build the zip entry name by making the path relative to srcDir
		relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		// If it's a directory, create a folder entry (so empty dirs are preserved)
		if d.IsDir() {
			// A directory entry must end in "/"
			_, err := zw.Create(relPath + "/")
			return err
		}
		// Otherwise it's a file: open and copy its contents
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		// Create a zip header for this file
		fw, err := zw.Create(relPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(fw, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		zw.Close() // make sure to close before returning
		return nil, err
	}
	// Finish writing the ZIP
	if err := zw.Close(); err != nil {
		return nil, err
	}

	// figure out program directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot determine executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// write the ZIP file there
	zipName := fmt.Sprintf("%s.zip", filepath.Base(srcDir))
	zipPath := filepath.Join(exeDir, zipName)
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("writing zip to %s: %w", zipPath, err)
	}

	return buf.Bytes(), nil
}

// encrypt derives a 256-bit key via scrypt and seals plaintext with AES-GCM.
// Output = [16-byte salt][12-byte nonce][ciphertext].
func encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, len(salt)+len(nonce)+len(ct))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// ---------------------------------------------------------------------------------
// importConfigFromFile reads a file, decrypts it, and extracts the config files.
func importConfig(parent *qt.QWidget) {
	// 1. Show file open dialog for .cnn file
	fileDlg := qt.NewQFileDialog6(parent, "Select Encrypted Conan Settings File", "", "Conan Settings (*.cnn)")
	fileDlg.SetFileMode(qt.QFileDialog__ExistingFile)
	if fileDlg.Exec() != int(qt.QDialog__Accepted) {
		return
	}
	files := fileDlg.SelectedFiles()
	if len(files) == 0 {
		return
	}
	filePath := files[0]

	// 2. Read file data
	data, err := os.ReadFile(filePath)
	if err != nil {
		QTshowWarn(parent, "Error", fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// 3. Ask for password
	pwDlg := qt.NewQInputDialog(parent)
	pwDlg.SetLabelText("Import Settings – Enter Password")
	pwDlg.SetTextEchoMode(qt.QLineEdit__Password)
	pwDlg.Resize(340, 120)
	if pwDlg.Exec() != int(qt.QDialog__Accepted) {
		return
	}
	pw := pwDlg.TextValue()
	if pw == "" {
		QTshowWarn(parent, "Error", "Password cannot be empty.")
		return
	}

	// 4. Decrypt
	plain, err := decrypt(data, pw)
	if err != nil {
		QTshowWarn(parent, "Error", fmt.Sprintf("Decrypt failed: %v", err))
		return
	}

	// 5. Unzip into configDir
	if err := decompressZipToDir(plain, env.configDir); err != nil {
		QTshowWarn(parent, "Error", fmt.Sprintf("Failed to decompress settings: %v", err))
		return
	}

	// 6. Confirm dialog for restart
	confirmDlg := qt.NewQMessageBox6(qt.QMessageBox__Information, "Imported", "Settings imported. Restart now?", qt.QMessageBox__Yes|qt.QMessageBox__No, parent)
	ret := confirmDlg.Exec()
	if ret == int(qt.QMessageBox__Yes) {
		doRestart()
	}
}

// decrypt reverses the encrypt operation: extracts salt, nonce, then AES-GCM opens.
func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 16+12 {
		return nil, errors.New("data too short")
	}
	salt := data[:16]
	nonce := data[16 : 16+12]
	ct := data[16+12:]
	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}

// decompressZipToDir unpacks a ZIP archive from buf into dir, recreating structure.
func decompressZipToDir(zipData []byte, destDir string) error {
	readerAt := bytes.NewReader(zipData)
	zr, err := zip.NewReader(readerAt, int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("reading zip data: %w", err)
	}

	// 1) Detect common root folder
	var root string
	for _, f := range zr.File {
		clean := filepath.Clean(f.Name)
		parts := strings.SplitN(clean, string(os.PathSeparator), 2)
		// record first component
		if root == "" {
			root = parts[0]
		} else if parts[0] != root {
			root = "" // mixed roots → don't strip
			break
		}
	}

	// 2) Extract entries, stripping root if set
	for _, f := range zr.File {
		clean := filepath.Clean(f.Name)

		// if we have a root to strip, and this entry begins with it, drop that segment
		if root != "" {
			prefix := root + string(os.PathSeparator)
			if strings.HasPrefix(clean, prefix) {
				clean = strings.TrimPrefix(clean, prefix)
			} else {
				// entry is the root folder itself (no "/" suffix), skip
				continue
			}
		}

		if clean == "" {
			// e.g. the top-level folder entry—nothing to do
			continue
		}

		// 3) Build destination path
		outPath := filepath.Join(destDir, clean)

		// ZipSlip protection
		if !strings.HasPrefix(outPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", outPath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
				return fmt.Errorf("creating directory %s: %w", outPath, err)
			}
			continue
		}

		// ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(outPath), os.ModePerm); err != nil {
			return fmt.Errorf("creating parent for %s: %w", outPath, err)
		}

		// extract file
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening %s: %w", f.Name, err)
		}
		defer rc.Close()

		outFile, err := os.OpenFile(outPath,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			f.Mode())
		if err != nil {
			return fmt.Errorf("creating file %s: %w", outPath, err)
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, rc); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}

	return nil
}
