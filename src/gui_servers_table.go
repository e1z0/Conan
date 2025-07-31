package main

/* Servers table
(c) 2025 e1z0 (e1z0@icloud.com)
sshexperiment - Conan project
*/

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mappu/miqt/qt"
	probing "github.com/prometheus-community/pro-bing"
)

var serverTableWindow *qt.QWidget
var lastSearch string // last‐used search term
var lastFoundRow int  // index of last match
var ServersListTable *qt.QTableWidget
var draggedRow int = -1

var ServerTableColumns = []string{
	"Host",
	"Type",
	"IP",
	"User",
	"Description",
	"Tags",
	"Source",
	"Availability",
}

func ShowConfirmDialog(parent *qt.QWidget, title, text string) bool {
	ret := qt.QMessageBox_Question4(
		parent,
		title,
		text,
		qt.QMessageBox__Yes, qt.QMessageBox__No)
	return ret == int(qt.QMessageBox__Yes)
}

func showCustomSearchDialog(parent *qt.QWidget, onSearch func(query string)) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Find server")

	layout := qt.NewQVBoxLayout(nil)
	entry := qt.NewQLineEdit(nil)
	entry.SetPlaceholderText("host, IP, user…")
	layout.AddWidget(entry.QWidget)

	buttonBox := qt.NewQDialogButtonBox5(qt.QDialogButtonBox__Ok|qt.QDialogButtonBox__Cancel, qt.Horizontal)

	layout.AddWidget(buttonBox.QWidget)

	dialog.SetLayout(layout.QLayout)

	// Handle OK/Cancel
	buttonBox.OnAccepted(func() {
		dialog.Accept()
	})
	buttonBox.OnRejected(func() {
		dialog.Reject()
	})

	// Optional: pressing Enter in entry triggers OK
	entry.OnReturnPressed(func() {
		dialog.Accept()
	})

	if dialog.Exec() == int(qt.QDialog__Accepted) {
		onSearch(entry.Text())
	}
	dialog.Destroy()
}

// Actual searchTable function, does the magic
func searchTable(query string, startRow int) int {
	n := len(servers)
	query = strings.ToLower(query)
	for i := 0; i < n; i++ {
		r := (startRow + i) % n
		s := servers[r]
		if strings.Contains(strings.ToLower(s.Host), query) ||
			strings.Contains(strings.ToLower(s.IP), query) ||
			strings.Contains(strings.ToLower(s.User), query) ||
			strings.Contains(strings.ToLower(s.Description), query) {
			return r
		}
	}
	return -1
}

// Search next function on CTRL+N
func searchNext() {
	if lastSearch == "" {
		return // nothing to repeat
	}
	idx := searchTable(lastSearch, lastFoundRow+1)
	if idx >= 0 {
		ServersListTable.SelectRow(idx)
		lastFoundRow = idx
	} else {
		// Optionally show a wrap-around message
		qt.QMessageBox_Information(serverTableWindow, "Search wrapped", "No more matches, wrapping to top")
		idx = searchTable(lastSearch, 0)
		if idx >= 0 {
			ServersListTable.SelectRow(idx)
			lastFoundRow = idx
		}
	}
}

func TableMoveRow(table *qt.QTableWidget, from, to int) {
	if from == to || from < 0 || to < 0 || from >= table.RowCount() || to >= table.RowCount() {
		return
	}

	// Move the items in the table
	cols := table.ColumnCount()
	data := make([]*qt.QTableWidgetItem, cols)
	for col := 0; col < cols; col++ {
		data[col] = table.TakeItem(from, col)
	}

	if from < to {
		for row := from; row < to; row++ {
			for col := 0; col < cols; col++ {
				item := table.TakeItem(row+1, col)
				table.SetItem(row, col, item)
			}
		}
	} else {
		for row := from; row > to; row-- {
			for col := 0; col < cols; col++ {
				item := table.TakeItem(row-1, col)
				table.SetItem(row, col, item)
			}
		}
	}

	for col := 0; col < cols; col++ {
		table.SetItem(to, col, data[col])
	}

	// Move the server in the slice to keep it synced
	if servers == nil || len(servers) == 0 {
		return
	}

	srvSlice := servers
	moved := srvSlice[from]

	// Remove from old position
	srvSlice = append(srvSlice[:from], srvSlice[from+1:]...)

	// Adjust `to` index if moving down, because slice got shorter
	if from < to {
		to--
	}

	// Insert at new position
	srvSlice = append(srvSlice[:to], append([]Server{moved}, srvSlice[to:]...)...)

	// Update the original slice pointer
	servers = srvSlice
}

