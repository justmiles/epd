package mdpng_test

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	mdpng "github.com/justmiles/epd/lib/md-png"
)

func TestConvert_BasicMarkdown(t *testing.T) {
	md := []byte("# Hello World\n\nThis is a paragraph.\n")
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty PNG output")
	}
	// Verify it's a valid PNG.
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
	if img.Bounds().Dx() != 800 {
		t.Errorf("Expected width 800, got %d", img.Bounds().Dx())
	}
}

func TestConvert_EmptyInput(t *testing.T) {
	var buf bytes.Buffer
	if err := mdpng.Convert([]byte(""), &buf); err != nil {
		t.Fatalf("Convert with empty input failed: %v", err)
	}
	// Should still produce a valid (minimal) PNG.
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty PNG output even for empty input")
	}
}

func TestConvert_WithWidth(t *testing.T) {
	md := []byte("# Test\n\nSome text.\n")
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf, mdpng.WithWidth(400)); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
	if img.Bounds().Dx() != 400 {
		t.Errorf("Expected width 400, got %d", img.Bounds().Dx())
	}
}

func TestConvert_DarkMode(t *testing.T) {
	md := []byte("# Dark\n\nSome text.\n")
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf, mdpng.WithDarkMode(true)); err != nil {
		t.Fatalf("Convert with dark mode failed: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
	// Check that the top-left pixel is dark (background).
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 > 50 || g>>8 > 50 || b>>8 > 50 {
		t.Errorf("Expected dark background, got RGB(%d, %d, %d)", r>>8, g>>8, b>>8)
	}
}

func TestConvert_LightMode(t *testing.T) {
	md := []byte("# Light\n\nSome text.\n")
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf, mdpng.WithDarkMode(false)); err != nil {
		t.Fatalf("Convert with light mode failed: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
	// Check that the top-left pixel is light (background).
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 < 200 || g>>8 < 200 || b>>8 < 200 {
		t.Errorf("Expected light background, got RGB(%d, %d, %d)", r>>8, g>>8, b>>8)
	}
}

func TestConvert_AllElements(t *testing.T) {
	md := []byte(`# Heading 1
## Heading 2
### Heading 3

This is **bold** and *italic* text.

- Unordered item 1
- Unordered item 2

1. Ordered item 1
2. Ordered item 2

> A blockquote

---

` + "```\ncode block\n```" + `

[A link](https://example.com)

~~strikethrough~~
`)
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf); err != nil {
		t.Fatalf("Convert with all elements failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty PNG output")
	}
	_, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
}

func TestConvert_Options(t *testing.T) {
	md := []byte("# Test\n\nParagraph.\n")

	tests := []struct {
		name string
		opts []mdpng.Option
	}{
		{"defaults", nil},
		{"custom width", []mdpng.Option{mdpng.WithWidth(1024)}},
		{"custom padding", []mdpng.Option{mdpng.WithPadding(40)}},
		{"custom font size", []mdpng.Option{mdpng.WithFontSize(18)}},
		{"all options", []mdpng.Option{
			mdpng.WithWidth(600),
			mdpng.WithPadding(30),
			mdpng.WithFontSize(16),
			mdpng.WithDarkMode(true),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := mdpng.Convert(md, &buf, tt.opts...); err != nil {
				t.Fatalf("Convert failed: %v", err)
			}
			if buf.Len() == 0 {
				t.Fatal("Expected non-empty PNG output")
			}
		})
	}
}

func TestConvert_LongContent(t *testing.T) {
	var md bytes.Buffer
	md.WriteString("# Long Document\n\n")
	for i := 0; i < 50; i++ {
		md.WriteString("This is a paragraph with some text to ensure the renderer handles long documents gracefully. ")
		md.WriteString("It should word-wrap properly and not crash.\n\n")
	}

	var buf bytes.Buffer
	if err := mdpng.Convert(md.Bytes(), &buf); err != nil {
		t.Fatalf("Convert with long content failed: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
	// Image should be tall for long content.
	if img.Bounds().Dy() < 500 {
		t.Errorf("Expected tall image for long content, got height %d", img.Bounds().Dy())
	}
}

func TestConvert_NestedLists(t *testing.T) {
	md := []byte(`- Item 1
  - Nested 1a
  - Nested 1b
- Item 2
  - Nested 2a
`)
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf); err != nil {
		t.Fatalf("Convert with nested lists failed: %v", err)
	}
	_, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
}

func TestConvert_WriteToFile(t *testing.T) {
	md := []byte("# File Test\n\nWriting to a file.\n")
	outPath := filepath.Join(t.TempDir(), "output.png")

	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	if err := mdpng.Convert(md, f); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	f.Close()

	// Verify the file was written.
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("Output file is empty")
	}
}

func TestConvert_GFMTable(t *testing.T) {
	md := []byte(`# Table Test

| Name | Age | City |
|------|----:|:----:|
| Alice | 30 | NYC |
| Bob | 25 | LA |
| Charlie | 35 | Chicago |
`)
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf); err != nil {
		t.Fatalf("Convert with GFM table failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty PNG output")
	}
	_, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
}

func TestConvert_TaskListCheckboxes(t *testing.T) {
	md := []byte(`# Task List

- [x] Completed task
- [ ] Pending task
- [x] Another done item
- Regular bullet item
`)
	var buf bytes.Buffer
	if err := mdpng.Convert(md, &buf); err != nil {
		t.Fatalf("Convert with task list failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty PNG output")
	}
	_, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("Output is not a valid PNG: %v", err)
	}
}
