package main

/*
(c) 2025 e1z0, sshexperiment - Conan
*/

import (
	"bufio"
	"fmt"
	"path/filepath"
	"syscall"

	"log"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"

	"strings"

	"github.com/rivo/tview"
	"gopkg.in/ini.v1"
)

var appbase = tview.NewApplication()
var grid = tview.NewFlex()
var table = tview.NewTable().SetSelectable(true, false)
var searchBox = tview.NewInputField()
var searchMode = false
var theme map[string]tcell.Color

func tuiCheckProtection() error {
	encrypted, err := IsEncryptedINI(env.settingsFile)
	if err != nil {
		log.Printf("Unable to determine if the program settings is encrypted: %s\n", err)
		return err
	}
	if encrypted {
		log.Printf("Program settings is encrypted!\n\n")
		for {
			fmt.Printf("Please enter password: ")
			bytePwd, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // move to next line after user hits Enter
			if err != nil {
				log.Fatalf("Failed to read password: %v", err)
				continue
			}
			password := string(bytePwd)
			_, err = LoadEncryptedINI(env.settingsFile, password)
			if err == nil {
				settings.DecryptPassword = password
				break
			}
			fmt.Println("❌ Incorrect—please try again.")
		}
		log.Printf("Password accepted!\n")
		loadSettings(settings.DecryptPassword)

	}
	return nil
}

func initTUI() {
	loadTheme()
	err := tuiCheckProtection()
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	initSearchBox()
	updateTable()
	applyTheme()

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if searchMode {
			return event
		}
		if event.Key() == tcell.KeyEnter && event.Modifiers() == tcell.ModAlt {
			row, _ := table.GetSelection()
			if row >= 1 {
				if row <= 0 || row > len(filteredServers) {

				} else {
					showContextMenu(filteredServers[row-1])
				}
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row >= 1 {
				if row <= 0 || row > len(filteredServers) {
				} else {
					ClientConnect(filteredServers[row-1])
				}
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appbase.Stop()
			case '/':
				searchMode = true
				appbase.SetFocus(searchBox)
			case 'h':
				showHelp()
			case 'i':
				insertServer()
			case 'e':
				row, _ := table.GetSelection()
				editServer(row)
			case 'r':
				updateTable()
			case 'l':
				row, _ := table.GetSelection()
				showContextMenu(filteredServers[row-1])
			case 'd':
				row, _ := table.GetSelection()
				deleteServer(row)
			}
		}
		return event
	})

	grid.SetDirection(tview.FlexRow).
		AddItem(searchBox, 1, 0, false).
		AddItem(table, 0, 1, true)

	if err := appbase.SetRoot(grid, true).Run(); err != nil {
		fmt.Printf("Got small error: %s\n")
		//panic(err)
	}
}

func loadTheme() {
	cfg, err := ini.Load(filepath.Join(env.configDir, "themes", "default.ini"))
	if err != nil {
		log.Println("Failed to load theme file:", err)
		os.Exit(1)
	}

	theme = map[string]tcell.Color{
		"background_color":               parseColor(cfg.Section("theme").Key("background_color").String()),
		"contrast_background_color":      parseColor(cfg.Section("theme").Key("contrast_background_color").String()),
		"more_contrast_background_color": parseColor(cfg.Section("theme").Key("more_contrast_background_color").String()),
		"border_color":                   parseColor(cfg.Section("theme").Key("border_color").String()),
		"title_color":                    parseColor(cfg.Section("theme").Key("title_color").String()),
		"graphics_color":                 parseColor(cfg.Section("theme").Key("graphics_color").String()),
		"primary_text_color":             parseColor(cfg.Section("theme").Key("primary_text_color").String()),
		"secondary_text_color":           parseColor(cfg.Section("theme").Key("secondary_text_color").String()),
		"tertiary_text_color":            parseColor(cfg.Section("theme").Key("tertiary_text_color").String()),

		// Table styling
		"header_background":   parseColor(cfg.Section("table").Key("header_background").String()),
		"header_text":         parseColor(cfg.Section("table").Key("header_text").String()),
		"row_even_background": parseColor(cfg.Section("table").Key("row_even_background").String()),
		"row_odd_background":  parseColor(cfg.Section("table").Key("row_odd_background").String()),
		"hostname_color":      parseColor(cfg.Section("table").Key("hostname_color").String()),
		"ip_color":            parseColor(cfg.Section("table").Key("ip_color").String()),
		"description_color":   parseColor(cfg.Section("table").Key("description_color").String()),
		"type_color":          parseColor(cfg.Section("table").Key("type_color").String()),

		// Buttons
		"button_background": parseColor(cfg.Section("buttons").Key("background").String()),
		"button_text":       parseColor(cfg.Section("buttons").Key("text").String()),

		// Searchbox
		"searchbox_label_color": parseColor(cfg.Section("searchbox").Key("label_color").String()),
		"searchbox_background":  parseColor(cfg.Section("searchbox").Key("background").String()),
	}
}

