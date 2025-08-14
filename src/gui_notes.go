package main

/* Notes frontend processing
(c) e1z0 2025
*/

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mappu/miqt/qt"
)

var notesWindowsQt = make(map[string]*NoteWindowQt)
var folderIcon *qt.QIcon
var folderOpenIcon *qt.QIcon
var fileIcon *qt.QIcon

// NoteWindowQt is a Qt port of your NotesWindow struct.
type NoteWindowQt struct {
	app           *qt.QApplication // Qt app
	win           *qt.QDialog      // Window for this notes dir
	treeData      map[string][]string
	isBranch      map[string]bool
	selectedUID   string
	viewMode      bool
	editor        *qt.QTextEdit
	viewer        *qt.QTextBrowser
	createdLbl    *qt.QLabel
	modifiedLbl   *qt.QLabel
	revisionsLbl  *qt.QLabel
	stickyCheck   *qt.QCheckBox
	viewContainer *qt.QWidget
	treeWidget    *qt.QTreeWidget
	gist          GistConfig
	service       *NoteService
	current       *Note
	// search state
	searchQuery      string
	searchHits       []SearchHit
	currentHitIndex  int
	pendingHit       *SearchHit
	highlightQTimer  *qt.QTimer
	highlightRetries int
}

func ShowNotesWindowQt(app *qt.QApplication, notesDir string, gist GistConfig) {
	absDir, _ := filepath.Abs(notesDir)
	key := filepath.Clean(absDir)

	if win, ok := notesWindowsQt[key]; ok {
		win.win.Show()
		win.win.ActivateWindow()
		win.win.Raise()
		return
	}
	service := &NoteService{NotesDir: notesDir, HistoryDir: ".history", Gist: gist}
	nw := &NoteWindowQt{
		app:     app,
		gist:    gist,
		service: service,
	}
	nw.initUI()
	notesWindowsQt[key] = nw
}

