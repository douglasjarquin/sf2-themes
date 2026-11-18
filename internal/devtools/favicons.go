package devtools

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
)

// faviconTheme is the catalog theme the site renders by default; the icon takes its colors.
const faviconTheme = "ryu"

// digitTwo is the pixel-art "2" on a 5x7 grid.
var digitTwo = [...]string{
	".###.",
	"#...#",
	"....#",
	"...#.",
	"..#..",
	".#...",
	"#####",
}

// iconLayout places the digit on a square canvas of size pixels. Every value is a whole pixel so the
// raster stays crisp: cell is the size of one digit pixel, shadow the drop-shadow offset (0 drops it),
// and radius the tile's corner radius (0 is a full-bleed square, as touch icons expect).
type iconLayout struct {
	size, cell, shadow, radius int
}

func (l iconLayout) origin() (x, y int) {
	w := len(digitTwo[0])*l.cell + l.shadow
	h := len(digitTwo)*l.cell + l.shadow
	return (l.size - w) / 2, (l.size - h) / 2
}

// The SVG keeps the 32px layout, which lands on whole pixels wherever browsers draw a 16px favicon at 2x.
// The 16px raster drops the shadow, which would smear a digit only two pixels wide.
var (
	svgLayout   = iconLayout{size: 32, cell: 3, shadow: 1, radius: 6}
	icoLayouts  = []iconLayout{{16, 2, 0, 3}, {32, 3, 1, 6}, {48, 5, 2, 9}}
	touchLayout = iconLayout{size: 180, cell: 16, shadow: 5, radius: 0}
)

type faviconColors struct{ tile, digit, shadow catalog.HexColor }

func faviconPalette(themes []catalog.Theme) (faviconColors, error) {
	theme, err := catalog.GetTheme(faviconTheme, themes)
	if err != nil {
		return faviconColors{}, err
	}
	return faviconColors{theme.UI.Background, theme.UI.Accent, theme.UI.AccentSecondary}, nil
}

// RenderFavicons returns every favicon file keyed by its name under web/public.
func RenderFavicons(themes []catalog.Theme) (map[string][]byte, error) {
	colors, err := faviconPalette(themes)
	if err != nil {
		return nil, err
	}
	icons := make([]*image.NRGBA, len(icoLayouts))
	for i, layout := range icoLayouts {
		icons[i] = renderIcon(layout, colors)
	}
	ico, err := encodeICO(icons)
	if err != nil {
		return nil, err
	}
	touch, err := encodePNG(renderIcon(touchLayout, colors))
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		"favicon.svg":          []byte(renderSVG(svgLayout, colors)),
		"favicon.ico":          ico,
		"apple-touch-icon.png": touch,
	}, nil
}

// FaviconNames lists the generated files in a stable order.
var FaviconNames = []string{"favicon.svg", "favicon.ico", "apple-touch-icon.png"}

func renderSVG(l iconLayout, c faviconColors) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+"\n", l.size, l.size)
	fmt.Fprintf(&b, `  <rect width="%d" height="%d" rx="%d" fill="%s" shape-rendering="geometricPrecision"/>`+"\n", l.size, l.size, l.radius, c.tile)
	fmt.Fprintf(&b, `  <path fill="%s" d="%s"/>`+"\n", c.shadow, digitPath(l, l.shadow))
	fmt.Fprintf(&b, `  <path fill="%s" d="%s"/>`+"\n", c.digit, digitPath(l, 0))
	b.WriteString("</svg>\n")
	return b.String()
}

func digitPath(l iconLayout, offset int) string {
	ox, oy := l.origin()
	var parts []string
	for row, line := range digitTwo {
		for col, ch := range line {
			if ch == '#' {
				x, y := ox+col*l.cell+offset, oy+row*l.cell+offset
				parts = append(parts, "M"+strconv.Itoa(x)+" "+strconv.Itoa(y)+"h"+strconv.Itoa(l.cell)+"v"+strconv.Itoa(l.cell)+"h-"+strconv.Itoa(l.cell)+"z")
			}
		}
	}
	return strings.Join(parts, "")
}