func parseColor(color string) tcell.Color {
	switch color {
	case "black":
		return tcell.ColorBlack
	case "gray":
		return tcell.ColorGray
	case "darkgray":
		return tcell.ColorDarkGray
	case "white":
		return tcell.ColorWhite
	case "green":
		return tcell.ColorGreen
	case "blue":
		return tcell.ColorBlue
	case "yellow":
		return tcell.ColorYellow
	case "lightcyan":
		return tcell.ColorLightCyan
	case "darkslategray":
		return tcell.ColorDarkSlateGray
	default:
		return tcell.ColorDefault
	}
}

func applyTheme() {
	tview.Styles.PrimitiveBackgroundColor = theme["background_color"]
	tview.Styles.ContrastBackgroundColor = theme["contrast_background_color"]
	tview.Styles.MoreContrastBackgroundColor = theme["more_contrast_background_color"]
	tview.Styles.BorderColor = theme["border_color"]
	tview.Styles.TitleColor = theme["title_color"]
	tview.Styles.GraphicsColor = theme["graphics_color"]
	tview.Styles.PrimaryTextColor = theme["primary_text_color"]
	tview.Styles.SecondaryTextColor = theme["secondary_text_color"]
	tview.Styles.TertiaryTextColor = theme["tertiary_text_color"]

	table.SetBackgroundColor(theme["background_color"])
	table.SetBorder(true).
		SetTitle(" [::b]🌌 Server List 🌌[::-] ").
		SetTitleAlign(tview.AlignCenter)

	searchBox.SetLabelColor(theme["searchbox_label_color"]).
		SetFieldBackgroundColor(theme["searchbox_background"])
}

func showHelp() {
	helpText := "[::b]Help Menu:[::-]\n\n"
	helpText += "[green]q[::-] - Quit\n"
	helpText += "[cyan]/[::-] - Enter search mode\n"
	helpText += "[yellow]h[::-] - Show this help menu\n"
	helpText += "[magenta]i[::-] - Insert a new server\n"
	helpText += "[red]d[::-] - Delete selected server\n"
	helpText += "[blue]Arrow Keys[::-] - Navigate server list\n"
	helpText += "[white]Enter[::-] - Connect to selected server"

	dialog := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"OK"}).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			returnToMainWindow()
		})

	appbase.SetRoot(dialog, true)
}

func fuzzySearch(query string) {
	query = strings.ToLower(query)
	if query == "" {
		filteredServers = servers // Reset to show all servers
	} else {
		filteredServers = nil
		for _, s := range servers {
			if fuzzy.Match(query, strings.ToLower(s.Host)) || fuzzy.Match(query, strings.ToLower(s.IP)) || fuzzy.Match(query, strings.ToLower(s.Description)) {
				filteredServers = append(filteredServers, s)
			}
		}
	}
	updateTable()
}

func initSearchBox() {
	searchBox.SetLabel("🔍 Search: ").
		SetFieldBackgroundColor(tcell.ColorGray).
		SetChangedFunc(func(text string) {
			fuzzySearch(text)
		}).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				searchMode = false
				appbase.SetFocus(table)
			}
			if key == tcell.KeyEnter {
				searchMode = false
				appbase.SetFocus(table)
			}
		})
}

func askYesNo(question string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(question + " (yes/no): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "yes" || input == "no" {
			return input
		}

		fmt.Println("Invalid input. Please enter 'yes' or 'no'.")
	}
}