func showServerTable() {
	// Prevent double creation
	if serverTableWindow != nil {
		serverTableWindow.Show()
		serverTableWindow.Raise()
		serverTableWindow.ActivateWindow()
		serverTableWindow.SetFocus()
		return
	}

	serverTableWindow = qt.NewQWidget(nil)
	serverTableWindow.SetWindowTitle("Servers Table")
	serverTableWindow.Resize(1000, 600)
	serverTableWindow.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		serverTableWindow.Hide()
	})

	updateServerTable()

	ServersListTable.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	ServersListTable.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	ServersListTable.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	ServersListTable.HorizontalHeader().SetSectionResizeMode(qt.QHeaderView__Interactive)
	ServersListTable.VerticalHeader().SetVisible(false)

	// movable row items
	//lastOrder := getRowOrder(ServersListTable)

	ServersListTable.SetDragEnabled(true)
	ServersListTable.SetDragDropMode(qt.QAbstractItemView__DragDrop)
	ServersListTable.SetDefaultDropAction(qt.MoveAction)
	ServersListTable.VerticalHeader().SetSectionsMovable(true)
	ServersListTable.SetAcceptDrops(true)
	ServersListTable.SetDragDropOverwriteMode(false)

	ServersListTable.OnDropEvent(func(super func(event *qt.QDropEvent), event *qt.QDropEvent) {
		pos := event.Pos()
		targetRow := ServersListTable.RowAt(pos.Y())

		selected := ServersListTable.SelectedItems()
		if len(selected) == 0 {
			return
		}

		// Get unique source rows
		sourceRows := map[int]struct{}{}
		for _, item := range selected {
			sourceRows[item.Row()] = struct{}{}
		}
		rows := make([]int, 0, len(sourceRows))
		for row := range sourceRows {
			rows = append(rows, row)
		}
		sort.Ints(rows)

		targetRowOffset := -1
		//var targetRowOffset int
		var targetuuid string
		// Move each row
		for i, row := range rows {
			offset := 0
			// check if targetRow is valid
			if targetRow >= 0 && targetRow < len(servers) {
				targetuuid = servers[targetRow].ID
				if row < targetRow {
					offset = i
				}
				TableMoveRow(ServersListTable, row, targetRow+offset)
			}
		}

		// save the aragement in servers file
		pushServersToFile()
		// workaround for server that lose positions, reload all the table...
		updateServerTable()

		for i, item := range servers {
			if item.ID == targetuuid {
				targetRowOffset = i
				break
			}
		}

		// the correct item should be selected depending on the uuid
		if targetRow > -1 {
			ServersListTable.SelectRow(targetRowOffset)
		}

		event.SetDropAction(qt.IgnoreAction) // prevents Qt from doing anything
		event.Accept()
	})

	// Handle double-click
	ServersListTable.OnCellDoubleClicked(func(row, col int) {
		if row >= 0 && row < len(servers) {
			println("Double clicked on:", servers[row].Host)
			go ClientConnect(servers[row])
		}
	})

	toolbar := qt.NewQToolBar(serverTableWindow)

	// Helper to add button with icon, tooltip, and callback
	addToolBtn := func(icon *qt.QIcon, name, tooltip string, cb func()) {
		action := qt.NewQAction6(icon, name, serverTableWindow.QObject)
		action.SetToolTip(tooltip)
		action.OnTriggered(cb)
		actions := []*qt.QAction{action}
		toolbar.AddActions(actions)
	}

	// standard icons
	//style := qt.QApplication_Style()
	//addIcon := style.StandardIcon(qt.QStyle__SP_FileIcon, nil, nil)
	//editIcon := style.StandardIcon(qt.QStyle__SP_DialogApplyButton, nil, nil)
	//deleteIcon := style.StandardIcon(qt.QStyle__SP_TrashIcon, nil, nil)
	//pushIcon := style.StandardIcon(qt.QStyle__SP_ArrowUp, nil, nil)
	//pullIcon := style.StandardIcon(qt.QStyle__SP_ArrowDown, nil, nil)
	//searchIcon := style.StandardIcon(qt.QStyle__SP_FileDialogContentsView, nil, nil)
	//pingIcon := style.StandardIcon(qt.QStyle__SP_MediaPlay, nil, nil)
	//importIcon := style.StandardIcon(qt.QStyle__SP_DirOpenIcon, nil, nil)
	//exportIcon := style.StandardIcon(qt.QStyle__SP_DriveFDIcon, nil, nil)
	//connectIcon := style.StandardIcon(qt.QStyle__SP_DriveNetIcon, nil, nil)
	//resizeIcon := style.StandardIcon(qt.QStyle__SP_TitleBarMaxButton, nil, nil)
	//quitIcon := style.StandardIcon(qt.QStyle__SP_DialogCloseButton, nil, nil)
	addIcon := qt.NewQIcon4(":/icons/new.png")
	editIcon := qt.NewQIcon4(":/icons/edit.png")
	deleteIcon := qt.NewQIcon4(":/icons/delete.png")
	pushIcon := qt.NewQIcon4(":/icons/syncpush.png")
	pullIcon := qt.NewQIcon4(":/icons/syncpull.png")
	searchIcon := qt.NewQIcon4(":/icons/search.png")

	pingIcon := qt.NewQIcon4(":/icons/ping.png")
	importIcon := qt.NewQIcon4(":/icons/import.png")
	exportIcon := qt.NewQIcon4(":/icons/export.png")
	connectIcon := qt.NewQIcon4(":/icons/connect.png")
	resizeIcon := qt.NewQIcon4(":/icons/auto.png")
	quitIcon := qt.NewQIcon4(":/icons/close.png")

	// custom icons support example needs to be embedded using src/resources.qrc and then "make embed"
	//newIcon := qt.NewQIcon4(":/qt-project.org/styles/commonstyle/images/file-128.png")

	deletefunc := func() {
		idx := ServersListTable.CurrentRow()
		if idx >= 0 && idx < len(servers) {
			if ShowConfirmDialog(serverTableWindow, "Delete?", "Are you sure you want to delete "+servers[idx].Host+" server?") {
				// User confirmed (Yes)
				log.Printf("deleting\n")
				servers = append(servers[:idx], servers[idx+1:]...)
				pushServersToFile()
				updateServerTable() // refresh the table
				updateTrayMenu()    // update tray menu
			} else {
				log.Printf("delete canceled\n")
			}
		} else {
			QTshowError(nil, "Error", "No server selected!")
			return
		}
	}

	addToolBtn(addIcon, "Add server", "Add a new server", func() {
		// show add dialog for servers[row]
		log.Printf("servers table add\n")
		showServerForm(nil, nil)
	})
	addToolBtn(editIcon, "Edit", "Edit selected server", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// show edit dialog for servers[row]
			log.Printf("servers table edit\n")
			showServerForm(&servers[row], nil)
		}
	})
	addToolBtn(deleteIcon, "Delete", "Delete selected server", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// confirm and delete servers[row]
			log.Printf("servers table delete\n")
			deletefunc()

		}
	})
	toolbar.AddSeparator()

	// search internal function as variable helper
	searchfunc := func() {
		if len(servers) > 0 {

			log.Printf("servers table search \n")
			showCustomSearchDialog(serverTableWindow, func(query string) {
				idx := searchTable(query, 0)
				if idx >= 0 {
					ServersListTable.SelectRow(idx)
					lastSearch = query
					lastFoundRow = idx
				} else {
					qt.QMessageBox_Information(serverTableWindow, "Not found", "No server matches \""+query+"\"")
				}
			})

		}
	}

	addToolBtn(searchIcon, "Search", "Filter your servers by name or IP", func() {
		searchfunc()
	})

	// Search hotkey
	FindShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+F"), ServersListTable.QWidget)
	FindShortcut.OnActivated(searchfunc)

	// Search next hotkey
	FindNextShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+N"), ServersListTable.QWidget)
	FindNextShortcut.OnActivated(searchNext)

	// Remove server hotkey
	DelShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Delete"), ServersListTable.QWidget)
	DelShortcut.OnActivated(deletefunc)
	// Remove server mac backspace key hotkey
	if runtime.GOOS == "darwin" {
		// Bind "Backspace" (normal Mac delete)
		backspaceShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Backspace"), ServersListTable.QWidget)
		backspaceShortcut.OnActivated(deletefunc)
	}

	// ping key
	PingShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+P"), ServersListTable.QWidget)
	PingShortcut.OnActivated(pingAll)

	// FIXME implement more key bindings
	// on fyne toolkit there was:
	/*
		P - ping all servers
		I - add server
		A - autosize
		E - edit
		Backspace/Del - delete
	*/

	addToolBtn(pingIcon, "Ping", "Ping all servers to check availability", func() {
		if len(servers) > 0 {
			// servers ping
			log.Printf("servers table ping\n")
			pingAll()
		}
	})
	toolbar.AddSeparator()

	addToolBtn(pushIcon, "Push", "Push serverrs to remote storage", func() {
		if len(servers) > 0 {
			// servers sync push
			log.Printf("servers table push\n")
			err := UploadGists()
			if err != nil {
				QTshowError(nil, "Error", fmt.Sprintf("Unable to push servers: %s", err))
				return
			}
			QTshowInfo(nil, "Info", "All servers pushed successfully.")
		} else {
			QTshowError(nil, "Error", "Servers list is empty!")
		}
	})

	addToolBtn(pullIcon, "Pull", "Pull servers from remote storage", func() {
		// servers sync pull
		err := DownloadGists()
		if err != nil {
			QTshowError(nil, "Error", fmt.Sprintf("Unable to pull servers: %s", err))
			return
		}
		fetchServersFromFiles()
		updateServerTable()
		updateTrayMenu()
		QTshowInfo(nil, "Info", "All servers pulled successfully.")
	})
	toolbar.AddSeparator()
	addToolBtn(importIcon, "Import", "Import servers from a file", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// Import servers
			log.Printf("servers table import\n")
			QTshowInfo(nil, "Infp", "This feature is not implemented yet.")
		}
	})
	addToolBtn(exportIcon, "Export", "Export servers to a file", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// Export servers
			log.Printf("servers table export\n")
			QTshowInfo(nil, "Infp", "This feature is not implemented yet.")
		}
	})
	toolbar.AddSeparator()
	addToolBtn(connectIcon, "Connect", "Connect to the selected server", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// Connect to server
			log.Printf("servers table Connect\n")
			go ClientConnect(servers[row])
		}
	})

	addToolBtn(resizeIcon, "AutoSize", "Autosize all columns of the table depending on the text length", func() {
		row := ServersListTable.CurrentRow()
		if row >= 0 && row < len(servers) {
			// Autoresize servers table columns
			log.Printf("servers table AutoSize\n")
			autoSizeColumns()
		}
	})
	toolbar.AddSeparator()
	addToolBtn(quitIcon, "Close", "Close this window", func() {
		log.Printf("servers table close\n")
		serverTableWindow.Close()
	})

	ServersListTable.SetContextMenuPolicy(qt.CustomContextMenu)
	ServersListTable.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		row := ServersListTable.RowAt(pos.Y())
		if row < 0 {
			return
		} // Not on a row

		menu := qt.NewQMenu(ServersListTable.QWidget)

		connectAction := qt.NewQAction3(connectIcon, "Connect")
		connectAction.SetToolTip("Connect to this server")
		connectAction.OnTriggered(func() {
			// server row connect context menu item
			go ClientConnect(servers[row])
		})

		editAction := qt.NewQAction3(editIcon, "Edit")
		editAction.SetToolTip("Edit this server")
		editAction.OnTriggered(func() {
			row := ServersListTable.CurrentRow()
			if row >= 0 && row < len(servers) {
				// server row edit context menu item
				log.Printf("servers table edit\n")
				showServerForm(&servers[row], nil)
			}
		})

		deleteAction := qt.NewQAction3(deleteIcon, "Delete")
		deleteAction.SetToolTip("Delete this server")
		deleteAction.OnTriggered(func() {
			// server row delete context menu item
			log.Printf("delete action")
			deletefunc()
		})

		menu.AddActions([]*qt.QAction{connectAction, editAction})
		menu.AddSeparator()
		menu.AddActions([]*qt.QAction{deleteAction})
		// launch context menu in the middle of row
		globalPos := ServersListTable.MapToGlobal(pos)
		menu.ExecWithPos(globalPos)
	})

	// Layout
	layout := qt.NewQVBoxLayout(nil)
	layout.AddWidget(toolbar.QWidget)
	layout.AddWidget(ServersListTable.QWidget)
	// set content bounds to match the window
	layout.SetContentsMargins(0, 0, 0, 0)
	layout.SetSpacing(0)
	serverTableWindow.SetLayout(layout.QLayout)
	serverTableWindow.Show()
	serverTableWindow.Raise()
	serverTableWindow.ActivateWindow()
	serverTableWindow.SetFocus()
}

