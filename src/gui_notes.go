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
}

func ShowNotesWindowQt(app *qt.QApplication, notesDir string, gist GistConfig) {
	key := filepath.Base(notesDir)
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
	deleteIcon := qt.NewQIcon4(":/icons/delete.png")
	saveIcon := qt.NewQIcon4(":/icons/save.png")
	viewModeIcon := qt.NewQIcon4(":/icons/show-hide.png")
	pushIcon := qt.NewQIcon4(":/icons/syncpush.png")
	pullIcon := qt.NewQIcon4(":/icons/syncpull.png")

	toolbar := qt.NewQHBoxLayout2()

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
	addToolBtn(deleteIcon, "Delete folder or note", func() { nw.doDeleteQt() })
	addToolBtn(saveIcon, "Save current note", func() { nw.saveNoteQt() })
	addToolBtn(viewModeIcon, "View mode editor or viewer", func() { nw.toggleViewQt() })
	addToolBtn(pushIcon, "Upload notes to the remote server (sync push)", func() { nw.pushSyncQt() })
	addToolBtn(pullIcon, "Download notes from the remote server (sync pull)", func() { nw.pullSyncQt() })

	splitter := qt.NewQSplitter3(qt.Horizontal)
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
