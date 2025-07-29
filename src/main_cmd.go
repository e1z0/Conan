package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	// UI/Mode flags
	trayFlag bool
	tuiFlag  bool
	// DB/Gist flags
	dbFlag   string
	pushFlag bool
	pullFlag bool
	// Import flags (delegated to importCmd)
	// Action flags
	mkeyFlag     bool
	chgKey       string
	testFlag     bool
	dirtyFlag    bool
	dbsFlag      bool
	showAtStart  bool
	globalHotkey int
)

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: "A Connection manager application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Mandatory startup
		InitializeEnvironment()

		// Adjust tray default on unsupported OS only if user didn't override
		if !cmd.PersistentFlags().Changed("tray") {
			//log.Printf("tray: %s\n", trayFlag)
			//log.Printf("Running macos bundle: %s Running windows exe %s\n", isRunningInAppBundle(), isWinExecutable())
			if isRunningInAppBundle() || isWinExecutable() {
				trayFlag = true
			} else {
				if len(os.Args) == 1 {
					tuiFlag = true
				}
			}
		}

		// load additional data if some flags are require that
		if trayFlag || tuiFlag || testFlag || pushFlag || pullFlag {
			initApp() // initializes program configuration and loads all server files
		}

		// require encryption check {
		if pushFlag || pullFlag {
			err := tuiCheckProtection()
			if err != nil {
				log.Printf("Error: %s\n", err)
				os.Exit(1)
			}
		}

		// Handle one-off action flags
		if err := handleActionFlags(); err != nil {
			return
		}
		return
	},
	Run: func(cmd *cobra.Command, args []string) {
		if trayFlag {
			// launch tray icon
			runGUI()
			return
		}
		if tuiFlag {
			// run TUI mode
			runTUI()
			return
		}
		// default GUI or other logic
		//runTUI()
	},
}

// initApp performs mandatory startup routines
func initApp() {
	firstStart()
	loadSettings("")

	if dbFlag != "" {
		if err, _ := checkServYmlFiles(dbFlag); err != nil {
			log.Printf("Servers DB Not found at: %s: %w\n", dbFlag, err)
			os.Exit(1)
		}
	} else {
		findServerFiles()
	}
	fetchServersFromFiles()
}

// handleActionFlags runs when standalone flags are provided at the root level
func handleActionFlags() error {
	if mkeyFlag {
		pass, err := generatePassword(32)
		if err != nil {
			return err
		}
		fmt.Printf("\nGenerated encryption key: %s\n\n", pass)
		os.Exit(0)
	}

	if *&dbsFlag {
		findServerFiles()
		os.Exit(0)
	}

	if pullFlag {
		err := DownloadGists()
		if err != nil {
			log.Printf("Error pulling from gist: %s\n", err)
		}
	}

	if pushFlag {
		err := UploadGists()
		if err != nil {
			log.Printf("Error pushing to gist: %s\n", err)
		}

	}

	if chgKey != "" {
		if err := changeEncryptionKey(dbFlag, chgKey); err != nil {
			return err
		}
		fmt.Println("Encryption key changed successfully!")
		os.Exit(0)
	}

	if dirtyFlag {
		fmt.Println("Dirty GUI loading... Not implemented yet.")
		os.Exit(0)
	}
	if testFlag {
		// debug output
		for _, srv := range servers {
			log.Printf("DEBUG: %s@%s (%s)\n", srv.User, srv.IP, srv.Host)
		}
		os.Exit(0)
	}
	return nil
}

func init() {
	// Compute default for tray: true when .exe on Windows or .app on macOS
	//trayFlag = (!isRunningInAppBundle() || !isWinExecutable())

	rootCmd.PersistentFlags().BoolVar(&trayFlag, "tray", false, "Run app in system tray (default on Windows/Mac .app)")
	rootCmd.PersistentFlags().BoolVar(&showAtStart, "show", false, "Show fuzzy search window immediately at startup")
	rootCmd.PersistentFlags().IntVar(&globalHotkey, "hotkey", 1, "Global hotkey enabled?")
	rootCmd.PersistentFlags().BoolVarP(&tuiFlag, "tui", "t", false, "Run in TUI mode")
	rootCmd.PersistentFlags().StringVar(&dbFlag, "db", "", "Servers database to use")
	rootCmd.PersistentFlags().StringVar(&chgKey, "chgkey", "", "Change encryption key for DB")
	rootCmd.PersistentFlags().BoolVar(&pushFlag, "push", false, "Push server list changes to GitHub Gist")
	rootCmd.PersistentFlags().BoolVar(&pullFlag, "pull", false, "Pull server list changes from GitHub Gist")
	rootCmd.PersistentFlags().BoolVar(&mkeyFlag, "mkey", false, "Generate a random encryption key")
	rootCmd.PersistentFlags().BoolVar(&dbsFlag, "dbs", false, "Show available server database files")
	rootCmd.PersistentFlags().BoolVar(&dirtyFlag, "q", false, "Quick and dirty server selection (old one)")
	rootCmd.PersistentFlags().BoolVar(&testFlag, "test", false, "Test some functions")

	// Global flags can go here...

	// Add import subcommand
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(importSettingsCmd)
}
