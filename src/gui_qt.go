package main

import (
	"github.com/mappu/miqt/qt"
	"github.com/mappu/miqt/qt/mainthread"
)

// QT Toolkit helpers

// CallOnQtMain queues fn() to run on the Qt GUI thread, from any goroutine.
func CallOnQtMain(fn func()) {
	mainthread.Wait(fn)
}

func QTshowError(parent *qt.QWidget, title, message string) {
	qt.QMessageBox_Critical(parent, title, message)
}

func QTshowInfo(parent *qt.QWidget, title, message string) {
	qt.QMessageBox_Information(parent, title, message)
}

func QTshowWarn(parent *qt.QWidget, title, message string) {
	qt.QMessageBox_Warning(parent, title, message)
}

func QTWidgetWithLayout(layout *qt.QLayout) *qt.QWidget {
	w := qt.NewQWidget(nil)
	w.SetLayout(layout)
	return w
}

func QTPromptPasswordVerify(parent *qt.QWidget, labelText string) string {
	for {
		dlg1 := qt.NewQInputDialog(parent)
		dlg1.SetLabelText("Enter encryption password:")
		dlg1.SetTextEchoMode(qt.QLineEdit__Password)
		if dlg1.Exec() != int(qt.QDialog__Accepted) {
			return "" // Cancelled
		}
		pw1 := dlg1.TextValue()

		dlg2 := qt.NewQInputDialog(parent)
		dlg2.SetLabelText("Repeat encryption password:")
		dlg2.SetTextEchoMode(qt.QLineEdit__Password)
		if dlg2.Exec() != int(qt.QDialog__Accepted) {
			return "" // Cancelled
		}
		pw2 := dlg2.TextValue()

		if pw1 == pw2 && pw1 != "" {
			return pw1
		}

		// Show mismatch warning
		msg := qt.NewQMessageBox5(qt.QMessageBox__Warning, "Error", "Passwords do not match, try again.", qt.QMessageBox__Ok)
		msg.Exec()
	}
}

func AddBounceOnClick(btn *qt.QPushButton) {
	// Save the original geometry
	origGeom := btn.Geometry()
	scale := 0.88 // How much to "squash"

	btn.OnClicked(func() {
		// Animate shrink
		shrinkGeom := qt.NewQRect4(
			origGeom.X()+int(float64(origGeom.Width())*(1-scale)/2),
			origGeom.Y()+int(float64(origGeom.Height())*(1-scale)/2),
			int(float64(origGeom.Width())*scale),
			int(float64(origGeom.Height())*scale),
		)

		anim := qt.NewQPropertyAnimation2(btn.QObject, []byte("geometry"))
		anim.SetDuration(90) // ms
		anim.SetStartValue(qt.NewQVariant31(origGeom))
		anim.SetEndValue(qt.NewQVariant31(shrinkGeom))
		anim.SetEasingCurve(qt.NewQEasingCurve3(qt.QEasingCurve__OutQuad))

		// Animate back
		anim2 := qt.NewQPropertyAnimation2(btn.QObject, []byte("geometry"))
		anim2.SetDuration(90)
		anim2.SetStartValue(qt.NewQVariant31(shrinkGeom))
		anim2.SetEndValue(qt.NewQVariant31(origGeom))
		anim2.SetEasingCurve(qt.NewQEasingCurve3(qt.QEasingCurve__OutBounce))

		anim.OnFinished(func() {
			anim2.Start()
		})

		anim.Start()
	})
}
