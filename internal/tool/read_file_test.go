package tool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile_TextFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "hello.txt")
	if err := os.WriteFile(p, []byte("hello text"), 0o644); err != nil {
		t.Fatal(err)
	}
	argsJSON := `{"path":"` + p + `"}`
	result, err := ReadFile(argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsMultimodal {
		t.Error("text file should not be multimodal")
	}
	if result.TextContent != "hello text" {
		t.Errorf("expected 'hello text', got %q", result.TextContent)
	}
}

func TestReadFile_MissingFile(t *testing.T) {
	argsJSON := `{"path":"/tmp/nonexistent_xmd_test_file.txt"}`
	_, err := ReadFile(argsJSON)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestReadFile_EmptyPath(t *testing.T) {
	_, err := ReadFile(`{"path":""}`)
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestReadFile_InvalidJSON(t *testing.T) {
	_, err := ReadFile(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestReadFile_PNG(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "img.png")
	if err := os.WriteFile(p, []byte("fakeimagedata"), 0o644); err != nil {
		t.Fatal(err)
	}
	argsJSON := `{"path":"` + p + `"}`
	result, err := ReadFile(argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsMultimodal {
		t.Error("PNG should be multimodal")
	}
	if result.MediaCategory != "image" {
		t.Errorf("expected MediaCategory='image', got %q", result.MediaCategory)
	}
	if result.MIMEType != "image/png" {
		t.Errorf("expected MIMEType='image/png', got %q", result.MIMEType)
	}
}

func TestReadFile_JPG(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "photo.jpg")
	if err := os.WriteFile(p, []byte("fakejpegdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ReadFile(`{"path":"` + p + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MIMEType != "image/jpeg" {
		t.Errorf("expected 'image/jpeg', got %q", result.MIMEType)
	}
}

func TestReadFile_MP3(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "audio.mp3")
	if err := os.WriteFile(p, []byte("fakemp3data"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ReadFile(`{"path":"` + p + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsMultimodal {
		t.Error("MP3 should be multimodal")
	}
	if result.MediaCategory != "audio" {
		t.Errorf("expected MediaCategory='audio', got %q", result.MediaCategory)
	}
	if result.MIMEType != "audio/mpeg" {
		t.Errorf("expected MIMEType='audio/mpeg', got %q", result.MIMEType)
	}
}

func TestReadFile_WAV(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "sound.wav")
	if err := os.WriteFile(p, []byte("fakewavdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ReadFile(`{"path":"` + p + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MIMEType != "audio/wav" {
		t.Errorf("expected MIMEType='audio/wav', got %q", result.MIMEType)
	}
}

func TestReadFile_ImageReadError(t *testing.T) {
	// Path has .png extension but file does not exist → error in os.ReadFile for image branch
	tmp := t.TempDir()
	p := filepath.Join(tmp, "nonexistent.png")
	_, err := ReadFile(`{"path":"` + p + `"}`)
	if err == nil {
		t.Error("expected error for missing image file, got nil")
	}
}

func TestReadFile_AudioReadError(t *testing.T) {
	// Path has .mp3 extension but file does not exist → error in os.ReadFile for audio branch
	tmp := t.TempDir()
	p := filepath.Join(tmp, "nonexistent.mp3")
	_, err := ReadFile(`{"path":"` + p + `"}`)
	if err == nil {
		t.Error("expected error for missing audio file, got nil")
	}
}
