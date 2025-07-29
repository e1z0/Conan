package main

import (
	"log"

	"github.com/mappu/miqt/qt"
)

var aboutWin *qt.QDialog
var anim *qt.QPropertyAnimation

func showAboutQt(parent *qt.QWidget) {
	if aboutWin != nil {
		aboutWin.Show()
		aboutWin.Raise()
		aboutWin.ActivateWindow()
		aboutWin.SetFocus()
		anim.Start()
		return
	}

	log.Printf("Loading about screen...\n")
	aboutWin = qt.NewQDialog(parent)
	aboutWin.SetWindowTitle("About " + appName)
	aboutWin.Resize(350, 400)

	aboutWin.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		aboutWin.Hide()
	})

	mainLayout := qt.NewQVBoxLayout2()

	// Logo image
	logoLabel := qt.NewQLabel(nil)
	pixmap := qt.NewQPixmap()
	pixmap.Load(":/Icon.png")
	scaled := pixmap.Scaled3(128, 128, qt.KeepAspectRatio, qt.SmoothTransformation)

	logoLabel.SetPixmap(scaled)
	logoLabel.SetAlignment(qt.AlignCenter)
	logoLabel.SetFixedSize2(128, 128)

	logoLabel.Move(aboutWin.Width()/2-64, 40) // center at start

	// Version label
	info := RepresentativeName + " Version " + version + " (build " + build + ") lines of code: " + lines + "\nCopyright (c) 2025 Justinas K (e1z0@icloud.com)"
	verLabel := qt.NewQLabel5(info, nil)
	verLabel.SetAlignment(qt.AlignCenter)

	// Spacer
	mainLayout.AddSpacing(15)

	// Add logo and version
	mainLayout.AddWidget(logoLabel.QWidget)
	mainLayout.AddWidget(verLabel.QWidget)

	mainLayout.AddSpacing(25)

	// Close button
	closeBtn := qt.NewQPushButton5("Close", nil)
	closeBtn.OnClicked(func() { aboutWin.Hide() })

	// Layout button at bottom center
	btnLayout := qt.NewQHBoxLayout2()
	btnLayout.AddStretch()
	btnLayout.AddWidget(closeBtn.QWidget)
	btnLayout.AddStretch()
	mainLayout.AddLayout(btnLayout.QLayout)

	aboutWin.SetLayout(mainLayout.QLayout)

	// --- Animation: Bounce logo up and down ---

	// Animate vertical position (move up and down 12px)
	startPos := logoLabel.Pos()
	endPos := qt.NewQPoint2(startPos.X(), startPos.Y()+50)

	anim = qt.NewQPropertyAnimation2(logoLabel.QObject, []byte("pos"))
	anim.SetDuration(5100)
	anim.SetStartValue(qt.NewQVariant27(startPos))
	anim.SetEndValue(qt.NewQVariant27(endPos))
	//anim.SetLoopCount(-1)
	anim.SetEasingCurve(qt.NewQEasingCurve3(qt.QEasingCurve__OutBounce))
	anim.SetDirection(qt.QAbstractAnimation__Forward)
	anim.Start()

	//aboutWin.Exec()
	aboutWin.Show()
	aboutWin.Raise()
	aboutWin.ActivateWindow()
	aboutWin.SetFocus()
}
