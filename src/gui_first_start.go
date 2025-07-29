package main

import (
	"fmt"

	"github.com/mappu/miqt/qt"
)

func showWelcomeWindow() {
	// Modal dialog
	welcomeWin := qt.NewQDialog(nil)
	welcomeWin.SetWindowTitle("Welcome to Conan")
	welcomeWin.Resize(450, 300)

	welcomeWin.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		welcomeWin.Hide()
	})

	// Vertical layout
	mainLayout := qt.NewQVBoxLayout(welcomeWin.QWidget)
	mainLayout.SetSpacing(18)
	mainLayout.SetContentsMargins(32, 32, 32, 32)

	// Headline (big, bold)
	title := qt.NewQLabel5("First Time Setup", nil)
	font := title.Font()
	font.SetPointSize(font.PointSize() + 4)
	font.SetBold(true)
	title.SetFont(font)
	title.SetAlignment(qt.AlignCenter)

	// Description
	descText := "Application is ready to use.\n"
	if welcome {
		descText = "It looks like this is your first time running the application.\nNo active settings are configured yet."
	}
	desc := qt.NewQLabel5(descText, nil)
	desc.SetAlignment(qt.AlignCenter)

	// Count label (bold)
	total := fmt.Sprintf("Total remote connections found: %d", len(servers))
	countLabel := qt.NewQLabel5(total, nil)
	cfont := countLabel.Font()
	cfont.SetBold(true)
	countLabel.SetFont(cfont)
	countLabel.SetAlignment(qt.AlignCenter)

	// Settings button
	settingsBtn := qt.NewQPushButton5("Settings", nil)
	settingsBtn.OnClicked(func() {
		//welcomeWin.Hide()
		showSettingsWindow(nil, env.settingsFile)
	})

	// Import config button
	importBtn := qt.NewQPushButton5("Import config", nil)
	importBtn.OnClicked(func() {
		importConfig(welcomeWin.QWidget)
	})

	// Center buttons using horizontal layouts
	btnSettingsLayout := qt.NewQHBoxLayout2()
	btnSettingsLayout.AddStretch()
	btnSettingsLayout.AddWidget(settingsBtn.QWidget)
	btnSettingsLayout.AddStretch()

	btnImportLayout := qt.NewQHBoxLayout2()
	btnImportLayout.AddStretch()
	btnImportLayout.AddWidget(importBtn.QWidget)
	btnImportLayout.AddStretch()

	// Add widgets to main layout
	mainLayout.AddWidget(title.QWidget)
	mainLayout.AddSpacing(8)
	mainLayout.AddWidget(desc.QWidget)
	mainLayout.AddSpacing(8)
	mainLayout.AddWidget(countLabel.QWidget)
	mainLayout.AddStretch()
	mainLayout.AddLayout(btnSettingsLayout.QLayout)
	mainLayout.AddLayout(btnImportLayout.QLayout)

	welcomeWin.SetLayout(mainLayout.QLayout)
	welcomeWin.Exec()
}