// Function to display server info
func showServerInfo(srv Server) {
	info := fmt.Sprintf(
		"Hostname: %s\nIP: %s\nDescription: %s\nType: %s",
		srv.Host, srv.IP, srv.Description, srv.Type,
	)

	dialog := tview.NewModal().
		SetText(info).
		AddButtons([]string{"OK"}).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack).
		SetButtonBackgroundColor(tcell.ColorBlue).
		SetButtonTextColor(tcell.ColorWhite).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			returnToMainWindow()
		})

	appbase.SetRoot(dialog, true)
}

func showContextMenu(srv Server) {
	pages := tview.NewPages()
	pages.AddPage("main", grid, true, true)
	ContextMenu(appbase, pages, "Context Menu", []string{"Open", "Info"}, func(index int, option string) {
		// Handle menu selection here
		switch option {
		case "Open":
			jumpserver(srv)
		case "Info":
			showServerInfo(srv)
		}
	})
	appbase.SetRoot(pages, true)
}

// ShowContextMenu displays a list of options in a modal-style context menu
func ContextMenu(app *tview.Application, pages *tview.Pages, title string, options []string, onSelect func(index int, option string)) {
	menu := tview.NewList()
	for i, opt := range options {
		i := i
		menu.AddItem(opt, "", rune('a'+i), func() {
			onSelect(i, opt)
			pages.RemovePage("context")
		})
	}
	menu.AddItem("Cancel", "Close menu", 'q', func() {
		pages.RemovePage("context")
	})

	menu.SetBorder(true).SetTitle(title)

	// Show the menu on a centered flex
	pages.AddPage("context",
		tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(menu, 30, 1, true),
		true, true,
	)

	appbase.SetFocus(menu)
}

// return to the main window
func returnToMainWindow() {
	appbase.SetRoot(grid, true)
}

func ShowMessageBox(title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			returnToMainWindow()
		})
		//SetFocus(0)

	modal.SetTitle(title).SetBorder(true)
	appbase.SetRoot(modal, false).SetFocus(modal)
}

func insertServer() {
	form := tview.NewForm()
	form.AddDropDown("File", baseNames(ymlfiles), 0, nil).
		AddInputField("Hostname", "", 20, nil, nil).
		AddInputField("IP Address", "", 15, nil, nil).
		AddInputField("Port", "", 15, nil, nil).
		AddInputField("Username", "", 20, nil, nil).
		AddInputField("Password", "", 20, nil, nil).
		AddInputField("Description", "", 30, nil, nil).
		AddDropDown("Type", ServerTypes, 0, nil).
		AddButton("Save", func() {
			hostname := form.GetFormItemByLabel("Hostname").(*tview.InputField).GetText()
			username := form.GetFormItemByLabel("Username").(*tview.InputField).GetText()
			passw := form.GetFormItemByLabel("Password").(*tview.InputField).GetText()
			ip := form.GetFormItemByLabel("IP Address").(*tview.InputField).GetText()
			port := form.GetFormItemByLabel("Port").(*tview.InputField).GetText()
			desc := form.GetFormItemByLabel("Description").(*tview.InputField).GetText()
			typeIndex, _ := form.GetFormItemByLabel("Type").(*tview.DropDown).GetCurrentOption()
			serverType := ServerTypes[typeIndex]
			selectFileIndex, _ := form.GetFormItemByLabel("File").(*tview.DropDown).GetCurrentOption()
			selectedFileBaseName := baseNames(ymlfiles)[selectFileIndex]

			selectedFileFullPath, err := fullPathFor(selectedFileBaseName, ymlfiles)

			if err != nil {
				ShowMessageBox("Error", "Failed to get full path for selected file: "+err.Error())
				return
			}

			if hostname != "" && ip != "" {
				srv := Server{ID: uuid.NewString(), SourceName: selectedFileBaseName, SourcePath: selectedFileFullPath, Host: hostname, User: username, Password: "", IP: ip, Port: port, Description: desc, Type: serverType}
				srv.Password = srv.EncryptPassword(passw)
				servers = append(servers, srv)

				pushServersToFile()
				fetchServersFromFiles()
				updateTable()
			}
			returnToMainWindow()
		}).
		AddButton("Cancel", func() { returnToMainWindow() })

	form.SetBorder(true).SetTitle(" Add Server ").SetTitleAlign(tview.AlignCenter)
	appbase.SetRoot(form, true)
}