func (nw *NoteWindowQt) initUI() {
	nw.win = qt.NewQDialog(nil)
	nw.win.SetWindowTitle("Notes: " + filepath.Base(nw.service.NotesDir))
	nw.win.Resize(800, 600)
	nw.win.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		event.Ignore()
		nw.win.Hide()
	})

	// initialize icons
	style := qt.QApplication_Style()
	folderIcon = style.StandardIcon(qt.QStyle__SP_DirClosedIcon, nil, nil)
	folderOpenIcon = style.StandardIcon(qt.QStyle__SP_DirOpenIcon, nil, nil)
	fileIcon = style.StandardIcon(qt.QStyle__SP_FileIcon, nil, nil)

	os.MkdirAll(nw.service.NotesDir, 0755)

	// --- Tree
	td, ib, _ := nw.service.ListTree()
	nw.treeData, nw.isBranch = td, ib

	tree := qt.NewQTreeWidget(nil)
	tree.SetHeaderHidden(true)
	populateNotesTreeQt(tree, td, ib)
	nw.treeWidget = tree

	tree.OnItemExpanded(func(item *qt.QTreeWidgetItem) {
		if item.Data(1, int(qt.UserRole)).ToBool() {
			item.SetIcon(0, folderOpenIcon)
		}
	})
	tree.OnItemCollapsed(func(item *qt.QTreeWidgetItem) {
		if item.Data(1, int(qt.UserRole)).ToBool() {
			item.SetIcon(0, folderIcon)
		}
	})

	tree.OnItemSelectionChanged(func() {
		nw.onSelectQt()
	})

	// --- Editor & Viewer
	nw.editor = qt.NewQTextEdit(nil)
	nw.editor.OnInsertFromMimeData(func(super func(source *qt.QMimeData), source *qt.QMimeData) {
		if source.HasText() {
			nw.editor.InsertPlainText(source.Text())
			return
		}
		super(source)
	})

	nw.viewer = qt.NewQTextBrowser(nil)
	nw.editor.SetVisible(false)
	nw.viewer.SetVisible(true)

	nw.viewer.SetOpenLinks(false)
	nw.viewer.SetOpenExternalLinks(false)

	nw.viewer.OnAnchorClicked(func(url *qt.QUrl) {
		qt.QDesktopServices_OpenUrl(url)
	})

	// default is viewmode
	nw.viewMode = true

	// --- Footer labels
	nw.createdLbl = qt.NewQLabel5("", nil)
	nw.modifiedLbl = qt.NewQLabel5("", nil)
	nw.revisionsLbl = qt.NewQLabel5("", nil)
	nw.stickyCheck = qt.NewQCheckBox4("Sticky", nil)
	nw.stickyCheck.OnStateChanged(func(state int) {
		if nw.current != nil {
			nw.current.Meta.Sticky = nw.stickyCheck.IsChecked()
			_ = nw.service.Save(nw.current)
			key := filepath.Base(nw.service.NotesDir)
			if val, ok := Stickies[key]; ok {
				val.Refresh()
			}

		}
	})

	footer := qt.NewQHBoxLayout2()
	footer.AddWidget(qt.NewQLabel5("Created:", nil).QWidget)
	footer.AddWidget(nw.createdLbl.QWidget)
	footer.AddStretch()
	footer.AddWidget(qt.NewQLabel5("Modified:", nil).QWidget)
	footer.AddWidget(nw.modifiedLbl.QWidget)
	footer.AddStretch()
	footer.AddWidget(qt.NewQLabel5("Revisions:", nil).QWidget)
	footer.AddWidget(nw.revisionsLbl.QWidget)
	footer.AddStretch()
	footer.AddWidget(nw.stickyCheck.QWidget)

	nw.viewContainer = qt.NewQWidget(nil)
	vl := qt.NewQVBoxLayout2()
	vl.AddWidget(nw.viewer.QWidget)
	vl.AddLayout(footer.QLayout)
	nw.viewContainer.SetLayout(vl.QLayout)

	// --- Toolbar
	addNoteIcon := qt.NewQIcon4(":/icons/newnote.png")
	addFolderIcon := qt.NewQIcon4(":/icons/newfolder.png")
	searchIcon := qt.NewQIcon4(":/icons/search.png")
	deleteIcon := qt.NewQIcon4(":/icons/delete.png")
	saveIcon := qt.NewQIcon4(":/icons/save.png")
	viewModeIcon := qt.NewQIcon4(":/icons/show-hide.png")
	pushIcon := qt.NewQIcon4(":/icons/syncpush.png")
	pullIcon := qt.NewQIcon4(":/icons/syncpull.png")
	quitIcon := qt.NewQIcon4(":/icons/close.png")

	toolbar := qt.NewQHBoxLayout2()

	// --- Search box + hotkey
	// Search hotkey
	FindShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+F"), nw.win.QWidget)
	FindShortcut.OnActivated(nw.searchNotesQt)
	FindNextShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+N"), nw.win.QWidget)
	FindNextShortcut.OnActivated(nw.gotoNextHit)

	// GUI-thread timer for deferred highlight (no goroutines)
	nw.highlightQTimer = qt.NewQTimer()
	nw.highlightQTimer.SetInterval(50)
	nw.highlightQTimer.SetSingleShot(false)
	nw.highlightQTimer.OnTimeout(func() {
		if nw.pendingHit == nil {
			nw.highlightQTimer.Stop()
			return
		}
		want := strings.ReplaceAll(nw.pendingHit.RelPath, "\\", "/")
		have := strings.ReplaceAll(nw.selectedUID, "\\", "/")
		if have != want {
			nw.highlightRetries++
			if nw.highlightRetries > 60 {
				log.Printf("[notes] highlight timeout: want=%q have=%q", want, have)
				nw.pendingHit = nil
				nw.highlightQTimer.Stop()
			}
			return
		}
		h := *nw.pendingHit
		nw.pendingHit = nil
		nw.highlightQTimer.Stop()

		if nw.viewMode {
			nw.highlightInTextWidgetViewer(nw.searchQuery, h.Occurrence)
		} else {
			nw.highlightInTextWidgetEditor(nw.searchQuery, h.Occurrence)
		}
	})

	// add button helper function
	addToolBtn := func(icon *qt.QIcon, tooltip string, cb func()) {
		btnn := qt.NewQPushButton4(icon, "")
		btnn.SetIconSize(qt.NewQSize2(48, 48))
		btnn.SetToolTip(tooltip)
		btnn.SetStyleSheet(`
    QPushButton {
        border: none;
        background: transparent;
        padding: 2px;
    }
    QPushButton:hover {
        background:rgba(23, 180, 228, 0.73);
		padding: 10px; /* visually larger, but the button resizes */
    }
`)
		btnn.OnClicked(cb)
		AddBounceOnClick(btnn)
		toolbar.AddWidget(btnn.QWidget)
	}

	// toolbar buttons
	addToolBtn(addNoteIcon, "Add new note", func() { nw.doNewNoteQt() })
	addToolBtn(addFolderIcon, "Add folder", func() { nw.doNewFolderQt() })
	addToolBtn(searchIcon, "Search notes", func() { nw.searchNotesQt() })
	addToolBtn(deleteIcon, "Delete folder or note", func() { nw.doDeleteQt() })
	addToolBtn(saveIcon, "Save current note", func() { nw.saveNoteQt() })
	addToolBtn(viewModeIcon, "View mode editor or viewer", func() { nw.toggleViewQt() })
	addToolBtn(pushIcon, "Upload notes to the remote server (sync push)", func() { nw.pushSyncQt() })
	addToolBtn(pullIcon, "Download notes from the remote server (sync pull)", func() { nw.pullSyncQt() })
	addToolBtn(quitIcon, "Close notes window", func() {
		nw.win.Close()
	})

	splitter := qt.NewQSplitter3(qt.Horizontal)

	// debounce
	splitterSaveTimer := qt.NewQTimer()
	splitterSaveTimer.SetSingleShot(true)
	splitterSaveTimer.OnTimeout(func() {
		log.Printf("Saving splitter state")
		// save splitter state to settings
		Store.Set("Window-"+filepath.Base(nw.service.NotesDir), "splitter", splitter.SaveState())
	})

	splitter.OnSplitterMoved(func(pos, index int) {
		//log.Printf("Splitter moved: pos=%d, index=%d\n", pos, index)
		splitterSaveTimer.Start(800)
	})

	//nw.treeWidget.QWidget.SetFixedWidth(200)
	//	SetMinimumWidth(200) // Try 350, or whatever you like
	nw.treeWidget.QWidget.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Expanding)

	splitter.AddWidget(nw.treeWidget.QWidget)

	// 2. Right side: QWidget with its own QVBoxLayout
	right := qt.NewQWidget(nil)
	rightLayout := qt.NewQVBoxLayout2()

	// Add the toolbar at the top (as a widget or layout)
	rightLayout.AddLayout(toolbar.QLayout) // or AddWidget(toolbar.QWidget)
	rightLayout.AddWidget(nw.editor.QWidget)
	rightLayout.AddWidget(nw.viewContainer)
	right.SetLayout(rightLayout.QLayout)
	right.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Expanding)

	// 3. Add right pane to splitter
	splitter.AddWidget(right)
	splitter.SetSizes([]int{200, 400})
	splitter.SetStretchFactor(0, 1) // Left pane gets 2x the "stretch weight"
	splitter.SetStretchFactor(1, 2)

	nw.treeWidget.QWidget.SetMinimumWidth(200)
	nw.treeWidget.QWidget.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Expanding)
	right.SetMinimumWidth(400)
	right.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Expanding)
	splitter.AddWidget(nw.treeWidget.QWidget)
	splitter.AddWidget(right)
	splitter.SetSizes([]int{350, 650})

	// Add toolbar above
	mainLayout := qt.NewQVBoxLayout2()
	//mainLayout.AddLayout(toolbar.QLayout)
	mainLayout.AddWidget(splitter.QWidget)
	mainLayout.SetContentsMargins(0, 0, 0, 0) // Remove all space around
	mainLayout.SetSpacing(0)                  // Optional: remove spacing between widgets
	nw.win.SetLayout(mainLayout.QLayout)

	data, err := Store.GetBytes("Window-"+filepath.Base(nw.service.NotesDir), "splitter")
	if err == nil {
		ok := splitter.RestoreState(data)
		log.Printf("Restoring splitter state: success=%v", ok)
	}

	nw.win.Show()
	nw.win.Raise()
	nw.win.ActivateWindow()
	nw.win.SetFocus()
}

