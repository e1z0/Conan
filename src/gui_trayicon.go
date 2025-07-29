package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/mappu/miqt/qt"
)

// var myIcon *fyne.StaticResource
var tray *qt.QSystemTrayIcon

const maxItems = 20 // max menu items in tray

var uiCmdChan = make(chan string)
var Stickies = make(map[string]*StickyManagerQt)
var GUIMODE bool
var qtapp *qt.QApplication

// Linux and other unsupported os implemenation under way
func globalKeysAdd() {

}

func globalKeysListen() {
	os := GetOS()
	if os == "windows" {
		win32_bindkey()
	} else if os == "darwin" {
		darwin_bindkey()
	} else {
		globalKeysAdd()
	}
}

// showLogin creates the login window. If closed or password is wrong enough times,
// it exits. On a correct entry it calls onSuccess().
func showLogin(parent *qt.QWidget, onSuccess func()) {
	loginWin := qt.NewQDialog(parent)
	loginWin.SetWindowTitle("Enter Password")
	loginWin.SetModal(true)
	loginWin.SetWindowFlags(qt.Dialog | qt.CustomizeWindowHint | qt.WindowTitleHint | qt.WindowCloseButtonHint)
	icon := qt.NewQIcon4(":/Icon.png")
	loginWin.SetWindowIcon(icon)

	// Prevent window from closing with the X button: exit instead
	loginWin.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		os.Exit(1)
	})

	mainLayout := qt.NewQVBoxLayout(loginWin.QWidget)

	label := qt.NewQLabel5("Please enter the password to continue:", nil)
	pwEntry := qt.NewQLineEdit(nil)
	pwEntry.SetEchoMode(qt.QLineEdit__Password)
	pwEntry.SetPlaceholderText("Password")
	pwEntry.SetMinimumWidth(180)

	loginBtn := qt.NewQPushButton3("Login")

	// Try login logic
	tryLogin := func() {
		_, err := LoadEncryptedINI(env.settingsFile, pwEntry.Text())
		if err == nil {
			settings.DecryptPassword = pwEntry.Text()
			loginWin.Accept()
			onSuccess()
		} else {
			qt.QMessageBox_Warning(loginWin.QWidget, "Error", "Incorrect password")
			pwEntry.SetText("")
			pwEntry.SetFocus()
		}
	}

	// On Enter pressed
	pwEntry.OnReturnPressed(tryLogin)
	// On button clicked
	loginBtn.OnClicked(tryLogin)

	mainLayout.AddWidget(label.QWidget)
	mainLayout.AddWidget(pwEntry.QWidget)
	mainLayout.AddWidget(loginBtn.QWidget)

	loginWin.SetLayout(mainLayout.QLayout)
	loginWin.Resize(320, 140)
	//loginWin.SetFixedSize(320, 140)

	// Center on parent or screen
	pwEntry.SetFocus()
	loginWin.Exec() // Modal: waits for Accept or Reject
}

// creates basic objects and checks if the password is set
func trayIcon() {
	printThreadID("trayicon")
	qtapp = qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetQuitLockEnabled(true)
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)

	qtapp.OnLastWindowClosed(func() {
		log.Printf("last window is closed\n")
	})

	if runtime.GOOS == "windows" {
		qt.QApplication_SetStyleWithStyle("windowsvista")
	}
	if runtime.GOOS == "darwin" {
		qt.QApplication_SetStyleWithStyle("macintosh")
	}

	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if settings file is encrypted %s\n", err)
		return
	}

	GUIMODE = true

	if encrypted {
		log.Printf("Settings file is encrypted!")
		// we need to create dummy tray icon, because if we do not make it on runtime, we can't make it later
		DummyTrayIcon()
		//// 1) Kick off the login flow:
		showLogin(nil, func() {
			//	// 2) Once the correct password (“1234”) is entered:
			decryptSettings()
			trayIconLoad()
		})
	} else {
		trayIconLoad()
	}

	for _, item := range ymlfiles {
		fname := trimYML(filepath.Base(item))
		if fname != "" {
			gst := findGist(filepath.Base(item))
			notesdir := filepath.Join(env.configDir, fname+"-notes")
			//SpawnStickyWindows(a, filepath.Join(env.configDir, fname+"-notes"), gst)
			service := &NoteService{NotesDir: notesdir, HistoryDir: ".history", Gist: gst}
			sm := NewStickyManagerQt(service, nil)
			// store in registry by folder name
			key := filepath.Base(notesdir)
			log.Printf("key is: %s\n", key)
			Stickies[key] = sm
			// initial spawn
			sm.Refresh()
			log.Printf("Stickies %v\n", Stickies)
		}
	}

	IgnoreSignum()
	qt.QApplication_Exec()
}

