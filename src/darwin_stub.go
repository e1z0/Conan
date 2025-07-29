//go:build !darwin
// +build !darwin

// this file will not be used in non Darwin OS operating systems it's dummy file just to declare the function

package main

import "syscall"

func Ignore(sigNum syscall.Signal) {
	return
}

func IgnoreSignum() {

}

func darwin_bindkey() {
	// dummy version
}