func renderIcon(l iconLayout, c faviconColors) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, l.size, l.size))
	tile := rgb(c.tile)
	for y := range l.size {
		for x := range l.size {
			px := tile
			px.A = uint8(255 * tileCoverage(l, x, y) / 16)
			img.SetNRGBA(x, y, px)
		}
	}
	ox, oy := l.origin()
	paint := func(offset int, col color.NRGBA) {
		for row, line := range digitTwo {
			for column, ch := range line {
				if ch != '#' {
					continue
				}
				x0, y0 := ox+column*l.cell+offset, oy+row*l.cell+offset
				for y := y0; y < y0+l.cell; y++ {
					for x := x0; x < x0+l.cell; x++ {
						img.SetNRGBA(x, y, col)
					}
				}
			}
		}
	}
	if l.shadow > 0 {
		paint(l.shadow, rgb(c.shadow))
	}
	paint(0, rgb(c.digit))
	return img
}

// tileCoverage samples a pixel on a 4x4 grid and returns how many of the 16 samples fall inside the
// rounded tile, which anti-aliases only the corners.
func tileCoverage(l iconLayout, x, y int) int {
	if l.radius == 0 {
		return 16
	}
	r := float64(l.radius)
	size := float64(l.size)
	inside := 0
	for sy := range 4 {
		for sx := range 4 {
			px, py := float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4
			cx, cy := min(max(px, r), size-r), min(max(py, r), size-r)
			if dx, dy := px-cx, py-cy; dx*dx+dy*dy <= r*r {
				inside++
			}
		}
	}
	return inside
}

func rgb(hex catalog.HexColor) color.NRGBA {
	v, _ := strconv.ParseUint(strings.TrimPrefix(string(hex), "#"), 16, 32)
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeICO wraps PNG images in an ICO container, which every browser that reads favicon.ico accepts.
func encodeICO(icons []*image.NRGBA) ([]byte, error) {
	images := make([][]byte, len(icons))
	for i, icon := range icons {
		data, err := encodePNG(icon)
		if err != nil {
			return nil, err
		}
		images[i] = data
	}
	var buf bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&buf, le, [3]uint16{0, 1, uint16(len(icons))})
	offset := 6 + 16*len(icons)
	for i, icon := range icons {
		size := icon.Bounds().Dx()
		dim := uint8(size)
		if size >= 256 {
			dim = 0
		}
		_ = binary.Write(&buf, le, struct {
			W, H, Colors, Reserved uint8
			Planes, Bits           uint16
			Bytes, Offset          uint32
		}{dim, dim, 0, 0, 1, 32, uint32(len(images[i])), uint32(offset)})
		offset += len(images[i])
	}
	for _, data := range images {
		buf.Write(data)
	}
	return buf.Bytes(), nil
}

// Favicons writes the favicon files into output, or with check verifies the files there are current.
func Favicons(output string, check bool, stdout, stderr io.Writer) int {
	themes, err := catalog.Load()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	files, err := RenderFavicons(themes)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if check {
		var stale []string
		for _, name := range FaviconNames {
			current, err := os.ReadFile(filepath.Join(output, name))
			if err != nil || !bytes.Equal(current, files[name]) {
				stale = append(stale, filepath.Join(output, name))
			}
		}
		if len(stale) > 0 {
			_, _ = fmt.Fprintf(stderr, "stale favicons: %s\nrun: mise run generate-favicons\n", strings.Join(stale, ", "))
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "verified favicons: %s\n", output)
		return 0
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	for _, name := range FaviconNames {
		path := filepath.Join(output, name)
		if err := os.WriteFile(path, files[name], 0o644); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "wrote %s\n", path)
	}
	return 0
}
