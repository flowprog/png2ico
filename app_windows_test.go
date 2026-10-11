package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestManualPathsDoNotStartConversion(t *testing.T) {
	a := &app{}
	tt := ui.NewTester(a.view, 640, 420)
	source := filepath.Join(t.TempDir(), "图片.png")
	if err := tt.Click("PNG path"); err != nil {
		t.Fatal(err)
	}
	tt.Type(source)
	want := filepath.Join(filepath.Dir(source), "图片.ico")
	if a.source != source || a.output != want || a.busy || a.succeeded {
		t.Fatalf("typing should fill paths without converting: %+v", a)
	}
	if _, err := os.Stat(want); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("input created output: %v", err)
	}
	custom := filepath.Join(t.TempDir(), "自定义.ico")
	if err := tt.Click("ICO path"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(custom)
	if a.output != custom || !a.customOutput {
		t.Fatalf("custom output was not accepted: %+v", a)
	}
	if err := tt.Click("PNG path"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(filepath.Join(filepath.Dir(source), "另一张.png"))
	if a.output != custom || a.busy {
		t.Fatalf("changing PNG should preserve custom destination: %+v", a)
	}
}

func TestDroppedPathsAndBusyProtection(t *testing.T) {
	a := &app{}
	a.acceptDrop([]string{`D:\图片\第一张.PNG`})
	if a.source != `D:\图片\第一张.PNG` || a.output != `D:\图片\第一张.ico` || a.busy || a.failed {
		t.Fatalf("drop should fill paths only: %+v", a)
	}
	a.acceptDrop([]string{`D:\图片\第二张.png`})
	if a.output != `D:\图片\第二张.ico` {
		t.Fatalf("default output did not follow source: %s", a.output)
	}
	a.acceptDrop([]string{`D:\文件.txt`})
	if !a.failed || a.source != `D:\图片\第二张.png` {
		t.Fatalf("invalid drop replaced selection: %+v", a)
	}
	a.acceptDrop([]string{`D:\a.png`, `D:\b.png`})
	if !a.failed || a.source != `D:\图片\第二张.png` {
		t.Fatalf("multiple drop replaced selection: %+v", a)
	}
	a.busy = true
	a.acceptDrop([]string{`D:\图片\第三张.png`})
	if a.source != `D:\图片\第二张.png` {
		t.Fatal("drop changed a running conversion's source")
	}
	// The conversion action must not run again while busy (nil window would
	// expose an unintended worker if the disabled button were clickable).
	tt := ui.NewTester(a.view, 640, 420)
	if err := tt.Click("Please wait…"); err != nil {
		t.Fatal(err)
	}
	if !a.busy || a.status == "Converting…" {
		t.Fatal("busy button started another conversion")
	}
}
