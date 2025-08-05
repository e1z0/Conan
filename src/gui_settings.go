package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/mappu/miqt/qt"
	"gopkg.in/ini.v1"
)

var encryptionPass string
var settingsWindow *qt.QDialog

// showSettingsWindow reads the INI at configPath and displays an editable settings form.
func showSettingsWindow(parent *qt.QWidget, configPath string) {
	if settingsWindow != nil {
		settingsWindow.Show()
		settingsWindow.Raise()
		settingsWindow.ActivateWindow()
		settingsWindow.SetFocus()
		return
	}

	// Read INI
	encrypted, err := IsEncryptedINI(configPath)
	if err != nil {
		QTshowWarn(parent, "Error", "Unable to determine if settings are encrypted.")
		return
	}

	var cfg *ini.File
	if encrypted {
		cfg, err = LoadEncryptedINI(configPath, settings.DecryptPassword)
		if err != nil {
			QTshowWarn(parent, "Error", "Unable to load encrypted settings.")
			return
		}
	} else {
		cfg, err = ini.Load(configPath)
		if err != nil {
			QTshowWarn(parent, "Error", err.Error())
			return
		}
	}

	// Main window
	settingsWindow = qt.NewQDialog(parent)
	settingsWindow.SetWindowTitle("Settings")
	settingsWindow.SetWindowIcon(globalIcon)
	settingsWindow.Resize(700, 600)

	settingsWindow.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		settingsWindow.Hide()
	})
	mainLayout := qt.NewQVBoxLayout(settingsWindow.QWidget)

	// TAB WIDGET
	tabs := qt.NewQTabWidget(settingsWindow.QWidget)

	// GENERAL TAB
	generalTab := qt.NewQWidget(settingsWindow.QWidget)
	generalLayout := qt.NewQFormLayout(generalTab)

	// Entries
	general := cfg.Section("General")

	enckeyEdit := qt.NewQLineEdit4(general.Key("enckey").String(), nil)
	sshClientCombo := qt.NewQComboBox(nil)
	for _, cli := range sshConnectionClients {
		sshClientCombo.AddItem(cli)
	}
	sshClientCombo.SetCurrentText(general.Key("ssh_client").String())

	osSuffix := GetOS()
	sshCmd := qt.NewQLineEdit4(general.Key(osSuffix+"_ssh").String(), nil)
	rdpCmd := qt.NewQLineEdit4(general.Key(osSuffix+"_rdp").String(), nil)
	winboxCmd := qt.NewQLineEdit4(general.Key(osSuffix+"_winbox").String(), nil)

	generalLayout.AddRow3("Global Encryption Key", enckeyEdit.QWidget)
	generalLayout.AddRow3("SSH Client", sshClientCombo.QWidget)
	generalLayout.AddRow3("SSH Cmd", sshCmd.QWidget)
	generalLayout.AddRow3("RDP Cmd", rdpCmd.QWidget)
	generalLayout.AddRow3("WinBox Cmd", winboxCmd.QWidget)

	// IGNORE SERVERS FILE - MULTI-LIST
	rawIgnore := general.Key("ignore").String()
	ignoreOptions := []string{}
	if rawIgnore != "" {
		ignoreOptions = strings.Split(rawIgnore, ",")
	}

	// Create combo box
	ignoreCombo := qt.NewQComboBox(nil)
	ignoreCombo.SetEditable(true)
	ignoreCombo.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	ignoreCombo.SetMinimumWidth(250) // Or 300, whatever looks best

	for _, v := range ignoreOptions {
		ignoreCombo.AddItem(v)
	}
	// Optionally: set the first as current
	if len(ignoreOptions) > 0 {
		ignoreCombo.SetCurrentText(ignoreOptions[0])
	}

	// Add button to add new entry
	addIgnoreBtn := qt.NewQPushButton3("Add")
	addIgnoreBtn.OnClicked(func() {
		txt := ignoreCombo.CurrentText()
		if txt != "" && ignoreCombo.FindText(txt) == -1 {
			ignoreCombo.AddItem(txt)
		}
	})

	// Remove button for selected
	removeIgnoreBtn := qt.NewQPushButton3("Remove")
	removeIgnoreBtn.OnClicked(func() {
		idx := ignoreCombo.CurrentIndex()
		if idx >= 0 {
			ignoreCombo.RemoveItem(idx)
			if ignoreCombo.Count() > 0 {
				ignoreCombo.SetCurrentIndex(0)
			}
		}
	})

	// Horizontal layout for combo + buttons
	ignoreLabel := qt.NewQLabel3("Ignore servers file")
	// FIX to align to the same line as the combobox
	ignoreLabel.SetFixedHeight(50)

	ignoreRowLayout := qt.NewQHBoxLayout2()
	ignoreRowLayout.AddWidget(ignoreCombo.QWidget)
	ignoreRowLayout.AddWidget(addIgnoreBtn.QWidget)
	ignoreRowLayout.AddWidget(removeIgnoreBtn.QWidget)
	rowWidget := qt.NewQWidget(nil)
	rowWidget.SetLayout(ignoreRowLayout.QLayout)

	generalLayout.AddRow(ignoreLabel.QWidget, rowWidget)

	// After creating QLineEdit/QComboBox for each field
	for _, w := range []*qt.QWidget{enckeyEdit.QWidget, sshClientCombo.QWidget, sshCmd.QWidget, rdpCmd.QWidget, winboxCmd.QWidget} {
		w.SetMinimumWidth(400)
		w.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	}

	generalLayout.SetLabelAlignment(qt.AlignRight | qt.AlignVCenter)
	generalLayout.SetFormAlignment(qt.AlignLeft | qt.AlignTop)
	generalLayout.SetFieldGrowthPolicy(qt.QFormLayout__ExpandingFieldsGrow)
	generalLayout.SetHorizontalSpacing(18)
	generalLayout.SetVerticalSpacing(12)

	generalTab.SetLayout(generalLayout.QLayout)

	encryptSettingsCheckbox := qt.NewQCheckBox4("Encrypt settings", nil)
	encryptSettingsCheckbox.SetChecked(encrypted)

	encryptSettingsCheckbox.OnClickedWithChecked(func(checked bool) {
		if checked {
			pw := QTPromptPasswordVerify(settingsWindow.QWidget, "Set encryption password:")
			if pw == "" {
				encryptSettingsCheckbox.SetChecked(false)
			} else {
				encryptionPass = pw
			}
		}
	})

	generalLayout.AddRow3("Encrypt settings", encryptSettingsCheckbox.QWidget)

	// ---- SYNC SETTINGS (GISTS) ----
	sections := cfg.SectionStrings()
	gistNames := []string{}
	for _, sec := range sections {
		if strings.HasPrefix(sec, "gist ") {
			gistNames = append(gistNames, strings.TrimPrefix(sec, "gist "))
		}
	}
	selectedGist := qt.NewQComboBox(nil)
	selectedGist.AddItem("")
	for _, n := range gistNames {
		selectedGist.AddItem(n)
	}
	gistID := qt.NewQLineEdit(nil)
	gistSec := qt.NewQLineEdit(nil)
	gistEnc := qt.NewQLineEdit(nil)
	gistNotesEncryption := qt.NewQCheckBox4("Encrypt notes", nil)

	loadGist := func(name string) {
		if name == "" {
			gistID.SetText("")
			gistSec.SetText("")
			gistEnc.SetText("")
			gistNotesEncryption.SetChecked(false)
			return
		}
		sec := cfg.Section("gist " + name)
		gistID.SetText(sec.Key("gistid").String())
		gistSec.SetText(sec.Key("gistsec").String())
		gistEnc.SetText(sec.Key("enckey").String())
		gistNotesEncryption.SetChecked(sec.Key("encrypt_notes").MustBool())
	}
	selectedGist.OnCurrentIndexChanged(func(idx int) {
		loadGist(selectedGist.CurrentText())
	})

	// Add/remove buttons
	addGistBtn := qt.NewQPushButton5("Add", nil)
	removeGistBtn := qt.NewQPushButton5("Remove", nil)
	addGistBtn.OnClicked(func() {
		inputDlg := qt.NewQInputDialog(settingsWindow.QWidget)
		inputDlg.SetLabelText("Gist name:")
		if inputDlg.Exec() == int(qt.QDialog__Accepted) {
			txt := inputDlg.TextValue()
			if txt != "" {
				selectedGist.AddItem(txt)
				selectedGist.SetCurrentText(txt)
			}
		}
	})
	removeGistBtn.OnClicked(func() {
		name := selectedGist.CurrentText()
		if name == "" {
			return
		}
		cfg.DeleteSection("gist " + name)
		selectedGist.RemoveItem(selectedGist.CurrentIndex())
		selectedGist.SetCurrentIndex(0)
	})

	gistBtnLayout := qt.NewQHBoxLayout2()
	gistBtnLayout.AddWidget(addGistBtn.QWidget)
	gistBtnLayout.AddWidget(removeGistBtn.QWidget)

	gistForm := qt.NewQFormLayout(nil)
	gistForm.AddRow3("Gist ID", gistID.QWidget)
	gistForm.AddRow3("Gist Secret", gistSec.QWidget)
	gistForm.AddRow3("Gist Encrypt Key", gistEnc.QWidget)
	gistForm.AddRow3("Local notes", gistNotesEncryption.QWidget)

	syncContainer := qt.NewQVBoxLayout2()
	syncContainer.AddWidget(selectedGist.QWidget)
	syncContainer.AddLayout(gistBtnLayout.QLayout)
	syncContainer.AddLayout(gistForm.QLayout)

	syncTab := qt.NewQWidget(settingsWindow.QWidget)
	syncTab.SetLayout(syncContainer.QLayout)

	// SERVER TABLE TAB
	serversTableTab := qt.NewQWidget(settingsWindow.QWidget)
	serversLayout := qt.NewQFormLayout(serversTableTab)
	serverstable := cfg.Section("ServersTable")
	notes := cfg.Section("notes")

	disableTooltips := qt.NewQCheckBox4("Disable server table all items tooltips", nil)
	disableTooltips.SetChecked(serverstable.Key("disabletooltips").MustBool())
	disableRowTooltips := qt.NewQCheckBox4("Disable server table row item tooltips", nil)
	disableRowTooltips.SetChecked(serverstable.Key("disablerowtooltips").MustBool())
	hostCol := qt.NewQLineEdit4(serverstable.Key("HostColumn").String(), nil)
	typeCol := qt.NewQLineEdit4(serverstable.Key("TypeColumn").String(), nil)
	ipCol := qt.NewQLineEdit4(serverstable.Key("IpColumn").String(), nil)
	userCol := qt.NewQLineEdit4(serverstable.Key("UserColumn").String(), nil)
	descCol := qt.NewQLineEdit4(serverstable.Key("DescriptionColumn").String(), nil)
	tagsCol := qt.NewQLineEdit4(serverstable.Key("TagsColumn").String(), nil)
	sourceCol := qt.NewQLineEdit4(serverstable.Key("SourceColumn").String(), nil)
	availCol := qt.NewQLineEdit4(serverstable.Key("AvailabilityColumn").String(), nil)

	serversLayout.AddRow3("Disable all tooltips", disableTooltips.QWidget)
	serversLayout.AddRow3("Disable row tooltips", disableRowTooltips.QWidget)
	serversLayout.AddRow3("Host Column Size", hostCol.QWidget)
	serversLayout.AddRow3("Type Column Size", typeCol.QWidget)
	serversLayout.AddRow3("IP Column Size", ipCol.QWidget)
	serversLayout.AddRow3("User Column Size", userCol.QWidget)
	serversLayout.AddRow3("Description Column Size", descCol.QWidget)
	serversLayout.AddRow3("Tags Column Size", tagsCol.QWidget)
	serversLayout.AddRow3("Source Column Size", sourceCol.QWidget)
	serversLayout.AddRow3("Availability Column Size", availCol.QWidget)
	serversTableTab.SetLayout(serversLayout.QLayout)

	// EXPERT TAB
	expertTab := qt.NewQWidget(settingsWindow.QWidget)
	expertLayout := qt.NewQFormLayout(expertTab)
	syncCheckbox := qt.NewQCheckBox4("Enable Gist sync", nil)
	syncCheckbox.SetChecked(general.Key("sync").MustBool())
	notesOnTopCheckbox := qt.NewQCheckBox4("Enable always on top", nil)
	notesOnTopCheckbox.SetChecked(notes.Key("alwaysontop").MustBool())
	defaultSSHKey := qt.NewQLineEdit4(general.Key("defaultsshkey").String(), nil)
	expertLayout.AddRow3("Gist Sync", syncCheckbox.QWidget)
	expertLayout.AddRow3("Default SSH key", defaultSSHKey.QWidget)
	expertLayout.AddRow3("Notes stickies", notesOnTopCheckbox.QWidget)
	expertTab.SetLayout(expertLayout.QLayout)

	// Add tabs
	tabs.AddTab(generalTab, "General")
	tabs.AddTab(syncTab, "Sync")
	tabs.AddTab(serversTableTab, "Server Table")
	tabs.AddTab(expertTab, "Expert")

	mainLayout.AddWidget(tabs.QWidget)

	// Save/Cancel Buttons
	btnBox := qt.NewQDialogButtonBox5(qt.QDialogButtonBox__Save|qt.QDialogButtonBox__Cancel, qt.Horizontal)
	// Import/Export buttons
	importBtn := qt.NewQPushButton3("Import")
	exportBtn := qt.NewQPushButton3("Export")

	importBtn.OnClicked(func() { importConfig(settingsWindow.QWidget) })
	exportBtn.OnClicked(func() { exportConfig(settingsWindow.QWidget) })

	btnBox.AddButton(importBtn.QAbstractButton, qt.QDialogButtonBox__ActionRole)
	btnBox.AddButton(exportBtn.QAbstractButton, qt.QDialogButtonBox__ActionRole)

	btnBox.OnAccepted(func() {
		// On Save:
		general.Key("enckey").SetValue(enckeyEdit.Text())
		general.Key("ssh_client").SetValue(sshClientCombo.CurrentText())
		general.Key(osSuffix + "_ssh").SetValue(sshCmd.Text())
		general.Key(osSuffix + "_rdp").SetValue(rdpCmd.Text())
		general.Key(osSuffix + "_winbox").SetValue(winboxCmd.Text())

		serverstable.Key("disabletooltips").SetValue(strconv.FormatBool(disableTooltips.IsChecked()))
		serverstable.Key("disablerowtooltips").SetValue(strconv.FormatBool(disableRowTooltips.IsChecked()))
		serverstable.Key("HostColumn").SetValue(hostCol.Text())
		serverstable.Key("TypeColumn").SetValue(typeCol.Text())
		serverstable.Key("IpColumn").SetValue(ipCol.Text())
		serverstable.Key("UserColumn").SetValue(userCol.Text())
		serverstable.Key("DescriptionColumn").SetValue(descCol.Text())
		serverstable.Key("TagsColumn").SetValue(tagsCol.Text())
		serverstable.Key("SourceColumn").SetValue(sourceCol.Text())
		serverstable.Key("AvailabilityColumn").SetValue(availCol.Text())

		general.Key("sync").SetValue(strconv.FormatBool(syncCheckbox.IsChecked()))
		general.Key("defaultsshkey").SetValue(defaultSSHKey.Text())

		notes.Key("alwaysontop").SetValue(strconv.FormatBool(notesOnTopCheckbox.IsChecked()))

		// Save ignore files
		ignore := []string{}
		for i := 0; i < ignoreCombo.Count(); i++ {
			ignore = append(ignore, ignoreCombo.ItemText(i))
		}
		general.Key("ignore").SetValue(strings.Join(ignore, ","))

		// Save selected gist section
		name := selectedGist.CurrentText()
		if name != "" {
			sec := cfg.Section("gist " + name)
			sec.Key("gistid").SetValue(gistID.Text())
			sec.Key("gistsec").SetValue(gistSec.Text())
			sec.Key("enckey").SetValue(gistEnc.Text())
			sec.Key("encrypt_notes").SetValue(strconv.FormatBool(gistNotesEncryption.IsChecked()))
		}

		// Encryption handling as needed (add dialog if you wish)
		// Save config file
		log.Printf("saving the configuration...\n")
		if encryptSettingsCheckbox.IsChecked() && encryptionPass != "" {
			settings.DecryptPassword = encryptionPass
			if err := SaveEncryptedINI(cfg, configPath, encryptionPass); err != nil {
				log.Fatalf("failed to save encrypted settings: %s", err)
			}
		} else {
			cfg.SaveTo(configPath)
		}
		settingsWindow.Hide()
	})
	btnBox.OnRejected(func() {
		log.Printf("Dismissing settings window...\n")
		settingsWindow.Hide()
	})

	mainLayout.AddWidget(btnBox.QWidget)
	// Padding for the main layout
	mainLayout.SetContentsMargins(20, 20, 20, 20)
	mainLayout.SetSpacing(16)

	settingsWindow.SetLayout(mainLayout.QLayout)
	//settingsWindow.Exec()
	settingsWindow.Show()
	settingsWindow.Raise()
	settingsWindow.ActivateWindow()
	settingsWindow.SetFocus()
}