func deleteServer(row int) {
	if row <= 0 || row > len(filteredServers) {
		return
	}
	srv := filteredServers[row-1]

	// Create a styled modal confirmation dialog
	confirmation := tview.NewModal().
		SetText("Are you sure you want to delete " + srv.Host + "?").
		SetTextColor(tcell.ColorWhite).            // Light text for contrast
		SetBackgroundColor(tcell.Color16).         // Darker background to match gh-dash
		SetButtonBackgroundColor(tcell.ColorBlue). // Match button colors
		SetButtonTextColor(tcell.ColorWhite).      // Button text in white
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				// find and remove from global slice
				for i, s := range servers {
					if s.ID == srv.ID {
						// delete element i
						servers = append(servers[:i], servers[i+1:]...)
						break
					}
				}

				pushServersToFile()
				fetchServersFromFiles()
				updateTable()
			}
			returnToMainWindow()
		})

	appbase.SetRoot(confirmation, true)
}

func editServer(row int) {
	//servers := fetchServers()
	if row <= 0 || row > len(filteredServers) {
		return
	}
	srv := filteredServers[row-1]
	// decrypt the password
	pass := srv.DecryptPassword()

	// get index of the selected type
	idx := indexOf(ServerTypes, srv.Type)
	if idx == -1 {
		idx = 0 // fallback to default
	}
	srvIdx := -1
	for i, s := range servers {
		if s.ID == srv.ID {
			srvIdx = i
			break
		}
	}
	if srvIdx == -1 {
		return
	}

	form := tview.NewForm()
	form.AddInputField("File", srv.SourceName, 20, nil, nil).
		AddInputField("Hostname", srv.Host, 20, nil, nil).
		AddInputField("IP Address", srv.IP, 15, nil, nil).
		AddInputField("Username", srv.User, 15, nil, nil).
		AddInputField("Password", pass, 15, nil, nil).
		AddInputField("Port", srv.Port, 15, nil, nil).
		AddInputField("Description", srv.Description, 60, nil, nil).
		AddDropDown("Type", ServerTypes, idx, nil). // declared in servers_yml.go
		AddButton("Save", func() {
			form.GetFormItemByLabel("File").(*tview.InputField).SetDisabled(true)
			hostname := form.GetFormItemByLabel("Hostname").(*tview.InputField).GetText()
			username := form.GetFormItemByLabel("Username").(*tview.InputField).GetText()
			passw := form.GetFormItemByLabel("Password").(*tview.InputField).GetText()
			ip := form.GetFormItemByLabel("IP Address").(*tview.InputField).GetText()
			port := form.GetFormItemByLabel("Port").(*tview.InputField).GetText()
			desc := form.GetFormItemByLabel("Description").(*tview.InputField).GetText()
			typeIndex, _ := form.GetFormItemByLabel("Type").(*tview.DropDown).GetCurrentOption()
			serverType := ServerTypes[typeIndex]
			srv.Host = hostname
			srv.User = username
			srv.Password = srv.EncryptPassword(passw)
			srv.IP = ip
			srv.Port = port
			srv.Description = desc
			srv.Type = serverType
			servers[srvIdx] = srv

			pushServersToFile()
			fetchServersFromFiles()
			updateTable()
			returnToMainWindow()
		}).
		AddButton("Cancel", func() { returnToMainWindow() })

	form.SetBorder(true).SetTitle(" Edit Server ").SetTitleAlign(tview.AlignCenter)
	appbase.SetRoot(form, true)
}

func updateTable() {
	table.Clear()
	headers := []string{"Hostname", "IP Address", "Description", "Type"}
	colors := []tcell.Color{theme["hostname_color"], theme["ip_color"], theme["description_color"], theme["type_color"]}

	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(theme["header_text"]).
			SetBackgroundColor(theme["header_background"]).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	}

	for i, srv := range filteredServers {
		row := i + 1
		data := []string{srv.Host, srv.IP, srv.Description, srv.Type}
		bgColor := theme["row_odd_background"]
		// different colors every second row
		if i%2 == 0 {
			bgColor = theme["row_even_background"]
		}
		for col, text := range data {
			table.SetCell(row, col, tview.NewTableCell(text).
				SetTextColor(colors[col]).
				SetBackgroundColor(bgColor).
				SetAlign(tview.AlignLeft))
		}
	}
}
