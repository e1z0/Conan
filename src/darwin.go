//go:build darwin
// +build darwin

package main

import (
	"log"
	"syscall"

	"golang.design/x/hotkey"
)

/*
#include <stdint.h>
#include <stdio.h>

#ifdef __cplusplus
#include <csignal>
#else
#include <signal.h>
#endif

void Ignore(int sigNum);

void Ignore(int sigNum) {
    struct sigaction sa;
    sa.sa_handler = SIG_DFL;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags |= SA_ONSTACK;
    sigaction(sigNum, &sa, NULL);
}

*/
import "C"

func Ignore(sigNum syscall.Signal) {
	C.Ignore(C.int(sigNum))
}

func IgnoreSignum() {
	Ignore(syscall.SIGURG)
}

func darwin_bindkey() {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeySpace)

	err := hk.Register()
	if err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}

	for range hk.Keydown() {
		//<-hk.Keydown()
		log.Printf("hotkey ctrl+space is triggered\n")
		uiCmdChan <- "show"
	}
}
