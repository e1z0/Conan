package main

import (
	"log"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/mappu/miqt/qt"
)

var (
	filteredItems []Server
	listWidget    *qt.QListWidget
	entry         *qt.QLineEdit
	searchWindow  *qt.QWidget
)

func showFuzzySearchWindow() {
	if searchWindow != nil {
		searchWindow.Show()
		searchWindow.Raise()
		searchWindow.ActivateWindow()
		searchWindow.SetFocus()
		entry.SetFocus()
		return
	}
	searchWindow := qt.NewQWidget(nil)
	searchWindow.SetWindowTitle("")
	searchWindow.Resize(520, 340)

	searchWindow.SetWindowFlags(qt.FramelessWindowHint | qt.WindowStaysOnTopHint | qt.Dialog)
	searchWindow.SetAttribute(qt.WA_TranslucentBackground)

	searchWindow.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		searchWindow.Hide()
	})
	screen := qt.QApplication_Desktop().ScreenGeometry(searchWindow)
	x := (screen.Width() - searchWindow.Width()) / 2
	y := (screen.Height() - searchWindow.Height()) / 3
	searchWindow.Move(x, y)

	// Set always-on-top if you wish, comment out if error
	//searchWindow.SetWindowFlags(searchWindow.WindowFlags() | qt.WindowStaysOnTopHint)

	entry = qt.NewQLineEdit(nil)
	entry.SetPlaceholderText("Type to search...")
	font := qt.NewQFont6("Helvetica Neue", 21)
	font.SetBold(true)
	//, 21, int(qt.QFont__Normal), false)
	entry.SetFont(font)
	entry.SetStyleSheet(`
    QLineEdit {
        padding: 14px;
        border-radius: 18px;
        background: rgba(240,240,240,0.88);
        border: none;
        color: #222;
        margin: 10px 10px 18px 10px;
    }
`)

	listWidget = qt.NewQListWidget(nil)
	listWidget.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	listWidget.SetStyleSheet(`
    QListWidget {
        border: none;
        background: rgba(255,255,255,0.94);
        font-size: 18px;
        color: #333;
        padding: 5 10px 0 10px;
    }
    QListWidget::item:selected {
        background: #007aff;
        color: white;
        border-radius: 8px;
    }
    QListWidget::item {
        padding: 7px 10px 7px 10px;
        margin-bottom: 1px;
    }
`)

	layout := qt.NewQVBoxLayout(nil)
	layout.SetContentsMargins(24, 24, 24, 24)
	layout.AddWidget(entry.QWidget)
	layout.AddWidget(listWidget.QWidget)
	searchWindow.SetLayout(layout.QLayout)

	// Initial fill
	//filteredItems = make([]Server, len(servers))
	copy(filteredItems, servers)
	updateFuzzyList("")

	// Live search update
	entry.OnTextChanged(func(text string) {
		updateFuzzyList(text)
	})

	ConnectCommand := func(srv Server) {
		go ClientConnect(srv)
		searchWindow.Hide()
	}

	// Handle return key in entry
	entry.OnReturnPressed(func() {
		row := listWidget.CurrentRow()
		if row >= 0 && row < len(filteredItems) {
			log.Printf("Connecting to %s\n", filteredItems[row])
			ConnectCommand(filteredItems[row])
		}
	})

	// Handle list activation (double-click or Enter)
	listWidget.OnItemActivated(func(item *qt.QListWidgetItem) {
		row := listWidget.CurrentRow()
		if row >= 0 && row < len(filteredItems) {
			log.Printf("Connecting to %s\n", filteredItems[row])
			ConnectCommand(filteredItems[row])
		}
		entry.SetFocus() // return focus to entry
	})

	listWidget.OnItemClicked(func(item *qt.QListWidgetItem) {
		entry.SetFocus()
	})

	listWidget.OnKeyPressEvent(func(super func(event *qt.QKeyEvent), event *qt.QKeyEvent) {
		switch event.Key() {
		case int(qt.Key_Escape):
			searchWindow.Hide()
		default:
			super(event)
		}
	})

	// Optional: ESC closes window (capture keypresses on entry)

	entry.OnKeyPressEvent(func(super func(param1 *qt.QKeyEvent), param1 *qt.QKeyEvent) {
		switch param1.Key() {
		case int(qt.Key_Escape):
			searchWindow.Hide()
		case int(qt.Key_Up):
			curr := listWidget.CurrentRow()
			if curr > 0 {
				listWidget.SetCurrentRow(curr - 1)
			}
		case int(qt.Key_Down):
			curr := listWidget.CurrentRow()
			if curr < listWidget.Count()-1 {
				listWidget.SetCurrentRow(curr + 1)
			}
		default:
			super(param1)
		}
	})
	searchWindow.Show()
	searchWindow.Raise()
	searchWindow.ActivateWindow()
	searchWindow.SetFocus()
	entry.SetFocus()

}

// This must be adapted to keep filteredItems in sync in your real app.
// For demo purposes, pass as parameter; for real code, make it global or use closure.
func updateFuzzyList(query string) {
	query = strings.ToLower(query)
	listWidget.Clear()
	if query == "" {
		filteredItems = servers
	} else {
		filteredItems = nil
		for _, s := range servers {
			if fuzzy.Match(query, strings.ToLower(s.Host)) ||
				fuzzy.Match(query, strings.ToLower(s.IP)) ||
				fuzzy.Match(query, strings.ToLower(s.Description)) {
				filteredItems = append(filteredItems, s)
			}
		}
	}
	for _, s := range filteredItems {
		listWidget.AddItem(s.Host)
	}
	if len(filteredItems) > 0 {
		listWidget.SetCurrentRow(0)
	}
}

func onSubmitFuzzyList() {
	row := listWidget.CurrentRow()
	if row >= 0 && row < len(filteredItems) {
		go ClientConnect(filteredItems[row])
		searchWindow.Hide()
	}
}

func onSelectFuzzyList(row int) {
	if row >= 0 && row < len(filteredItems) {
		go ClientConnect(filteredItems[row])
		searchWindow.Hide()
	}
}
