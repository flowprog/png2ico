package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	detachGUIConsole()
	mygo.App.SetName("PNG to ICO")
	a := &app{status: "Choose a PNG, confirm the output path, then click Convert."}
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "PNG to ICO", Width: 640, Height: 420,
			MinWidth: 480, MinHeight: 380, UseContentSize: true,
			Page:    mygo.PageOptions{DevTools: mygo.DevToolsDisabled},
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