// initial function of this file
func trayIconLoad() {
	log.Printf("Continue loading tray icon...\n")

	updateTrayMenu()
	//showFuzzySearchWindow(true)

	//mainWin = qt.NewQMainWindow(nil)

	// Handle tray commands
	go func() {
		for cmd := range uiCmdChan {
			switch cmd {
			case "show":
				CallOnQtMain(func() {
					showFuzzySearchWindow()

				})
			case "hide":
				CallOnQtMain(func() {
					searchWindow.Hide()
				})
			}
		}
	}()
	go globalKeysListen()

	if welcome {
		showWelcomeWindow()
	}

}

func DummyTrayIcon() {
	if tray != nil {
		tray.Delete()
	}
	tray = qt.NewQSystemTrayIcon()
	pixmap := qt.NewQPixmap()
	pixmap.Load(":/Icon.png")
	icon := qt.NewQIcon2(pixmap)
	tray.SetIcon(icon)
	tray.SetVisible(true)

	// -- Menu for tray icon --
	menu := qt.NewQMenu(nil)

	quitAction := menu.AddAction("Quit")
	quitAction.OnTriggered(func() {
		qt.QCoreApplication_Exit()
		os.Exit(0)
	})

	tray.SetContextMenu(menu)
}

func updateTrayMenu() {
	if tray != nil {
		tray.Delete()
	}
	tray = qt.NewQSystemTrayIcon()
	pixmap := qt.NewQPixmap()
	pixmap.Load(":/Icon.png")
	icon := qt.NewQIcon2(pixmap)
	tray.SetIcon(icon) // Use a real path or embed .png with rcc
	tray.SetVisible(true)

	// -- Menu for tray icon --
	menu := qt.NewQMenu(nil)

	showAction := menu.AddAction("Show")
	showAction.SetVisible(true)
	showAction.OnTriggered(func() {
		showFuzzySearchWindow()
	})

	srvTableItem := menu.AddAction("Servers table")
	srvTableItem.OnTriggered(func() {
		showServerTable()
	})

	for _, item := range ymlfiles {
		fname := trimYML(filepath.Base(item))
		if fname != "" {
			gst := findGist(filepath.Base(item))
			menu.AddAction("-> " + fname + " notes").OnTriggered(func() {
				ShowNotesWindowQt(nil, filepath.Join(env.configDir, fname+"-notes"), gst)
			})
		}
	}

	optionsMenu := qt.NewQMenu(nil)
	optionsMenu.SetTitle("Options")

	settingsItem := optionsMenu.AddAction("Settings")
	settingsItem.OnTriggered(func() {
		showSettingsWindow(nil, env.settingsFile)
		//showSettingsWindow(a, w, env.settingsFile)
	})

	welcomeDlgItem := optionsMenu.AddAction("Welcome dialog")
	welcomeDlgItem.OnTriggered(func() {
		showWelcomeWindow()
	})

	configLocItem := optionsMenu.AddAction("Config location")
	configLocItem.OnTriggered(func() {
		openFileOrDir(env.configDir)
	})

	logFileItem := optionsMenu.AddAction("Logfile")
	logFileItem.OnTriggered(func() {
		openFileOrDir(env.appDebugLog)
	})

	restartItem := optionsMenu.AddAction("Restart app")
	restartItem.OnTriggered(func() {
		doRestart()
	})

	updateTrayItem := optionsMenu.AddAction("Update traymenu")
	updateTrayItem.OnTriggered(func() {
		updateTrayMenu()
	})

	aboutItem := optionsMenu.AddAction("About...")
	aboutItem.OnTriggered(func() {
		showAboutQt(nil)
	})

	menu.AddMenu(optionsMenu)

	menu.AddSeparator()

	// servers menu here
	createServerMenus(menu, servers)

	//		trayMenu.Items = append(trayMenu.Items, createServerMenuItems()...)

	menu.AddSeparator()

	quitAction := menu.AddAction("Quit")
	quitAction.OnTriggered(func() {
		qt.QCoreApplication_Exit()
		os.Exit(0)
	})

	tray.SetContextMenu(menu)
}