func populateNotesTreeQt(tree *qt.QTreeWidget, treeData map[string][]string, isBranch map[string]bool) {
	tree.Clear()

	// For every child of the root (path == ""), add as a top-level item
	for _, child := range treeData[""] {
		populateTreeNode(tree, nil, child, treeData, isBranch)
	}
	tree.ExpandAll()
}

func populateTreeNode(tree *qt.QTreeWidget, parent *qt.QTreeWidgetItem, path string, treeData map[string][]string, isBranch map[string]bool) {
	base := filepath.Base(path)
	var item *qt.QTreeWidgetItem
	if parent == nil {
		item = qt.NewQTreeWidgetItem3(tree)
		tree.AddTopLevelItem(item)
	} else {
		item = qt.NewQTreeWidgetItem6(parent)
		parent.AddChild(item)
	}
	// Store relPath as item data for quick access on selection
	item.SetData(0, int(qt.UserRole), qt.NewQVariant14(path))
	item.SetData(1, int(qt.UserRole), qt.NewQVariant11(isBranch[path]))
	if !isBranch[path] && strings.HasSuffix(base, ".md") {
		item.SetText(0, strings.TrimSuffix(base, ".md"))
		item.SetIcon(0, fileIcon)
	} else {
		item.SetText(0, base)
		//item.SetIcon(0, folderIcon)
		item.SetIcon(0, folderOpenIcon)
	}
	if isBranch[path] {
		for _, child := range treeData[path] {
			populateTreeNode(tree, item, child, treeData, isBranch)
		}
	}
}

