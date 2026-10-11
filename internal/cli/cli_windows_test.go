package cli

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIArguments(t *testing.T) {
	for _, args := range [][]string{
		{"-f", "-o", "输出 图片.ico", "输入 图片.png"},
		{"输入 图片.png", "--output=输出 图片.ico", "--force"},
		{"--force", "输入 图片.png", "输出 图片.ico"},
	} {
		opts, err := parseCLI(args)
		if err != nil || opts.source != "输入 图片.png" || opts.output != "输出 图片.ico" || !opts.force {
			t.Fatalf("args %q: %+v %v", args, opts, err)
		}
	}
	opts, err := parseCLI([]string{"--", "-图片.png"})
	if err != nil || opts.source != "-图片.png" {
		t.Fatalf("-- did not preserve filename: %+v %v", opts, err)
	}
	for _, args := range [][]string{
		nil, {""}, {"--unknown"}, {"-o"}, {"a.png", "b.ico", "c.ico"},
		{"-o", "x.ico", "a.png", "b.ico"}, {"a.png", "--output="},
		{"a.png", "-o", "x.ico", "-o", "y.ico"},
	} {
		var out, errs bytes.Buffer
		if code := Run(args, &out, &errs); code != 2 || out.Len() != 0 || errs.Len() == 0 {
			t.Fatalf("args %q: code=%d stdout=%q stderr=%q", args, code, out.String(), errs.String())
		}
	}
	var out, errs bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "--output") || errs.Len() != 0 {
		t.Fatalf("help: code=%d stdout=%q stderr=%q", code, out.String(), errs.String())
	}
}

func TestCLIConversionAndOverwrite(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "命令行 图片.png")
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	code := Run([]string{source}, &out, &errs)
	output := filepath.Join(dir, "命令行 图片.ico")
	if code != 0 || strings.TrimSpace(out.String()) != output || errs.Len() != 0 {
		t.Fatalf("default conversion: code=%d stdout=%q stderr=%q", code, out.String(), errs.String())
	}
	original, err := os.ReadFile(output)
	if err != nil || len(original) < 6 {
		t.Fatalf("missing ICO: %v", err)
	}
	out.Reset()
	errs.Reset()
	if code := Run([]string{source}, &out, &errs); code != 1 || !strings.Contains(errs.String(), "--force") || out.Len() != 0 {
		t.Fatalf("overwrite protection: code=%d stdout=%q stderr=%q", code, out.String(), errs.String())
	}
	after, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatalf("overwrite protection damaged file: %v", err)
	}
	out.Reset()
	errs.Reset()
	if code := Run([]string{source, "--force"}, &out, &errs); code != 0 {
		t.Fatalf("force conversion: %d %q", code, errs.String())
	}
	out.Reset()
	errs.Reset()
	custom := filepath.Join(t.TempDir(), "自定义.ico")
	if code := Run([]string{source, "-o", custom}, &out, &errs); code != 0 || strings.TrimSpace(out.String()) != custom {
		t.Fatalf("custom conversion: %d %q %q", code, out.String(), errs.String())
	}
	out.Reset()
	errs.Reset()
	if code := Run([]string{filepath.Join(dir, "missing.png")}, &out, &errs); code != 1 || errs.Len() == 0 || out.Len() != 0 {
		t.Fatalf("missing input: %d %q %q", code, out.String(), errs.String())
	}
}
