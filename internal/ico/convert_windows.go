// Package ico converts PNG images to Windows icons with fixed default sizes.
package ico

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/image/draw"
)

var ErrOutputExists = errors.New("an ICO file with the same name already exists")

var sizes = [...]int{16, 32, 48, 64, 128, 256}

// DefaultOutput returns a same-name ICO path beside the selected PNG.
func DefaultOutput(source string) string {
	source = cleanPath(source)
	if source == "" {
		return ""
	}
	return strings.TrimSuffix(source, filepath.Ext(source)) + ".ico"
}

func cleanPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "\"")
}

// Convert writes to output, or a sibling .ico when output is empty.
// Existing files require explicit overwrite.
// The PNG is never modified. A temporary file keeps failures from damaging
// an existing icon; MoveFileEx publishes it atomically on Windows.
func Convert(source, output string, overwrite bool) (string, error) {
	source, output = cleanPath(source), cleanPath(output)
	if !strings.EqualFold(filepath.Ext(source), ".png") {
		return "", errors.New("please select a .png file")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("resolve input path: %w", err)
	}
	if output == "" {
		output = DefaultOutput(source)
	}
	if !strings.EqualFold(filepath.Ext(output), ".ico") {
		return output, errors.New("the output file must use the .ico extension")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("resolve ICO path: %w", err)
	}
	if info, err := os.Lstat(output); err == nil {
		if !info.Mode().IsRegular() {
			return output, errors.New("the output path is already occupied by a directory or link")
		}
		if !overwrite {
			return output, ErrOutputExists
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return output, fmt.Errorf("check output path: %w", err)
	}

	img, err := readPNG(source)
	if err != nil {
		return output, err
	}
	data, err := encode(img)
	if err != nil {
		return output, fmt.Errorf("encode icon: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".png2ico-*.tmp")
	if err != nil {
		return output, fmt.Errorf("create output file: %w", err)
	}
	defer os.Remove(tmp.Name())
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return output, fmt.Errorf("write icon: %w", err)
	}
	if err := publish(tmp.Name(), output, overwrite); err != nil {
		if errors.Is(err, syscall.ERROR_ALREADY_EXISTS) || errors.Is(err, syscall.ERROR_FILE_EXISTS) {
			return output, ErrOutputExists
		}
		return output, fmt.Errorf("save icon: %w", err)
	}
	return output, nil
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read PNG: %w", err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("invalid PNG image: %w", err)
	}
	// Bound decoded memory before allocating it (up to 16-bit RGBA).
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 32*1024*1024 {
		return nil, errors.New("image too large; use a PNG with no more than 32 million pixels")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("read PNG: %w", err)
	}
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	return img, nil
}

func encode(src image.Image) ([]byte, error) {
	const directorySize = 6 + 16*len(sizes)
	data := make([]byte, directorySize)
	binary.LittleEndian.PutUint16(data[2:4], 1)
	binary.LittleEndian.PutUint16(data[4:6], uint16(len(sizes)))
	for i, size := range sizes {
		img := fit(src, size)
		var payload []byte
		if size == 256 {
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return nil, err
			}
			payload = buf.Bytes()
		} else {
			payload = bitmap(img)
		}
		entry := data[6+16*i : 6+16*(i+1)]
		entry[0], entry[1] = byte(size%256), byte(size%256)
		binary.LittleEndian.PutUint16(entry[4:6], 1)
		binary.LittleEndian.PutUint16(entry[6:8], 32)
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(payload)))
		binary.LittleEndian.PutUint32(entry[12:16], uint32(len(data)))
		data = append(data, payload...)
	}
	return data, nil
}

// Fit the whole image into a square, with transparent padding and no crop.
func fit(src image.Image, size int) *image.NRGBA {
	b := src.Bounds()
	scale := float64(size) / float64(max(b.Dx(), b.Dy()))
	w := max(1, int(math.Round(float64(b.Dx())*scale)))
	h := max(1, int(math.Round(float64(b.Dy())*scale)))
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	r := image.Rect((size-w)/2, (size-h)/2, (size-w)/2+w, (size-h)/2+h)
	draw.CatmullRom.Scale(dst, r, src, b, draw.Src, nil)
	return dst
}

// ICO's small frames are 32-bit bottom-up DIBs followed by a padded AND mask.
// The 256px frame uses PNG compression, as supported by modern Windows.
func bitmap(img *image.NRGBA) []byte {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	maskStride := ((w + 31) / 32) * 4
	pixelBytes := w * h * 4
	data := make([]byte, 40+pixelBytes+maskStride*h)
	binary.LittleEndian.PutUint32(data[0:4], 40)
	binary.LittleEndian.PutUint32(data[4:8], uint32(w))
	binary.LittleEndian.PutUint32(data[8:12], uint32(h*2))
	binary.LittleEndian.PutUint16(data[12:14], 1)
	binary.LittleEndian.PutUint16(data[14:16], 32)
	binary.LittleEndian.PutUint32(data[20:24], uint32(pixelBytes))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := color.NRGBAModel.Convert(img.At(x, h-1-y)).(color.NRGBA)
			offset := 40 + (y*w+x)*4
			data[offset], data[offset+1], data[offset+2], data[offset+3] = p.B, p.G, p.R, p.A
			if p.A == 0 {
				data[40+pixelBytes+y*maskStride+x/8] |= 0x80 >> uint(x%8)
			}
		}
	}
	return data
}

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func publish(from, to string, overwrite bool) error {
	src, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	var flags uintptr
	if overwrite {
		flags = 1 // MOVEFILE_REPLACE_EXISTING
	}
	ok, _, err := moveFileEx.Call(uintptr(unsafe.Pointer(src)), uintptr(unsafe.Pointer(dst)), flags)
	if ok == 0 {
		return err
	}
	return nil
}
