package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// solidImage returns a small test image.
func solidImage(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.NRGBA{R: 70, G: 95, B: 255, A: 255})
		}
	}
	return img
}

// pngBytes encodes an image as PNG.
func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestDecodePNGAcceptsPNG checks a normal PNG is decoded.
func TestDecodePNGAcceptsPNG(t *testing.T) {
	data := pngBytes(t, solidImage(32))

	img, err := DecodePNG(data)
	if err != nil {
		t.Fatalf("DecodePNG() error = %v", err)
	}
	if got := img.Bounds().Size(); got.X != 32 || got.Y != 32 {
		t.Errorf("size = %v, want 32x32", got)
	}
}

// TestDecodePNGAcceptsJPEG checks the JPEG answer of the image service is
// accepted too: the service sometimes sends a JPEG behind a .png URL.
func TestDecodePNGAcceptsJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, solidImage(32), nil); err != nil {
		t.Fatal(err)
	}

	img, err := DecodePNG(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodePNG() error = %v", err)
	}
	if got := img.Bounds().Size(); got.X != 32 || got.Y != 32 {
		t.Errorf("size = %v, want 32x32", got)
	}
}

// TestDecodePNGRejectsGarbage checks a body that is not an image is refused.
func TestDecodePNGRejectsGarbage(t *testing.T) {
	if _, err := DecodePNG([]byte("this is not an image at all")); err == nil {
		t.Fatal("DecodePNG() expected error for a body that is not an image")
	}
}

// TestResizeProducesSquare checks the picture is scaled to the wanted edge.
func TestResizeProducesSquare(t *testing.T) {
	resized := Resize(solidImage(512), 64)

	got := resized.Bounds().Size()
	if got.X != 64 || got.Y != 64 {
		t.Errorf("size = %v, want 64x64", got)
	}
}

// TestResizeKeepsOriginalForZeroSize checks a zero size means "leave it".
func TestResizeKeepsOriginalForZeroSize(t *testing.T) {
	src := solidImage(48)

	if got := Resize(src, 0); got.Bounds().Size().X != 48 {
		t.Errorf("size = %v, want the original 48", got.Bounds().Size())
	}
}

// TestSavePNGAndReadDataURI checks an icon survives a round trip through disk
// and comes back as a usable data URI.
func TestSavePNGAndReadDataURI(t *testing.T) {
	dir := t.TempDir()
	name := "sub-1.png"

	if err := SavePNG(dir, name, Resize(solidImage(256), 64)); err != nil {
		t.Fatalf("SavePNG() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("icon file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Error("icon file is empty")
	}

	dataURI, err := ReadDataURI(dir, name)
	if err != nil {
		t.Fatalf("ReadDataURI() error = %v", err)
	}
	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Errorf("data URI = %q, want a PNG data URI", dataURI[:32])
	}
}

// TestSavePNGLeavesNoTempFiles checks the atomic write cleans up after
// itself.
func TestSavePNGLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()

	if err := SavePNG(dir, "sub-1.png", Resize(solidImage(128), 64)); err != nil {
		t.Fatalf("SavePNG() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "sub-1.png" {
		t.Errorf("directory contains %d entries, want only the icon", len(entries))
	}
}

// TestValidNameRejectsTraversal checks a crafted name cannot point outside the
// icons folder.
func TestValidNameRejectsTraversal(t *testing.T) {
	bad := []string{
		"",
		"..",
		"../secret.png",
		"..\\secret.png",
		"sub/../../data.json",
		"sub\\data.json",
		"/etc/passwd",
	}
	for _, name := range bad {
		if ValidName(name) {
			t.Errorf("ValidName(%q) = true, want false", name)
		}
	}

	good := []string{"sub-1.png", "acc-abc123.png", "app-icon.png"}
	for _, name := range good {
		if !ValidName(name) {
			t.Errorf("ValidName(%q) = false, want true", name)
		}
	}
}

// TestReadDataURIRefusesTraversal checks the reader itself refuses a name
// that tries to escape the icons folder.
func TestReadDataURIRefusesTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "..", "secret.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadDataURI(dir, "../secret.png"); err == nil {
		t.Fatal("ReadDataURI() expected error for a traversing name")
	}
}

// TestSavePNGRefusesBadName checks an invalid name is rejected before any
// file is written.
func TestSavePNGRefusesBadName(t *testing.T) {
	dir := t.TempDir()

	if err := SavePNG(dir, "../evil.png", solidImage(32)); err == nil {
		t.Fatal("SavePNG() expected error for an invalid name")
	}
}

