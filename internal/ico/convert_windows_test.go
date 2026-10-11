package ico

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

func makePNG(t *testing.T, path string) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 240, G: 70, B: 30, A: 128})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestConvertAndWindowsLoad(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "透明 图片.PNG")
	original := makePNG(t, source)
	output, err := Convert(source, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if output != filepath.Join(dir, "透明 图片.ico") {
		t.Fatalf("unexpected output: %s", output)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data[:6], []byte{0, 0, 1, 0, 6, 0}) {
		t.Fatalf("invalid ICO header: %x", data[:6])
	}
	for i, size := range []int{16, 32, 48, 64, 128, 256} {
		entry := data[6+i*16 : 6+(i+1)*16]
		if entry[0] != byte(size%256) || entry[1] != byte(size%256) || binary.LittleEndian.Uint16(entry[6:8]) != 32 {
			t.Fatalf("invalid directory entry for %d: %x", size, entry)
		}
		offset := int(binary.LittleEndian.Uint32(entry[12:16]))
		length := int(binary.LittleEndian.Uint32(entry[8:12]))
		if offset < 102 || length <= 0 || offset+length > len(data) {
			t.Fatalf("invalid payload bounds for %d", size)
		}
		frame := data[offset : offset+length]
		if size == 256 {
			img, err := png.Decode(bytes.NewReader(frame))
			if err != nil || img.Bounds() != image.Rect(0, 0, 256, 256) {
				t.Fatalf("invalid PNG frame: %v", err)
			}
			center := color.NRGBAModel.Convert(img.At(128, 128)).(color.NRGBA)
			_, _, _, paddingAlpha := img.At(128, 10).RGBA()
			if center.A != 128 || center.R < 235 || paddingAlpha != 0 {
				t.Fatalf("PNG transparency or aspect ratio lost: center=%v padding=%d", center, paddingAlpha)
			}
		} else {
			if binary.LittleEndian.Uint32(frame[:4]) != 40 || binary.LittleEndian.Uint32(frame[8:12]) != uint32(size*2) {
				t.Fatalf("invalid DIB for %d", size)
			}
			center := 40 + ((size/2)*size+size/2)*4
			if frame[center+2] < 235 || frame[center+3] != 128 || frame[43] != 0 {
				t.Fatalf("DIB transparency or aspect ratio lost for %d", size)
			}
			mask := frame[40+size*size*4:]
			if mask[0] != 255 {
				t.Fatalf("transparent padding has no AND mask for %d", size)
			}
		}
		// Let Windows itself decode every generated size, including PNG 256.
		path, err := syscall.UTF16PtrFromString(output)
		if err != nil {
			t.Fatal(err)
		}
		user32 := syscall.NewLazyDLL("user32.dll")
		handle, _, loadErr := user32.NewProc("LoadImageW").Call(0, uintptr(unsafe.Pointer(path)), 1, uintptr(size), uintptr(size), 0x10)
		if handle == 0 {
			t.Fatalf("Windows cannot load %dpx ICO: %v", size, loadErr)
		}
		user32.NewProc("DestroyIcon").Call(handle)
	}
	after, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatalf("source was modified: %v", err)
	}
}

func TestExistingOutputAndFailedConversion(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "test.png")
	makePNG(t, source)
	output := filepath.Join(dir, "test.ico")
	original := []byte("existing icon")
	if err := os.WriteFile(output, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Convert(source, "", false); !errors.Is(err, ErrOutputExists) {
		t.Fatalf("expected overwrite protection: %v", err)
	}
	if err := os.WriteFile(source, []byte("not a PNG"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Convert(source, "", true); err == nil {
		t.Fatal("accepted an invalid PNG")
	}
	data, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("existing output was damaged: %v", err)
	}
	makePNG(t, source)
	if _, err := Convert(source, "", true); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(output)
	if err != nil || bytes.Equal(data, original) {
		t.Fatalf("explicit overwrite failed: %v", err)
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".png2ico-*.tmp"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("temporary files left over: %v %v", temps, err)
	}
}

func TestInvalidInput(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"missing.png", "other.jpg", "broken.png"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			if name == "broken.png" {
				// A valid header with a truncated image must also be rejected.
				data := makePNG(t, path)
				if err := os.WriteFile(path, data[:len(data)/2], 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Convert(path, "", false); err == nil {
				t.Fatal("accepted invalid input")
			}
			output := filepath.Join(dir, name[:len(name)-len(filepath.Ext(name))]+".ico")
			if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("created an output after failure: %v", err)
			}
		})
	}
}

func TestPublishDoesNotReplaceWithoutPermission(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "temp"), filepath.Join(dir, "output.ico")
	for path, contents := range map[string]string{from: "new", to: "old"} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := publish(from, to, false); err == nil {
		t.Fatal("replaced destination without permission")
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "old" {
		t.Fatalf("destination was damaged: %s %v", data, err)
	}
}

func TestCustomOutputAndQuotedPaths(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "源 图片.png")
	original := makePNG(t, source)
	output := filepath.Join(t.TempDir(), "自定义名字.ICO")
	got, err := Convert("\""+source+"\"", " \""+output+"\" ", false)
	if err != nil || got != output {
		t.Fatalf("custom destination failed: %s %v", got, err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(DefaultOutput(source)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected sibling ICO: %v", err)
	}
	if _, err := Convert(source, output, false); !errors.Is(err, ErrOutputExists) {
		t.Fatalf("custom output lost overwrite protection: %v", err)
	}
	if _, err := Convert(source, source, true); err == nil {
		t.Fatal("accepted PNG source as output")
	}
	if _, err := Convert(source, filepath.Join(dir, "missing", "icon.ico"), false); err == nil {
		t.Fatal("accepted nonexistent destination directory")
	}
	after, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatalf("source was changed: %v", err)
	}
}
