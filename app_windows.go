package main

import (
	"errors"
	"path/filepath"
	"strings"

	"png2ico/internal/ico"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// UI state is only read or changed on MyGo's main thread.
type app struct {
	window       *mygo.Window
	source       string
	output       string
	customOutput bool
	status       string
	busy         bool
	failed       bool
	succeeded    bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	root := ui.Scroll(c).Fill().Disabled(a.busy)
	if files := root.DroppedFiles(); len(files) > 0 {
		a.acceptDrop(files)
	}
	if root.FileDragOver() {
		root.Border(2, t.Accent)
	}
	root.Children(func() {
		ui.Column(c).FillWidth().Padding(28).Gap(18).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, "PNG → ICO").FontSize(28).Bold()
				ui.Text(c, "Enter a path, browse, or drop a PNG into this window.").TextColor(t.TextMuted)
			})
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, "PNG file").FontSize(12).TextColor(t.TextMuted)
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					input := ui.TextInput(c, &a.source).Label("PNG path").Placeholder("Enter or drop a PNG file path").Grow(1).MinWidth(0).Disabled(a.busy)
					if input.Changed() {
						a.sourceChanged()
					}
					if ui.Button(c, "Browse…").Label("Choose PNG").Disabled(a.busy).Clicked() {
						a.busy = true
						go a.choosePNG(a.source)
					}
				})
			})
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, "ICO output path").FontSize(12).TextColor(t.TextMuted)
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					input := ui.TextInput(c, &a.output).Label("ICO path").Placeholder("Defaults to the PNG's folder").Grow(1).MinWidth(0).Disabled(a.busy)
					if input.Changed() {
						a.outputChanged()
					}
					if ui.Button(c, "Browse…").Label("Choose ICO output path").Disabled(a.busy).Clicked() {
						a.busy = true
						go a.chooseICO(a.output)
					}
				})
			})
			label := "Convert"
			if a.busy {
				label = "Please wait…"
			}
			if ui.PrimaryButton(c, label).Disabled(a.busy || cleanPath(a.source) == "" || cleanPath(a.output) == "").FillWidth().PaddingY(12).Clicked() {
				a.busy = true
				a.status, a.failed, a.succeeded = "Converting…", false, false
				// Snapshot both paths before starting the worker.
				go a.convert(a.source, a.output)
			}
			color := t.TextMuted
			if a.failed {
				color = t.Danger
			} else if a.succeeded {
				color = t.Success
			}
			ui.Text(c, a.status).FillWidth().TextColor(color).Selectable()
		})
	})
}

func cleanPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "\"")
}

func (a *app) sourceChanged() {
	if !a.customOutput || cleanPath(a.output) == "" {
		a.output = ico.DefaultOutput(a.source)
		a.customOutput = false
	}
	a.pathsChanged()
}

func (a *app) outputChanged() {
	a.customOutput = cleanPath(a.output) != "" && !strings.EqualFold(cleanPath(a.output), ico.DefaultOutput(a.source))
	a.pathsChanged()
}

func (a *app) pathsChanged() {
	a.failed, a.succeeded = false, false
	a.status = "Confirm the output path, then click Convert."
}

func (a *app) setSource(path string) {
	a.source = cleanPath(path)
	a.sourceChanged()
}

func (a *app) acceptDrop(paths []string) {
	if a.busy {
		return
	}
	if len(paths) != 1 || !strings.EqualFold(filepath.Ext(paths[0]), ".png") {
		a.status, a.failed, a.succeeded = "Drop one PNG file at a time.", true, false
		return
	}
	a.setSource(paths[0])
}

func (a *app) choosePNG(previous string) {
	defaultPath := ""
	if cleanPath(previous) != "" {
		defaultPath = filepath.Dir(cleanPath(previous))
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: a.window, Title: "Choose a PNG file", ButtonLabel: "Select",
		DefaultPath: defaultPath,
		Filters:     []mygo.FileFilter{{Name: "PNG images", Extensions: []string{"png"}}},
	})
	a.window.Update(func() {
		a.busy = false
		if err != nil {
			a.status, a.failed, a.succeeded = "Unable to open the file picker: "+err.Error(), true, false
		} else if len(paths) > 0 {
			a.setSource(paths[0])
		}
	})
}

func (a *app) chooseICO(previous string) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: a.window, Title: "Choose an ICO output path", ButtonLabel: "Select path",
		DefaultPath: cleanPath(previous),
		Filters:     []mygo.FileFilter{{Name: "ICO icons", Extensions: []string{"ico"}}},
	})
	a.window.Update(func() {
		a.busy = false
		if err != nil {
			a.status, a.failed, a.succeeded = "Unable to open the output file picker: "+err.Error(), true, false
		} else if path != "" {
			if filepath.Ext(path) == "" {
				path += ".ico"
			}
			a.output = path
			a.outputChanged()
		}
	})
}

func (a *app) convert(source, destination string) {
	output, err := ico.Convert(source, destination, false)
	if errors.Is(err, ico.ErrOutputExists) {
		choice, dialogErr := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: a.window, Type: mygo.MessageQuestion, Title: "Confirm overwrite",
			Message: "The destination ICO already exists. Overwrite it?", Detail: output,
			Buttons: []string{"Cancel", "Overwrite"}, DefaultButton: 0, CancelButton: 0,
		})
		if dialogErr != nil {
			a.finish("Unable to confirm overwrite: "+dialogErr.Error(), "", true)
			return
		}
		if choice.Button != 1 {
			a.finish("Conversion canceled. The existing ICO was not changed.", "", false)
			return
		}
		output, err = ico.Convert(source, destination, true)
	}
	if err != nil {
		a.finish("Conversion failed: "+err.Error(), "", true)
		return
	}
	a.finish("Conversion complete. ICO saved.", output, false)
}

func (a *app) finish(status, output string, failed bool) {
	a.window.Update(func() {
		a.status, a.failed, a.busy = status, failed, false
		a.succeeded = output != "" && !failed
		if a.succeeded {
			a.output = output
		}
	})
}
