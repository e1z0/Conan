//go:build windows
// +build windows

package main

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	registerHotKey   = user32.NewProc("RegisterHotKey")
	unregisterHotKey = user32.NewProc("UnregisterHotKey")
	getMessageW      = user32.NewProc("GetMessageW")
)

const (
	MOD_CONTROL = 0x0002
	VK_SPACE    = 0x20
	WM_HOTKEY   = 0x0312
)

func win32_bindkey() {
	// Register Ctrl + Space as global hotkey with ID 1
	if r, _, err := registerHotKey.Call(0, 1, MOD_CONTROL, VK_SPACE); r == 0 {
		log.Println("win32 Failed to register hotkey:", err)
		return
	}
	defer unregisterHotKey.Call(0, 1)

	log.Println("Hotkey registered. Press Ctrl + Space...")

	var msg struct {
		hWnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}

	for {
		r, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			break // WM_QUIT
		}
		if msg.message == WM_HOTKEY {
			log.Println("Ctrl + Space pressed!")
			uiCmdChan <- "show"
		}
	}
}
