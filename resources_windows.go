package main

// Embed the supplied ICO as resource ID 1, which MyGo uses for Windows' title
// bar and taskbar icons. Go automatically links the generated .syso files.
//go:generate go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64,arm64 --icon assets/png2ico-icon.ico --manifest gui --out appicon
//go:generate go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64,arm64 --icon assets/png2ico-icon.ico --manifest cli --out cmd/png2ico-cli/appicon