func (nw *NoteWindowQt) onSelectQt() {
	item := nw.treeWidget.CurrentItem()
	if item == nil {
		log.Printf("note item is nil\n")
		return
	}
	// Reconstruct relative path from tree (or store mapping)
	// Assume here each item's Data holds relative path
	relPath := getItemRelPath(item)
	nw.selectedUID = relPath

	// Folder (branch): clear view/editor
	if nw.isBranch[relPath] {
		log.Printf("note is directory\n")
		nw.current = nil
		nw.editor.SetPlainText("")
		nw.viewer.SetMarkdown("")
		nw.createdLbl.SetText("")
		nw.modifiedLbl.SetText("")
		nw.revisionsLbl.SetText("")
		return
	}

	if !nw.isBranch[relPath] && strings.HasSuffix(relPath, ".md") {
		note, err := nw.service.Load(relPath)
		if err != nil {
			// FIXME when deleting note it throws up, we need to avoid it
			//qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
			return
		}
		nw.current = note

		nw.editor.SetPlainText(string(note.Body))
		if nw.viewMode {
			nw.viewer.SetMarkdown(string(note.Body))
			nw.updateHeaderQt()
		}
	}
}

func getItemRelPath(item *qt.QTreeWidgetItem) string {
	// Retrieve the relPath stored as data
	return item.Data(0, int(qt.UserRole)).ToString()
}

func (nw *NoteWindowQt) doNewNoteQt() {
	inputDlg := qt.NewQInputDialog(nw.win.QWidget)
	inputDlg.SetLabelText("Note name:")
	if inputDlg.Exec() == int(qt.QDialog__Accepted) && inputDlg.TextValue() != "" {
		parentRel := ""
		if nw.selectedUID != "" && nw.isBranch[nw.selectedUID] {
			parentRel = nw.selectedUID
		}
		if err := nw.service.NewNote(parentRel, inputDlg.TextValue()); err != nil {
			qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
			return
		}
		nw.refreshTreeQt()
	}
}

func (nw *NoteWindowQt) doNewFolderQt() {
	inputDlg := qt.NewQInputDialog(nw.win.QWidget)
	inputDlg.SetLabelText("Folder name:")
	if inputDlg.Exec() == int(qt.QDialog__Accepted) && inputDlg.TextValue() != "" {
		parentRel := ""
		if nw.selectedUID != "" && nw.isBranch[nw.selectedUID] {
			parentRel = nw.selectedUID
		}
		if err := nw.service.NewFolder(parentRel, inputDlg.TextValue()); err != nil {
			qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
			return
		}
		nw.refreshTreeQt()
	}
}