func autoSizeColumns() {
	// For each column
	for col := 0; col < ServersListTable.ColumnCount(); col++ {
		ServersListTable.ResizeColumnToContents(col)
		// Optionally: Add extra padding
		current := ServersListTable.ColumnWidth(col)
		ServersListTable.SetColumnWidth(col, current+16) // +16 px padding (adjust as needed)
	}
}

// updateServerTable function updates or initializes the table with servers data and builds up the columns and adjusts the static sizes
func updateServerTable() {
	if ServersListTable != nil {
		// clear existing data if table is already initialized
		ServersListTable.ClearContents()
		ServersListTable.SetRowCount(0)
	} else {
		// Setup Table
		ServersListTable = qt.NewQTableWidget(nil)
	}
	ServersListTable.SetRowCount(len(servers))
	ServersListTable.SetColumnCount(len(ServerTableColumns))
	ServersListTable.SetHorizontalHeaderLabels(ServerTableColumns)

	/*  another way to add and set defaults, be we will not use it at this time
	for col, name := range ServerTableColumns {
		item := qt.NewQTableWidgetItem2(name)
		ServersListTable.SetHorizontalHeaderItem(col, item)
		ServersListTable.SetColumnWidth(col, 5)
	}*/

	// Fill in server data
	for row, s := range servers {
		hostitem := qt.NewQTableWidgetItem2(s.Host)
		typeitem := qt.NewQTableWidgetItem2(s.Type)
		ipitem := qt.NewQTableWidgetItem2(s.IP)
		useritem := qt.NewQTableWidgetItem2(s.User)
		descitem := qt.NewQTableWidgetItem2(s.Description)
		tagsitem := qt.NewQTableWidgetItem2(s.Tags)
		srcitem := qt.NewQTableWidgetItem2(s.SourceName)
		srcavail := qt.NewQTableWidgetItem2(s.Availability)
		if !settings.ServerTableGui.DisableTooltips {
			hostitem.SetToolTip(s.Host)
			typeitem.SetToolTip(s.Type)
			ipitem.SetToolTip(s.IP)
			useritem.SetToolTip(s.User)
			descitem.SetToolTip(s.Description)
			tagsitem.SetToolTip(s.Tags)
			srcitem.SetToolTip(s.SourceName)
			srcavail.SetToolTip(s.Availability)
		}
		ServersListTable.SetItem(row, 0, hostitem)
		ServersListTable.SetItem(row, 1, typeitem)
		ServersListTable.SetItem(row, 2, ipitem)
		ServersListTable.SetItem(row, 3, useritem)
		ServersListTable.SetItem(row, 4, descitem)
		ServersListTable.SetItem(row, 5, tagsitem)
		ServersListTable.SetItem(row, 6, srcitem)
		ServersListTable.SetItem(row, 7, srcavail)
	}

	// FIXME should be loaded from the config file is specified
	// host column size
	ServersListTable.SetColumnWidth(0, 150)
	// type column size
	ServersListTable.SetColumnWidth(1, 80)
	// ip column size
	ServersListTable.SetColumnWidth(2, 120)
	// user column size
	ServersListTable.SetColumnWidth(3, 100)
	// description column size
	ServersListTable.SetColumnWidth(4, 150)
	// tags column size
	ServersListTable.SetColumnWidth(5, 120)
	// source column size
	ServersListTable.SetColumnWidth(6, 120)
	// availability column size
	ServersListTable.SetColumnWidth(7, 120)

}

