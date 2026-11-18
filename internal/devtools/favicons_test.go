package devtools

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
)

const committedFavicons = "../../web/public"

func renderedFavicons(t *testing.T) (map[string][]byte, faviconColors) {
	t.Helper()
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	files, err := RenderFavicons(themes)
	if err != nil {
		t.Fatal(err)
	}
	colors, err := faviconPalette(themes)
	if err != nil {
		t.Fatal(err)
	}
	return files, colors
}

func TestFaviconsCheckVerifiesCommittedFiles(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Favicons(committedFavicons, true, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q; run mise run generate-favicons", code, errOut.String())
	}
	if want := "verified favicons: " + committedFavicons + "\n"; out.String() != want {
		t.Fatalf("stdout %q, want %q", out.String(), want)
	}
}

func TestFaviconsCheckReportsStaleFiles(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	if code := Favicons(dir, false, &out, &errOut); code != 0 {
		t.Fatalf("write exit %d, stderr %q", code, errOut.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte("<svg/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := Favicons(dir, true, &out, &errOut); code != 1 {
		t.Fatalf("check exit %d, want 1", code)
	}
	if got := errOut.String(); !strings.Contains(got, filepath.Join(dir, "favicon.svg")) || strings.Contains(got, "favicon.ico") {
		t.Fatalf("stderr %q should name only the stale favicon.svg", got)
	}
}

func TestFaviconSVGUsesTheDefaultThemeOnWholeUnits(t *testing.T) {
	files, colors := renderedFavicons(t)
	svg := string(files["favicon.svg"])
	for _, want := range []string{`viewBox="0 0 32 32"`, `shape-rendering="crispEdges"`, string(colors.tile), string(colors.digit), string(colors.shadow)} {
		if !strings.Contains(svg, want) {
			t.Errorf("favicon.svg lacks %q", want)
		}
	}
	if regexp.MustCompile(`\d\.\d`).MatchString(svg) {
		t.Errorf("favicon.svg has a fractional coordinate:\n%s", svg)
	}
}

// decodeICO returns the images an ICO holds, keyed by the size its directory declares.
func decodeICO(t *testing.T, data []byte) map[int]image.Image {
	t.Helper()
	var header [3]uint16
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header); err != nil {
		t.Fatal(err)
	}
	if header[0] != 0 || header[1] != 1 {
		t.Fatalf("ICO header %v", header)
	}
	images := map[int]image.Image{}
	for i := range int(header[2]) {
		entry := data[6+16*i : 22+16*i]
		size := int(entry[0])
		length := binary.LittleEndian.Uint32(entry[8:12])
		offset := binary.LittleEndian.Uint32(entry[12:16])
		img, err := png.Decode(bytes.NewReader(data[offset : offset+length]))
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Fatalf("ICO entry %d declares %d but holds %v", i, size, b)
		}
		images[size] = img
	}
	return images
}

// pixelColors counts the opaque colors an icon uses; a crisp icon uses only its three palette colors.
func pixelColors(img image.Image) map[color.NRGBA]int {
	counts := map[color.NRGBA]int{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 255 {
				counts[c]++
			}
		}
	}
	return counts
}

func TestFaviconRastersAreCrisp(t *testing.T) {
	files, colors := renderedFavicons(t)
	images := decodeICO(t, files["favicon.ico"])
	if len(images) != 3 || images[16] == nil || images[32] == nil || images[48] == nil {
		t.Fatalf("favicon.ico holds sizes %v, want 16, 32, and 48", len(images))
	}
	touch, err := png.Decode(bytes.NewReader(files["apple-touch-icon.png"]))
	if err != nil {
		t.Fatal(err)
	}
	if b := touch.Bounds(); b.Dx() != 180 || b.Dy() != 180 {
		t.Fatalf("apple-touch-icon.png is %v, want 180x180", b)
	}
	images[180] = touch

	palette := map[color.NRGBA]string{rgb(colors.tile): "tile", rgb(colors.digit): "digit", rgb(colors.shadow): "shadow"}
	for size, img := range images {
		counts := pixelColors(img)
		for c := range counts {
			if _, ok := palette[c]; !ok {
				t.Errorf("%dpx has a blended opaque color %v", size, c)
			}
		}
		_, hasShadow := counts[rgb(colors.shadow)]
		if wantShadow := size != 16; hasShadow != wantShadow {
			t.Errorf("%dpx shadow present %v, want %v", size, hasShadow, wantShadow)
		}
	}
	if c := color.NRGBAModel.Convert(touch.At(0, 0)).(color.NRGBA); c != rgb(colors.tile) {
		t.Errorf("apple-touch-icon.png corner %v, want a full-bleed tile", c)
	}
}