// startSearch indexes occurrences across all notes and jumps to the first hit.
func (nw *NoteWindowQt) startSearch(q string) {
	nw.searchQuery = q
	nw.searchHits = nil
	nw.currentHitIndex = -1
	if q == "" {
		return
	}
	hits, err := nw.service.SearchOccurrencesCI(q)

	if err != nil || len(hits) == 0 {
		return
	}
	nw.searchHits = hits
	nw.gotoNextHit()
}

// gotoNextHit selects the next hit (Ctrl+N).
func (nw *NoteWindowQt) gotoNextHit() {

	if len(nw.searchHits) == 0 {
		return
	}
	nw.currentHitIndex = (nw.currentHitIndex + 1) % len(nw.searchHits)
	h := nw.searchHits[nw.currentHitIndex]
	nw.openAndHighlight(h)
}

// openAndHighlight selects the note in the tree and highlights the hit.
func (nw *NoteWindowQt) openAndHighlight(h SearchHit) {
	if item := nw.findTreeItemByRelPath(h.RelPath); item != nil {
		nw.treeWidget.SetCurrentItem(item)
	} else {
		log.Printf("[notes] tree item NOT found for %q (root=%q)", h.RelPath, nw.service.NotesDir)
	}
	nw.pendingHit = &SearchHit{RelPath: h.RelPath, Occurrence: h.Occurrence}
	nw.highlightRetries = 0
	nw.highlightQTimer.Start2()
}

// highlight in editor (QTextEdit) — manual selection to avoid QTextCharFormat queuing
func (nw *NoteWindowQt) highlightInTextWidgetEditor(query string, occurrence int) {
	if nw.editor == nil || query == "" {
		return
	}
	cur := nw.editor.TextCursor()
	cur.MovePosition3(qt.QTextCursor__Start, qt.QTextCursor__MoveAnchor, 1)
	nw.editor.SetTextCursor(cur)

	for i := 0; i <= occurrence; i++ {
		if !nw.editor.Find3(query, 0) { // 0 flags => case-insensitive
			return
		}
	}
	nw.editor.SetFocus()
}

// highlight in viewer (QTextBrowser)
func (nw *NoteWindowQt) highlightInTextWidgetViewer(query string, occurrence int) {
	if nw.viewer == nil || query == "" {
		return
	}
	cur := nw.viewer.TextCursor()
	cur.MovePosition3(qt.QTextCursor__Start, qt.QTextCursor__MoveAnchor, 1)
	nw.viewer.SetTextCursor(cur)

	for i := 0; i <= occurrence; i++ {
		if !nw.viewer.Find3(query, 0) { // 0 flags => case-insensitive
			return
		}
	}
	nw.viewer.SetFocus()
}

// findTreeItemByRelPath tries to locate a tree item by stored rel path.
func (nw *NoteWindowQt) findTreeItemByRelPath(rel string) *qt.QTreeWidgetItem {
	if nw.treeWidget == nil || rel == "" {
		return nil
	}
	role := int(qt.UserRole)

	getRel := func(it *qt.QTreeWidgetItem) string {
		v := it.Data(0, role)
		if v != nil && v.IsValid() {
			return v.ToString()
		}
		return ""
	}

	var walk func(*qt.QTreeWidgetItem) *qt.QTreeWidgetItem
	walk = func(p *qt.QTreeWidgetItem) *qt.QTreeWidgetItem {
		for i := 0; i < p.ChildCount(); i++ {
			ch := p.Child(i)
			if getRel(ch) == rel {
				return ch
			}
			if r := walk(ch); r != nil {
				return r
			}
		}
		return nil
	}

	for i := 0; i < nw.treeWidget.TopLevelItemCount(); i++ {
		top := nw.treeWidget.TopLevelItem(i)
		if getRel(top) == rel {
			return top
		}
		if r := walk(top); r != nil {
			return r
		}
	}
	return nil
}

// searchNotesQt prompts the user for a search query and starts the search.
func (nw *NoteWindowQt) searchNotesQt() {
	dialog := qt.NewQDialog(nil)
	dialog.SetWindowTitle("Find Notes")

	layout := qt.NewQVBoxLayout(nil)
	entry := qt.NewQLineEdit(nil)
	entry.SetPlaceholderText("type to search notes...")
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

	if dialog.Exec() == int(qt.QDialog__Accepted) && entry.Text() != "" {
		nw.startSearch(entry.Text())
	}
	dialog.Destroy()
}