// showServerForm - create or edit server window
func showServerForm(s *Server, parent *qt.QWidget) {
	isNew := s == nil
	var srv Server
	if isNew {
		srv = Server{}
	} else {
		srv = *s
	}

	dialog := qt.NewQDialog(parent)
	if isNew {
		dialog.SetWindowTitle("New server")
	} else {
		dialog.SetWindowTitle(s.Host)
	}
	dialog.Resize(440, 510)

	formLayout := qt.NewQFormLayout(dialog.QWidget)

	// -- ID (readonly)
	idEdit := qt.NewQLineEdit(dialog.QWidget)
	idEdit.SetReadOnly(true)
	idEdit.SetText(srv.ID)
	formLayout.AddRow(qt.NewQLabel5("ID", dialog.QWidget).QWidget, idEdit.QWidget)

	// -- Source file (YAML), Combobox
	ymlnames := baseNames(ymlfiles)
	nameCombo := qt.NewQComboBox(dialog.QWidget)
	for _, y := range ymlnames {
		nameCombo.AddItem(y)
	}
	if !isNew {
		for i, y := range ymlnames {
			if y == srv.SourceName {
				nameCombo.SetCurrentIndex(i)
				break
			}
		}
		nameCombo.SetEnabled(false)
	}
	formLayout.AddRow(qt.NewQLabel5("File", dialog.QWidget).QWidget, nameCombo.QWidget)

	// -- Host
	hostEdit := qt.NewQLineEdit(dialog.QWidget)
	hostEdit.SetText(srv.Host)
	formLayout.AddRow(qt.NewQLabel5("Host", dialog.QWidget).QWidget, hostEdit.QWidget)

	// -- IP
	ipEdit := qt.NewQLineEdit(dialog.QWidget)
	ipEdit.SetText(srv.IP)
	formLayout.AddRow(qt.NewQLabel5("IP", dialog.QWidget).QWidget, ipEdit.QWidget)

	// -- User
	userEdit := qt.NewQLineEdit(dialog.QWidget)
	userEdit.SetText(srv.User)
	formLayout.AddRow(qt.NewQLabel5("User", dialog.QWidget).QWidget, userEdit.QWidget)

	// -- Password (as password field)
	passEdit := qt.NewQLineEdit(dialog.QWidget)
	passEdit.SetEchoMode(qt.QLineEdit__Password)
	passEdit.SetText(srv.DecryptPassword())
	// Checkbox to show/hide password
	showPwCheck := qt.NewQCheckBox4("Show password", dialog.QWidget)
	showPwCheck.OnStateChanged(func(state int) {
		if state == int(qt.Checked) {
			passEdit.SetEchoMode(qt.QLineEdit__Normal) // Show as plain text
		} else {
			passEdit.SetEchoMode(qt.QLineEdit__Password) // Masked
		}
	})

	pwRowWidget := qt.NewQWidget(dialog.QWidget)
	pwRowLayout := qt.NewQHBoxLayout(pwRowWidget)
	pwRowLayout.AddWidget(passEdit.QWidget)
	pwRowLayout.AddWidget(showPwCheck.QWidget)
	formLayout.AddRow(qt.NewQLabel5("Password", dialog.QWidget).QWidget, pwRowWidget)

	// -- PrivateKey
	keyEdit := qt.NewQLineEdit(dialog.QWidget)
	keyEdit.SetText(srv.PrivateKey)
	formLayout.AddRow(qt.NewQLabel5("PrivateKey", dialog.QWidget).QWidget, keyEdit.QWidget)

	// -- Port
	portEdit := qt.NewQLineEdit(dialog.QWidget)
	portEdit.SetText(srv.Port)
	formLayout.AddRow(qt.NewQLabel5("Port", dialog.QWidget).QWidget, portEdit.QWidget)

	// -- Type (combobox)
	typeCombo := qt.NewQComboBox(dialog.QWidget)
	for _, t := range ServerTypes {
		typeCombo.AddItem(t)
	}
	if !isNew {
		for i, t := range ServerTypes {
			if t == srv.Type {
				typeCombo.SetCurrentIndex(i)
				break
			}
		}
	}
	formLayout.AddRow(qt.NewQLabel5("Type", dialog.QWidget).QWidget, typeCombo.QWidget)

	// -- Tags
	tagsEdit := qt.NewQLineEdit(dialog.QWidget)
	tagsEdit.SetText(srv.Tags)
	formLayout.AddRow(qt.NewQLabel5("Tags", dialog.QWidget).QWidget, tagsEdit.QWidget)

	// -- Description (multiline)
	descEdit := qt.NewQTextEdit(dialog.QWidget)
	descEdit.SetText(srv.Description)
	formLayout.AddRow(qt.NewQLabel5("Description", dialog.QWidget).QWidget, descEdit.QWidget)

	// -- OK and Cancel buttons
	btnBox := qt.NewQDialogButtonBox(dialog.QWidget)
	btnBox.SetStandardButtons(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	formLayout.AddRow(nil, btnBox.QWidget)

	// --- BUTTON LOGIC ---
	btnBox.OnAccepted(func() {
		// Validation
		if isNew {
			if hostEdit.Text() == "" {
				qt.QMessageBox_Warning(dialog.QWidget, "Info", "No hostname specified!")
				return
			}
			if ipEdit.Text() == "" {
				qt.QMessageBox_Warning(dialog.QWidget, "Info", "No IP specified!")
				return
			}
			if nameCombo.CurrentText() == "" {
				qt.QMessageBox_Warning(dialog.QWidget, "Info", "No file selected!")
				return
			}
			if typeCombo.CurrentText() == "" {
				qt.QMessageBox_Warning(dialog.QWidget, "Info", "No type selected!")
				return
			}
			srv.ID = uuid.NewString()
			srv.SourceName = nameCombo.CurrentText()
			selectedFileFullPath, err := fullPathFor(srv.SourceName, ymlfiles)
			if err != nil {
				log.Printf("Error finding full path for %s: %v\n", srv.SourceName, err)
			}
			srv.SourcePath = selectedFileFullPath
		}

		srv.Host = hostEdit.Text()
		srv.IP = ipEdit.Text()
		srv.User = userEdit.Text()
		srv.Port = portEdit.Text()
		srv.PrivateKey = keyEdit.Text()
		srv.Type = typeCombo.CurrentText()
		srv.Tags = tagsEdit.Text()
		srv.Description = descEdit.ToPlainText()
		srv.Password = srv.EncryptPassword(passEdit.Text())

		if isNew {
			servers = append(servers, srv)
		} else {
			for i := range servers {
				if servers[i].ID == srv.ID {
					servers[i] = srv
					break
				}
			}
		}
		pushServersToFile()
		updateServerTable()
		updateTrayMenu()
		dialog.Accept()
	})

	btnBox.OnRejected(func() {
		dialog.Reject()
	})

	dialog.SetLayout(formLayout.QLayout)
	dialog.Exec()
}

func pingAll() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // up to 10 concurrent
	go func() {
		for i := range servers {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				p, err := probing.NewPinger(servers[idx].IP)
				if err != nil {
					servers[idx].Availability = "Error"
				} else {
					p.Count = 3
					p.Timeout = time.Second * 2
					// If you have a GetOS() function, keep it:
					if GetOS() == "windows" {
						p.SetPrivileged(true)
					} else {
						p.SetPrivileged(false)
					}
					if err = p.Run(); err != nil {
						servers[idx].Availability = "Down"
					} else if p.Statistics().PacketsRecv > 0 {
						servers[idx].Availability = "Up"
					} else {
						servers[idx].Availability = "Down"
					}
				}
				log.Printf("Server: %s availability: %s\n", servers[idx].Host, servers[idx].Availability)
				CallOnQtMain(updateServerTable)
			}(i)

		}
		wg.Wait()
		CallOnQtMain(updateServerTable)
	}()
}
