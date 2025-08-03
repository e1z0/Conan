package main

import (
	"log"
	"strings"
	"time"

	"github.com/mappu/miqt/qt"
)

type StickyWindowQt struct {
	Win    *qt.QDialog      // The sticky note window
	Scroll *qt.QScrollArea  // Scrollable markdown area
	Label  *qt.QTextBrowser // For displaying markdown
	Card   *qt.QWidget      // Holds the card widget for custom style
}

type StickyManagerQt struct {
	service *NoteService
	windows map[string]*StickyWindowQt
	parent  *qt.QWidget // For parenting new dialogs (can be nil or main window)
}

// NewStickyManagerQt constructs a manager with its own service
func NewStickyManagerQt(service *NoteService, parent *qt.QWidget) *StickyManagerQt {
	return &StickyManagerQt{
		service: service,
		parent:  parent,
		windows: make(map[string]*StickyWindowQt),
	}
}

// Refresh scans for sticky notes, opens, updates, or closes windows
func (sm *StickyManagerQt) Refresh() {
	treeData, _, err := sm.service.ListTree()
	if err != nil {
		log.Printf("StickyManagerQt.Refresh: %v", err)
		return
	}
	seen := make(map[string]bool)

	for _, children := range treeData {
		for _, rel := range children {
			if !strings.HasSuffix(rel, ".md") {
				continue
			}
			note, err := sm.service.Load(rel)
			if err != nil || !note.Meta.Sticky {
				continue
			}
			seen[rel] = true

			// render markdown (use QTextBrowser for markdown)
			label := qt.NewQTextBrowser(nil)
			label.SetMarkdown(string(note.Body))
			label.SetReadOnly(true)
			label.SetLineWrapMode(qt.QTextEdit__WidgetWidth)
			label.SetOpenLinks(false)
			label.SetOpenExternalLinks(false)

			label.OnAnchorClicked(func(url *qt.QUrl) {
				qt.QDesktopServices_OpenUrl(url)
			})

			if sw, ok := sm.windows[rel]; ok {
				log.Printf("refreshing existing %s", rel)
				sw.Label.SetMarkdown(string(note.Body))
				sw.Win.SetWindowTitle("📌 " + note.Meta.Title)
				sw.Scroll.SetWidget(sw.Label.QWidget)
			} else {
				log.Printf("creating new sticky window for %s", rel)

				win := qt.NewQDialog(sm.parent)
				win.SetWindowTitle("📌 " + note.Meta.Title)
				win.Resize(320, 220)

				// true window round corners implementation
				// Must set mask every time after resize
				radius := 18.0
				if env.os == "windows" {
					radius = 20.0
				}
				setRoundedMask := func() {
					r := win.Rect()
					w, h := r.Width(), r.Height()
					pixmap := qt.NewQPixmap2(w, h)
					pixmap.Fill() // Fills with default (opaque black, but we cover it up)

					painter := qt.NewQPainter()
					painter.Begin(pixmap.QPaintDevice) // Paint ON pixmap
					painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
					color := qt.NewQColor3(0, 0, 0) // Black, opaque
					brush := qt.NewQBrush3(color)
					painter.SetBrush(brush)

					rf := qt.NewQRectF5(qt.NewQRect4(0, 0, w, h))
					path := qt.NewQPainterPath()
					path.AddRoundedRect(rf, radius, radius)
					painter.DrawPath(path)
					painter.End() // End painting

					// Clean up painter/path not needed in miqt (no DestroyQPainter or DestroyQPainterPath)

					white := qt.NewQColor3(255, 255, 255)
					mask := pixmap.CreateMaskFromColor(white)
					win.SetMask(mask)
					//}
				}
				setRoundedMask()

				// save settings
				saveStickyWindowGeometry := func() {
					secName := "Sticky " + rel

					Store.SetMany(secName, map[string]interface{}{
						"x":      win.Pos().X(),
						"y":      win.Pos().Y(),
						"width":  win.Size().Width(),
						"height": win.Size().Height(),
					})
				}

				// load settings
				load := func() {
					secName := "Sticky " + rel
					if Store.HasSection(secName) {
						x, _ := Store.GetInt(secName, "x")
						y, _ := Store.GetInt(secName, "y")
						w, _ := Store.GetInt(secName, "width")
						h, _ := Store.GetInt(secName, "height")

						// avoid invalid sizes, this can cause a crash
						if h > 10000 {
							return
						}
						if w > 10000 {
							return
						}

						// Use defaults if not present
						if w > 0 && h > 0 {
							win.Resize(w, h)
						}
						if x > 0 && y > 0 {
							win.Move(x, y)
						}
					}
				}

				load()

				win.OnMouseReleaseEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					//resizing = false
					super(event)
					saveStickyWindowGeometry() // Only save when user lets go of the handle
				})
				var resizeTimer *time.Timer

				// Save on resize
				win.OnResizeEvent(func(super func(event *qt.QResizeEvent), event *qt.QResizeEvent) {
					setRoundedMask()
					if resizeTimer != nil {
						resizeTimer.Stop()
					}
					resizeTimer = time.AfterFunc(1000*time.Millisecond, func() {
						saveStickyWindowGeometry()
					})
				})

				if settings.NotesSettings.AlwaysOnTop {
					log.Printf("Note stickies always on top\n")
					win.SetWindowFlags(
						qt.FramelessWindowHint |
							qt.Window |
							qt.WindowStaysOnTopHint, // stay on top of other windows
					)
				} else {
					log.Printf("Note stickies normal mode\n")
					win.SetWindowFlags(
						qt.FramelessWindowHint |
							qt.Window, // <-- makes it resizable!
					)
				}

				win.SetMinimumSize2(320, 220) // (optional) minimum size

				// Shadow effect: Upcast to QGraphicsEffect
				effect := qt.NewQGraphicsDropShadowEffect()
				effect.SetBlurRadius(28)
				effect.SetColor(qt.NewQColor11(255, 205, 40, 60)) // soft, yellow shadow
				effect.SetOffset2(0, 5)
				win.SetGraphicsEffect(effect.QGraphicsEffect)

				// Card widget
				card := qt.NewQWidget(win.QWidget)
				card.SetObjectName("stickyCard")
				if env.os == "windows" {
					card.SetStyleSheet(`
        QWidget#stickyCard {
                background: #fff98a; /* STICKY YELLOW */
                border-radius: 18px;
        border: 1px solid #eedc82;
        box-shadow: 2 4px 18px 2px #c1bc8430; /* not always supported, but keep for Mac/Linux */
        }
`)
				} else {
					card.SetStyleSheet(`
	QWidget#stickyCard {
		background: #fff98a; /* STICKY YELLOW */
		border-radius: 18px;
		border: 1px solid #ffea94;
	}
`)
				}
				// Title and close button
				titleLabel := qt.NewQLabel5(note.Meta.Title, nil)
				titleLabel.SetStyleSheet("font-weight: bold; font-size: 15px; color: #c5a300;")
				titleLabel.SetAlignment(qt.AlignLeft | qt.AlignVCenter)

				closeBtn := qt.NewQPushButton3("×")
				closeBtn.SetStyleSheet(`
	QPushButton {
		background: #ffe066;
		border-radius: 12px;
		min-width: 28px;
		min-height: 28px;
		font-size: 20px;
		border: 1px solid #ffe066;
	}
	QPushButton:hover {
		background: #fff3c4;
	}
`)
				closeBtn.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
				closeBtn.OnClicked(func() {
					win.Close()
					sm.service.DisableSticky(note)
					//note.Meta.Sticky = false
					//sm.service.Save(note)
				})

				header := qt.NewQHBoxLayout2()
				header.AddWidget(titleLabel.QWidget)
				header.AddStretch()
				header.AddWidget(closeBtn.QWidget)

				// Label (QTextBrowser)
				label := qt.NewQTextBrowser(nil)
				label.SetMarkdown(string(note.Body))
				label.SetReadOnly(true)

				label.SetOpenLinks(false)
				label.SetOpenExternalLinks(false)

				label.OnAnchorClicked(func(url *qt.QUrl) {
					qt.QDesktopServices_OpenUrl(url)
				})

				label.SetLineWrapMode(qt.QTextEdit__WidgetWidth)
				label.SetStyleSheet(`
	QTextBrowser {
		background: #fff98a;  /* same as card */
		border: none;
		font-size: 16px;
		padding: 10px 8px 8px 8px;
		color: #222;
	}
`)

				// Scroll area for label
				scroll := qt.NewQScrollArea(nil)
				scroll.SetWidgetResizable(true)
				scroll.SetFrameShape(qt.QFrame__NoFrame)
				scroll.SetHorizontalScrollBarPolicy(qt.ScrollBarAlwaysOff)
				scroll.SetVerticalScrollBarPolicy(qt.ScrollBarAsNeeded)
				scroll.SetWidget(label.QWidget)

				cardLayout := qt.NewQVBoxLayout2()
				cardLayout.SetContentsMargins(12, 10, 12, 10)
				cardLayout.SetSpacing(6)
				cardLayout.AddLayout(header.QLayout)
				cardLayout.AddWidget(scroll.QWidget)

				// --- Add resize handle ---
				resizeHandle := qt.NewQLabel5("⤡", nil)

				resizeHandle.SetAlignment(qt.AlignRight | qt.AlignBottom)
				resizeHandle.SetStyleSheet(`
    QLabel {
        color: #d7b81d;
        font-size: 20px;
        padding: 4px 8px 2px 2px;
        background: transparent;
        cursor: se-resize;
    }
`)
				resizeHandle.SetCursor(qt.NewQCursor2(qt.SizeFDiagCursor))
				cardLayout.AddWidget3(resizeHandle.QWidget, 0, qt.AlignRight|qt.AlignBottom)

				var resizing bool
				var resizeStartPos qt.QPoint
				var resizeStartSize qt.QSize

				resizeHandle.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					if event.Button() == qt.LeftButton {
						resizing = true
						resizeStartPos = *event.GlobalPos()
						resizeStartSize = *win.Size()
					}
				})
				resizeHandle.OnMouseMoveEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					if resizing && (event.Buttons()&qt.LeftButton != 0) {
						gp := event.GlobalPos()
						dx := gp.X() - resizeStartPos.X()
						dy := gp.Y() - resizeStartPos.Y()
						newWidth := resizeStartSize.Width() + dx
						newHeight := resizeStartSize.Height() + dy
						if newWidth < 220 {
							newWidth = 220
						}
						if newHeight < 120 {
							newHeight = 120
						}
						win.Resize(newWidth, newHeight)
					}
				})
				resizeHandle.OnMouseReleaseEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					resizing = false
				})

				// Card layout to card
				card.SetLayout(cardLayout.QLayout)

				// Center card in dialog
				mainLayout := qt.NewQVBoxLayout2()
				mainLayout.AddWidget(card)
				mainLayout.SetContentsMargins(0, 0, 0, 0)
				mainLayout.SetSpacing(0)
				win.SetLayout(mainLayout.QLayout)

				// Drag window by card (as before)
				var dragPos qt.QPoint
				card.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					gp := event.GlobalPos()
					winPos := win.Pos()
					dragPos = *qt.NewQPoint2(gp.X()-winPos.X(), gp.Y()-winPos.Y())
				})
				card.OnMouseMoveEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
					if event.Buttons()&qt.LeftButton != 0 {
						gp := event.GlobalPos()
						win.Move(gp.X()-dragPos.X(), gp.Y()-dragPos.Y())
					}
				})

				win.Show()

				sm.windows[rel] = &StickyWindowQt{
					Win:    win,
					Card:   card,
					Label:  label,
					Scroll: scroll,
				}
			}
		}
	}

	// close non-sticky windows
	for rel, sw := range sm.windows {
		if !seen[rel] {
			log.Printf("closing sticky window for %s", rel)
			sw.Win.Close()
			delete(sm.windows, rel)
		}
	}
}