// Returns a slice of *qt.QMenu representing your server group structure
func createServerMenus(parentMenu *qt.QMenu, servers []Server) {
	// 1. Group by config file basename
	fileGroups := make(map[string][]Server)
	for _, srv := range servers {
		base := strings.TrimSuffix(filepath.Base(srv.SourceName), filepath.Ext(srv.SourceName))
		fileGroups[base] = append(fileGroups[base], srv)
	}

	// 2. Sorted file names
	fileNames := make([]string, 0, len(fileGroups))
	for fn := range fileGroups {
		fileNames = append(fileNames, fn)
	}
	sort.Strings(fileNames)

	for _, fn := range fileNames {
		srvGroup := fileGroups[fn]
		// Group by tag
		tags := make(map[string][]Server)
		for _, srv := range srvGroup {
			if strings.TrimSpace(srv.Tags) != "" {
				for _, t := range strings.Split(srv.Tags, ",") {
					trimmed := strings.TrimSpace(t)
					tags[trimmed] = append(tags[trimmed], srv)
				}
			} else {
				tags["Untagged"] = append(tags["Untagged"], srv)
			}
		}

		// Sorted tag names
		tagNames := make([]string, 0, len(tags))
		for t := range tags {
			tagNames = append(tagNames, t)
		}
		sort.Strings(tagNames)

		// Build tag submenus (even if only "Untagged")
		fileMenu := qt.NewQMenu(nil)
		fileMenu.SetTitle(fn)
		for _, t := range tagNames {
			group := tags[t]
			tagMenu := qt.NewQMenu(nil)
			tagMenu.SetTitle(t)
			buildSplitMenu(tagMenu, group)
			fileMenu.AddMenu(tagMenu)
		}
		parentMenu.AddMenu(fileMenu)
	}
}

// Recursively build menu or submenus if needed
func buildSplitMenu(menu *qt.QMenu, srvList []Server) {
	sort.Slice(srvList, func(i, j int) bool { return srvList[i].Host < srvList[j].Host })
	if len(srvList) <= maxItems {
		buildLeafItems(menu, srvList)
		return
	}
	// Bucket by first letter
	buckets := make(map[string][]Server)
	for _, s := range srvList {
		let := "?"
		if r := []rune(s.Host); len(r) > 0 {
			let = strings.ToUpper(string(r[0]))
		}
		buckets[let] = append(buckets[let], s)
	}
	letters := make([]string, 0, len(buckets))
	for L := range buckets {
		letters = append(letters, L)
	}
	sort.Strings(letters)
	for _, L := range letters {
		group := buckets[L]
		letterMenu := qt.NewQMenu(nil)
		letterMenu.SetTitle(L)
		if len(group) <= maxItems {
			buildLeafItems(letterMenu, group)
		} else {
			chunks := chunkServers(group, maxItems)
			for _, chunk := range chunks {
				first, last := chunk[0].Host, chunk[len(chunk)-1].Host
				label := first + " … " + last
				chunkMenu := qt.NewQMenu(nil)
				chunkMenu.SetTitle(label)
				buildLeafItems(chunkMenu, chunk)
				letterMenu.AddMenu(chunkMenu)
			}
		}
		menu.AddMenu(letterMenu)
	}
}

// Add host items as actions
func buildLeafItems(menu *qt.QMenu, list []Server) {
	for _, s := range list {
		srvCopy := s // closure safety
		act := menu.AddAction(s.Host)
		act.OnTriggered(func() { ClientConnect(srvCopy) })
	}
}

// Helper: Chunk servers into max N per group
func chunkServers(sv []Server, size int) [][]Server {
	var chunks [][]Server
	for i := 0; i < len(sv); i += size {
		end := i + size
		if end > len(sv) {
			end = len(sv)
		}
		chunks = append(chunks, sv[i:end])
	}
	return chunks
}

func openFileOrDir(file string) {
	log.Printf("Opening external: %s\n", file)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", file)
	case "linux":
		cmd = exec.Command("xdg-open", file)
	case "windows":
		cmd = exec.Command("explorer", file)
	default:
		return
	}
	_ = cmd.Start()
}
