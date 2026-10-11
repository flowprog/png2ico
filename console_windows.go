package main

import "syscall"

// Plain go build uses the console subsystem. Detach before opening the GUI;
// parent terminals still used by other processes are not hidden or closed.
func detachGUIConsole() {
	syscall.NewLazyDLL("kernel32.dll").NewProc("FreeConsole").Call()
}
