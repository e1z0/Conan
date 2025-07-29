package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	importFile    string
	importIpCol   string
	importPortCol string
	importHostCol string
	importUserCol string
	importPassCol string
	importDescCol string
	importTypeCol string
	importColSep  string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import servers from CSV",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFile == "" {
			return fmt.Errorf("--file is required for import")
		}
		// call import logic
		importServersFromCSV(
			importFile,
			importIpCol,
			importUserCol,
			importPassCol,
			importPortCol,
			importHostCol,
			importDescCol,
			importTypeCol,
			importColSep,
		)
		return nil
	},
}

var importSettingsCmd = &cobra.Command{
	Use:   "importsettings",
	Short: "Import settings from .cnn file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFile == "" {
			return fmt.Errorf("--file is required for import")
		}
		// call import logic
		importSettingsFile(
			importFile,
		)
		return nil
	},
}

var exportSettingsCmd = &cobra.Command{
	Use:   "exportsettings",
	Short: "Export settings to .cnn file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFile == "" {
			return fmt.Errorf("--file is required for import")
		}
		// call import logic
		exportSettingsFile(
			importFile,
		)
		return nil
	},
}

func exportSettingsFile(path string) {
	if path == "" {
		log.Printf("No output file is specified by --file parameter\n")
		return
	}
	var password string
	for {
		fmt.Printf("Please enter password (ctrl+c to quit): ")
		bytePwd, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // move to next line after user hits Enter
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
			continue
		}
		fmt.Printf("Please enter password (verify): ")
		bytePwd2, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // move to next line after user hits Enter
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
			continue
		}

		if string(bytePwd) == string(bytePwd2) {
			password = string(bytePwd)
			break
		}
		fmt.Println("❌ Passwords do not match, try again.")
	}
	if password != "" {
		log.Printf("password is set!\n")
	}
	// 2) ZIP configDir into memory
	zipData, err := compressDirToBuffer(env.configDir)
	if err != nil {
		log.Printf("Unable to compress folder: %s\n", err)
		return
	}

	// 3) Encrypt with AES-GCM / scrypt key
	sealed, err := encrypt(zipData, password)
	if err != nil {
		log.Printf("Unable to seal the archive %s\n", err)
		return
	}

	if err := os.WriteFile(path, sealed, 0o644); err != nil {
		log.Fatalf("Write failed: %v", err)
		return
	}

	log.Printf("Export is complete! file is located at: %s\n", path)
}

// import settings used in CLI mode
func importSettingsFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening cnn file: %v\n", err)
		os.Exit(1)
	}
	// Ensure the file gets closed when we're done
	defer f.Close()

	// 2. Read entire file into memory
	data, err := io.ReadAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	var Decrpassword string

	for {
		fmt.Printf("Please enter password: ")
		bytePwd, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // move to next line after user hits Enter
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
			continue
		}
		password := string(bytePwd)
		_, err = decrypt(data, password)
		if err == nil {
			Decrpassword = password
			break
		}
		fmt.Println("❌ Incorrect—please try again.")
	}

	plain, err := decrypt(data, Decrpassword)
	if err != nil {
		log.Printf("Error decrypting archive: %s\n", err)
		os.Exit(1)
	}
	// 4) Unzip into configDir
	if err := decompressZipToDir(plain, env.configDir); err != nil {
		log.Printf("Error restoring files: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Restore succeeded!\nRe-Run the application\n")
	os.Exit(0)
}

// Function to import servers from a CSV file
func importServersFromCSV(filename, ipCol, userCol, passCol, portCol, hostCol, descCol, typeCol, colSep string) {

}

func init() {
	importCmd.Flags().StringVar(&importFile, "file", "", "Path to CSV file (required)")
	importCmd.Flags().StringVar(&importIpCol, "ipcol", "ip", "Column name for IP Address")
	importCmd.Flags().StringVar(&importPortCol, "portcol", "port", "Column name for Port")
	importCmd.Flags().StringVar(&importHostCol, "hostcol", "hostname", "Column name for Hostname")
	importCmd.Flags().StringVar(&importUserCol, "usercol", "username", "Column name for Username")
	importCmd.Flags().StringVar(&importPassCol, "passcol", "password", "Column name for Password")
	importCmd.Flags().StringVar(&importDescCol, "desccol", "description", "Column name for Description")
	importCmd.Flags().StringVar(&importTypeCol, "typecol", "type", "Column name for Type")
	importCmd.Flags().StringVar(&importColSep, "colsep", ";", "Column separator (, ; or |)")

	importSettingsCmd.Flags().StringVar(&importFile, "file", "", "Import .cnn file (required)")
	exportSettingsCmd.Flags().StringVar(&importFile, "file", "", "Export .cnn file (required)")

	//rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportSettingsCmd)
}