func (nw *NoteWindowQt) doDeleteQt() {
	if nw.selectedUID == "" {
		return
	}
	rel := nw.selectedUID
	oldid := rel
	fname := strings.TrimSuffix(oldid, ".md")
	reply := qt.QMessageBox_Question4(nw.win.QWidget, "Delete", "Are you sure, you want to delete "+fname+" note ?", qt.QMessageBox__Yes, qt.QMessageBox__No)
	if reply == int(qt.QMessageBox__Yes) {
		if err := nw.service.DeleteNote(rel); err != nil {
			qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
			return
		}
		nw.selectedUID = ""
		nw.treeWidget.ClearSelection()
		nw.refreshTreeQt()
		if nw.service.Gist.GistID != "" {
			reply2 := qt.QMessageBox_Question4(nw.win.QWidget, "Delete", "Do you want to delete from gist also?", qt.QMessageBox__Yes, qt.QMessageBox__No)
			if reply2 == int(qt.QMessageBox__Yes) {
				err := nw.service.DeleteFromGist(oldid)
				if err != nil {
					qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
					return
				}
				qt.QMessageBox_Information(nw.win.QWidget, "Info", "Note have been deleted from gist successfully!")
			}
		}
	}
}

func (nw *NoteWindowQt) saveNoteQt() {
	if nw.current == nil {
		return
	}
	nw.current.Body = []byte(nw.editor.ToPlainText())
	nw.current.Meta.Sticky = nw.stickyCheck.IsChecked()
	if err := nw.service.Save(nw.current); err != nil {
		qt.QMessageBox_Critical(nw.win.QWidget, "Error", err.Error())
	}
	// refresh stickies
	key := filepath.Base(nw.service.NotesDir)
	if val, ok := Stickies[key]; ok {
		val.Refresh()
	}
}

func (nw *NoteWindowQt) toggleViewQt() {
	if nw.current == nil {
		return
	}
	if nw.viewMode {
		nw.editor.SetVisible(true)
		nw.viewContainer.SetVisible(false)
		nw.viewMode = false
	} else {
		nw.saveNoteQt()
		nw.viewer.SetMarkdown(string(nw.current.Body))
		nw.updateHeaderQt()
		nw.editor.SetVisible(false)
		nw.viewContainer.SetVisible(true)
		nw.viewMode = true
	}
}

func (nw *NoteWindowQt) refreshTreeQt() {
	td, ib, _ := nw.service.ListTree()
	nw.treeData, nw.isBranch = td, ib
	populateNotesTreeQt(nw.treeWidget, td, ib)
}

func (nw *NoteWindowQt) updateHeaderQt() {
	if nw.current == nil {
		return
	}
	createdTime := ToLocalTime(nw.current.Meta.Created)
	updatedTime := ToLocalTime(nw.current.Meta.Updated)
	nw.createdLbl.SetText(createdTime.Format("2006-01-02 15:04:05"))
	nw.modifiedLbl.SetText(updatedTime.Format("2006-01-02 15:04:05"))
	nw.revisionsLbl.SetText(strconv.Itoa(len(nw.current.History)))
	nw.stickyCheck.SetChecked(nw.current.Meta.Sticky)
}
func (nw *NoteWindowQt) pushSyncQt() {
	if err := nw.service.PushSync(); err != nil {
		qt.QMessageBox_Critical(nw.win.QWidget, "Error", "Error pushing notes to gist: "+err.Error())
	} else {
		qt.QMessageBox_Information(nw.win.QWidget, "Info", "Notes pushed to gist successfully!")
	}
}

func (nw *NoteWindowQt) pullSyncQt() {
	if err := nw.service.PullSync(); err != nil {
		qt.QMessageBox_Critical(nw.win.QWidget, "Error", "Error pulling notes from gist: "+err.Error())
	} else {
		nw.refreshTreeQt()
		qt.QMessageBox_Information(nw.win.QWidget, "Info", "Notes pulled from gist successfully!")
	}
}