// TestWriteICOProducesValidFile checks the generated .ico has the expected
// header and one entry per wanted size.
func TestWriteICOProducesValidFile(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteICO(&buf, solidImage(512), []int{16, 32, 256}); err != nil {
		t.Fatalf("WriteICO() error = %v", err)
	}

	data := buf.Bytes()
	if reserved := binary.LittleEndian.Uint16(data[0:2]); reserved != 0 {
		t.Errorf("reserved = %d, want 0", reserved)
	}
	if kind := binary.LittleEndian.Uint16(data[2:4]); kind != 1 {
		t.Errorf("type = %d, want 1 (icon)", kind)
	}
	if count := binary.LittleEndian.Uint16(data[4:6]); count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	// Every entry must point at a PNG inside the file.
	for i := 0; i < 3; i++ {
		base := 6 + 16*i
		size := binary.LittleEndian.Uint32(data[base+8 : base+12])
		offset := binary.LittleEndian.Uint32(data[base+12 : base+16])
		if offset+size > uint32(len(data)) {
			t.Fatalf("entry %d points outside the file", i)
		}
		if !bytes.Equal(data[offset:offset+4], []byte{0x89, 'P', 'N', 'G'}) {
			t.Errorf("entry %d does not hold PNG data", i)
		}
	}
}

// TestWriteICOStores256AsZeroWidth checks the 256 pixel entry uses the zero
// marker the ICO format requires.
func TestWriteICOStores256AsZeroWidth(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteICO(&buf, solidImage(512), []int{256}); err != nil {
		t.Fatalf("WriteICO() error = %v", err)
	}

	if width := buf.Bytes()[6]; width != 0 {
		t.Errorf("width = %d, want 0 for the 256 pixel entry", width)
	}
}

// TestFileSizeReportsMissingFile checks the size of an absent icon is zero
// instead of an error.
func TestFileSizeReportsMissingFile(t *testing.T) {
	if got := FileSize(t.TempDir(), "nope.png"); got != 0 {
		t.Errorf("size = %d, want 0", got)
	}
}

// TestPresetsAreComplete checks every preset carries the fields the picker
// needs and uses a known category.
func TestPresetsAreComplete(t *testing.T) {
	known := map[string]bool{"AI": true, "Media": true, "Dev": true, "Storage": true, "Shopping": true}
	seen := map[string]bool{}

	for _, preset := range Presets() {
		if preset.Name == "" || preset.Plan == "" || preset.Prompt == "" {
			t.Errorf("preset %q is missing a field", preset.Name)
		}
		if !known[preset.Category] {
			t.Errorf("preset %q has unknown category %q", preset.Name, preset.Category)
		}
		if preset.Price <= 0 {
			t.Errorf("preset %q has price %v", preset.Name, preset.Price)
		}
		if seen[preset.Name] {
			t.Errorf("duplicate preset name %q", preset.Name)
		}
		seen[preset.Name] = true
	}
}

// TestBuildSubscriptionPromptUsesPreset checks a known service reuses the
// prompt of its preset.
func TestBuildSubscriptionPromptUsesPreset(t *testing.T) {
	want := ""
	for _, preset := range Presets() {
		if preset.Name == "Netflix Premium" {
			want = preset.Prompt
		}
	}
	if want == "" {
		t.Fatal("the Netflix preset is missing")
	}

	if got := BuildSubscriptionPrompt("Netflix Premium", "Media"); got != want {
		t.Errorf("prompt = %q, want the preset prompt", got)
	}
}

// TestBuildSubscriptionPromptFallsBackForUnknownService checks an unknown
// service still gets a usable prompt.
func TestBuildSubscriptionPromptFallsBackForUnknownService(t *testing.T) {
	got := BuildSubscriptionPrompt("Dịch vụ lạ", "AI")

	if got == "" {
		t.Fatal("prompt is empty")
	}
	if !strings.Contains(got, "Dịch vụ lạ") {
		t.Errorf("prompt = %q, want it to mention the service name", got)
	}
}

// TestBuildAvatarPromptIncludesColour checks the avatar prompt carries the
// account colour so the pictures differ from each other.
func TestBuildAvatarPromptIncludesColour(t *testing.T) {
	blue := BuildAvatarPrompt("Mail Chính", "blue")
	rose := BuildAvatarPrompt("Mail Chính", "rose")

	if blue == rose {
		t.Error("two colours produced the same prompt")
	}
	if !strings.Contains(blue, "azure blue") {
		t.Errorf("prompt = %q, want it to mention the blue tone", blue)
	}
}

// TestPromptsAvoidBrandLogos checks the prompts describe the shape and colour
// of a picture instead of asking for a brand logo.
func TestPromptsAvoidBrandLogos(t *testing.T) {
	for _, preset := range Presets() {
		lower := strings.ToLower(preset.Prompt)
		if !strings.Contains(lower, "no text") {
			t.Errorf("prompt for %q does not forbid text", preset.Name)
		}
		if strings.Contains(lower, "logo of") || strings.Contains(lower, "official logo") {
			t.Errorf("prompt for %q asks for a brand logo", preset.Name)
		}
	}
}
