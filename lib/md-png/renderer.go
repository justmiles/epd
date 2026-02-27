package mdpng

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"strings"

	"github.com/fogleman/gg"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// renderer walks a goldmark AST and draws onto a gg.Context.
type renderer struct {
	cfg    *config
	source []byte

	// Style state (stack-based for nesting).
	bold          int
	italic        int
	codeInline    int
	strikethrough int
	heading       int
	listDepth     int
	listIndex     []int // current item index per list depth (for ordered lists)
	blockquote    int
	inCodeBlock   bool

	// Layout state.
	y          float64 // current Y position
	lineHeight float64
}

func newRenderer(cfg *config, source []byte) *renderer {
	return &renderer{
		cfg:    cfg,
		source: source,
	}
}

// render performs rendering: optionally auto-fitting the font size, then drawing.
func (r *renderer) render(doc ast.Node, w io.Writer) error {
	contentWidth := float64(r.cfg.Width - 2*r.cfg.Padding)

	// If height is fixed, auto-fit font size via binary search.
	if r.cfg.Height > 0 {
		r.autoFitFontSize(doc, contentWidth)
	}

	// Measure pass to determine total height.
	measuredHeight := r.measure(doc, contentWidth)

	totalHeight := measuredHeight
	if r.cfg.Height > 0 {
		totalHeight = r.cfg.Height
	} else if totalHeight < 100 {
		totalHeight = 100
	}

	// Draw pass.
	r.y = float64(r.cfg.Padding)
	r.resetStyle()
	dc := gg.NewContext(r.cfg.Width, totalHeight)

	// Background.
	if r.cfg.DarkMode {
		dc.SetColor(color.RGBA{30, 30, 30, 255})
	} else {
		dc.SetColor(color.White)
	}
	dc.Clear()

	r.loadFont(dc, false, false)
	r.walkNode(doc, dc, contentWidth, false)

	return dc.EncodePNG(w)
}

// measure runs a measurement-only pass and returns the total height needed.
func (r *renderer) measure(doc ast.Node, contentWidth float64) int {
	r.y = float64(r.cfg.Padding)
	r.resetStyle()
	dc := gg.NewContext(r.cfg.Width, 10000)
	r.loadFont(dc, false, false)
	r.walkNode(doc, dc, contentWidth, true)
	return int(r.y) + r.cfg.Padding
}

// autoFitFontSize uses binary search to find the largest font size
// (between minFontSize and the configured size) where content fits
// within the fixed height.
func (r *renderer) autoFitFontSize(doc ast.Node, contentWidth float64) {
	const minFontSize = 4.0
	maxSize := r.cfg.FontSize
	targetHeight := r.cfg.Height

	// Quick check: does the current size already fit?
	h := r.measure(doc, contentWidth)
	if h <= targetHeight {
		return // already fits
	}

	// Binary search for the best font size.
	lo, hi := minFontSize, maxSize
	for hi-lo > 0.5 {
		mid := (lo + hi) / 2
		r.cfg.FontSize = mid
		h = r.measure(doc, contentWidth)
		if h <= targetHeight {
			lo = mid // fits, try larger
		} else {
			hi = mid // too big, try smaller
		}
	}
	r.cfg.FontSize = lo
}

func (r *renderer) resetStyle() {
	r.bold = 0
	r.italic = 0
	r.codeInline = 0
	r.strikethrough = 0
	r.heading = 0
	r.listDepth = 0
	r.listIndex = nil
	r.blockquote = 0
	r.inCodeBlock = false
}

func (r *renderer) textColor() color.Color {
	if r.cfg.DarkMode {
		return color.RGBA{230, 230, 230, 255}
	}
	return color.RGBA{30, 30, 30, 255}
}

func (r *renderer) mutedColor() color.Color {
	if r.cfg.DarkMode {
		return color.RGBA{150, 150, 160, 255}
	}
	return color.RGBA{100, 100, 110, 255}
}

func (r *renderer) codeBgColor() color.Color {
	if r.cfg.DarkMode {
		return color.RGBA{50, 50, 55, 255}
	}
	return color.RGBA{240, 240, 240, 255}
}

func (r *renderer) linkColor() color.Color {
	if r.cfg.DarkMode {
		return color.RGBA{100, 160, 255, 255}
	}
	return color.RGBA{0, 100, 200, 255}
}

func (r *renderer) ruleColor() color.Color {
	if r.cfg.DarkMode {
		return color.RGBA{80, 80, 80, 255}
	}
	return color.RGBA{200, 200, 200, 255}
}

func (r *renderer) fontSize() float64 {
	switch r.heading {
	case 1:
		return r.cfg.FontSize * 1.25
	case 2:
		return r.cfg.FontSize * 1.15
	case 3:
		return r.cfg.FontSize * 1.05
	case 4:
		return r.cfg.FontSize * 1.0
	case 5:
		return r.cfg.FontSize * 1.0
	case 6:
		return r.cfg.FontSize * 1.0
	default:
		return r.cfg.FontSize
	}
}

// loadFont sets the current font on the context based on style state.
func (r *renderer) loadFont(dc *gg.Context, bold, italic bool) {
	size := r.fontSize()
	if r.heading > 0 {
		bold = true
	}

	// fogleman/gg needs a TTF font loaded. We'll use the built-in
	// basic font as a fallback and try to load system fonts.
	// For simplicity, use LoadFontFace with common system font paths.
	fontPath := findFont(bold, italic, r.codeInline > 0 || r.inCodeBlock)
	if fontPath != "" {
		if err := dc.LoadFontFace(fontPath, size); err == nil {
			return
		}
	}
	// Last resort: try any common font
	for _, f := range fallbackFonts() {
		if err := dc.LoadFontFace(f, size); err == nil {
			return
		}
	}
}

// findFont returns a system TTF font path matching the desired style.
func findFont(bold, italic, mono bool) string {
	if mono {
		paths := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
			"/usr/share/fonts/truetype/noto/NotoSansMono-Regular.ttf",
			"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
			"/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf",
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
	}

	if bold && italic {
		paths := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-BoldOblique.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-BoldItalic.ttf",
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
	}
	if bold {
		paths := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
			"/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf",
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
	}
	if italic {
		paths := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Oblique.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Italic.ttf",
		}
		for _, p := range paths {
			if fileExists(p) {
				return p
			}
		}
	}

	// Regular
	paths := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
	}
	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fallbackFonts() []string {
	return []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu-sans-fonts/DejaVuSans.ttf",
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// walkNode recursively processes AST nodes.
func (r *renderer) walkNode(node ast.Node, dc *gg.Context, contentWidth float64, measureOnly bool) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.processNode(child, dc, contentWidth, measureOnly)
	}
}

func (r *renderer) processNode(node ast.Node, dc *gg.Context, contentWidth float64, measureOnly bool) {
	switch n := node.(type) {
	case *ast.Heading:
		r.heading = n.Level
		r.loadFont(dc, true, false)

		// Add spacing before heading.
		if r.y > float64(r.cfg.Padding) {
			r.y += r.fontSize() * 0.5
		}

		text := r.collectText(n)
		r.drawWrappedText(dc, text, contentWidth, measureOnly)
		r.y += r.fontSize() * 0.3 // spacing after heading

		r.heading = 0
		r.loadFont(dc, false, false)

	case *ast.Paragraph:
		if r.y > float64(r.cfg.Padding) {
			r.y += r.fontSize() * 0.4
		}
		r.renderInlineChildren(n, dc, contentWidth, measureOnly)
		r.y += r.fontSize() * 0.4

	case *ast.TextBlock:
		r.renderInlineChildren(n, dc, contentWidth, measureOnly)

	case *ast.ThematicBreak:
		r.y += r.fontSize() * 0.5
		if !measureOnly {
			xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20
			dc.SetColor(r.ruleColor())
			dc.SetLineWidth(1)
			dc.DrawLine(xStart, r.y, float64(r.cfg.Width-r.cfg.Padding), r.y)
			dc.Stroke()
		}
		r.y += r.fontSize() * 0.5

	case *ast.CodeBlock, *ast.FencedCodeBlock:
		r.y += r.fontSize() * 0.3
		r.inCodeBlock = true
		r.loadFont(dc, false, false)

		var lines []string
		l := node.Lines()
		for i := 0; i < l.Len(); i++ {
			line := l.At(i)
			lines = append(lines, string(line.Value(r.source)))
		}
		codeText := strings.Join(lines, "")
		codeText = strings.TrimRight(codeText, "\n")

		xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20 + float64(r.listDepth)*20

		if !measureOnly {
			// Draw code background.
			codeLines := strings.Split(codeText, "\n")
			codeHeight := float64(len(codeLines)) * r.fontSize() * 1.5
			dc.SetColor(r.codeBgColor())
			dc.DrawRoundedRectangle(xStart, r.y, contentWidth-float64(r.listDepth)*20-float64(r.blockquote)*20, codeHeight+r.fontSize(), 4)
			dc.Fill()
		}

		r.y += r.fontSize() * 0.5
		codeLines := strings.Split(codeText, "\n")
		for _, line := range codeLines {
			if !measureOnly {
				dc.SetColor(r.textColor())
				dc.DrawString(line, xStart+8, r.y+r.fontSize())
			}
			r.y += r.fontSize() * 1.5
		}
		r.y += r.fontSize() * 0.5

		r.inCodeBlock = false
		r.loadFont(dc, false, false)

	case *ast.Blockquote:
		r.blockquote++
		r.y += r.fontSize() * 0.2

		startY := r.y
		r.walkNode(n, dc, contentWidth-20, measureOnly)

		if !measureOnly {
			// Draw left border bar.
			x := float64(r.cfg.Padding) + float64(r.blockquote-1)*20 + 4
			dc.SetColor(r.ruleColor())
			dc.SetLineWidth(3)
			dc.DrawLine(x, startY, x, r.y)
			dc.Stroke()
		}
		r.y += r.fontSize() * 0.2
		r.blockquote--

	case *ast.List:
		r.listDepth++
		if n.IsOrdered() {
			r.listIndex = append(r.listIndex, 0)
		} else {
			r.listIndex = append(r.listIndex, -1) // -1 means unordered
		}

		if r.y > float64(r.cfg.Padding) && r.listDepth == 1 {
			r.y += r.fontSize() * 0.3
		}

		r.walkNode(n, dc, contentWidth, measureOnly)

		if r.listDepth == 1 {
			r.y += r.fontSize() * 0.3
		}
		r.listDepth--
		if len(r.listIndex) > 0 {
			r.listIndex = r.listIndex[:len(r.listIndex)-1]
		}

	case *ast.ListItem:
		indent := float64(r.listDepth)*20 + float64(r.blockquote)*20
		xStart := float64(r.cfg.Padding) + indent

		// Check if this list item has a task checkbox.
		checkbox := r.findCheckbox(n)

		// Determine bullet or number.
		bullet := "•"
		if checkbox != nil {
			if checkbox.IsChecked {
				bullet = "☑"
			} else {
				bullet = "☐"
			}
		} else if len(r.listIndex) > 0 && r.listIndex[len(r.listIndex)-1] >= 0 {
			r.listIndex[len(r.listIndex)-1]++
			bullet = fmt.Sprintf("%d.", r.listIndex[len(r.listIndex)-1])
		}

		if !measureOnly {
			if checkbox != nil && checkbox.IsChecked {
				dc.SetColor(r.mutedColor())
			} else {
				dc.SetColor(r.textColor())
			}
			dc.DrawString(bullet, xStart, r.y+r.fontSize())
		}

		// Render list item content indented past the bullet.
		oldY := r.y
		bulletWidth := 20.0
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				if !measureOnly {
					if checkbox != nil && checkbox.IsChecked {
						dc.SetColor(r.mutedColor())
					} else {
						dc.SetColor(r.textColor())
					}
				}
				r.renderInlineChildrenAt(child, dc, xStart+bulletWidth, contentWidth-indent-bulletWidth, measureOnly)
			default:
				r.processNode(child, dc, contentWidth, measureOnly)
			}
		}
		if r.y == oldY {
			r.y += r.fontSize() * 1.5
		}

	case *east.Strikethrough:
		text := r.collectText(n)
		if !measureOnly {
			dc.SetColor(r.mutedColor())
			xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20 + float64(r.listDepth)*10
			w, _ := dc.MeasureString(text)
			textY := r.y + r.fontSize()
			dc.DrawString(text, xStart, textY)
			// Draw strikethrough line.
			dc.SetLineWidth(1)
			dc.DrawLine(xStart, textY-r.fontSize()*0.35, xStart+w, textY-r.fontSize()*0.35)
			dc.Stroke()
		}
		r.y += r.fontSize() * 1.5

	case *east.TaskCheckBox:
		// Handled as part of list item rendering above.

	case *east.Table:
		r.renderTable(n, dc, contentWidth, measureOnly)

	default:
		// For unknown block nodes, try to walk children.
		if node.HasChildren() {
			r.walkNode(node, dc, contentWidth, measureOnly)
		}
	}
}

// renderInlineChildren collects the text from inline children and draws it
// with per-segment styling (bold, italic, code).
func (r *renderer) renderInlineChildren(node ast.Node, dc *gg.Context, contentWidth float64, measureOnly bool) {
	xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20 + float64(r.listDepth)*20
	effectiveWidth := contentWidth - float64(r.listDepth)*20 - float64(r.blockquote)*20
	r.renderInlineChildrenAt(node, dc, xStart, effectiveWidth, measureOnly)
}

// renderInlineChildrenAt renders inline children at a specific x position with per-segment styling.
func (r *renderer) renderInlineChildrenAt(node ast.Node, dc *gg.Context, xStart, effectiveWidth float64, measureOnly bool) {
	segments := r.collectInlineSegments(node)
	if len(segments) == 0 {
		return
	}

	// Build styled words: each word carries its own style.
	type styledWord struct {
		text          string
		bold          bool
		italic        bool
		code          bool
		strikethrough bool
	}

	var words []styledWord
	for _, seg := range segments {
		segWords := strings.Fields(seg.text)
		for _, w := range segWords {
			words = append(words, styledWord{
				text:          w,
				bold:          seg.bold,
				italic:        seg.italic,
				code:          seg.code,
				strikethrough: seg.strikethrough,
			})
		}
	}

	if len(words) == 0 {
		return
	}

	lineSpacing := r.fontSize() * 1.5
	curX := xStart

	for i, word := range words {
		// Load the appropriate font for this word.
		if word.code {
			r.codeInline++
		}
		r.loadFont(dc, word.bold, word.italic)
		if word.code {
			r.codeInline--
		}

		wordW, _ := dc.MeasureString(word.text)
		spaceW, _ := dc.MeasureString(" ")

		// Add space before word (except first word on a line).
		if i > 0 && curX > xStart {
			curX += spaceW
		}

		// Wrap to next line if needed.
		if curX+wordW > xStart+effectiveWidth && curX > xStart {
			r.y += lineSpacing
			curX = xStart
		}

		if !measureOnly {
			if word.strikethrough {
				dc.SetColor(r.mutedColor())
			} else {
				dc.SetColor(r.textColor())
			}
			dc.DrawString(word.text, curX, r.y+r.fontSize())

			// Draw strikethrough line.
			if word.strikethrough {
				lineY := r.y + r.fontSize()*0.65
				dc.SetLineWidth(1)
				dc.DrawLine(curX, lineY, curX+wordW, lineY)
				dc.Stroke()
			}
		}
		curX += wordW
	}

	// Advance past the last line.
	r.y += lineSpacing

	// Restore default font.
	r.loadFont(dc, false, false)
}

type inlineSegment struct {
	text          string
	bold          bool
	italic        bool
	code          bool
	strikethrough bool
	link          string
}

// collectInlineSegments walks inline children to gather styled text segments.
func (r *renderer) collectInlineSegments(node ast.Node) []inlineSegment {
	var segments []inlineSegment
	r.walkInline(node, &segments)
	return segments
}

func (r *renderer) walkInline(node ast.Node, segments *[]inlineSegment) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			text := string(n.Segment.Value(r.source))
			if n.SoftLineBreak() {
				text += " "
			}
			*segments = append(*segments, inlineSegment{
				text:          text,
				bold:          r.bold > 0,
				italic:        r.italic > 0,
				code:          r.codeInline > 0,
				strikethrough: r.strikethrough > 0,
			})

		case *ast.String:
			*segments = append(*segments, inlineSegment{
				text:          string(n.Value),
				strikethrough: r.strikethrough > 0,
			})

		case *ast.CodeSpan:
			code := r.collectText(n)
			*segments = append(*segments, inlineSegment{
				text:          code,
				code:          true,
				strikethrough: r.strikethrough > 0,
			})

		case *ast.Emphasis:
			if n.Level == 2 {
				r.bold++
				r.walkInline(n, segments)
				r.bold--
			} else {
				r.italic++
				r.walkInline(n, segments)
				r.italic--
			}

		case *east.Strikethrough:
			r.strikethrough++
			r.walkInline(n, segments)
			r.strikethrough--

		case *east.TaskCheckBox:
			// Skip — rendered as ☑/☐ by list item code.

		case *ast.Link:
			linkText := r.collectText(n)
			dest := string(n.Destination)
			*segments = append(*segments, inlineSegment{
				text: linkText,
				link: dest,
			})

		case *ast.AutoLink:
			*segments = append(*segments, inlineSegment{
				text: string(n.URL(r.source)),
				link: string(n.URL(r.source)),
			})

		case *ast.RawHTML:
			// Skip raw HTML.

		case *ast.HTMLBlock:
			// Skip HTML blocks.

		default:
			if child.HasChildren() {
				r.walkInline(child, segments)
			}
		}
	}
}

// collectText recursively collects all text content from a node.
func (r *renderer) collectText(node ast.Node) string {
	var buf strings.Builder
	r.collectTextRecursive(node, &buf)
	return buf.String()
}

func (r *renderer) collectTextRecursive(node ast.Node, buf *strings.Builder) {
	if node.Type() == ast.TypeInline {
		switch n := node.(type) {
		case *ast.Text:
			buf.Write(n.Segment.Value(r.source))
			if n.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		case *ast.String:
			buf.Write(n.Value)
		case *ast.CodeSpan:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				r.collectTextRecursive(child, buf)
			}
			return
		}
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.collectTextRecursive(child, buf)
	}
}

// drawWrappedText draws text at the default X position with word wrapping.
func (r *renderer) drawWrappedText(dc *gg.Context, text string, width float64, measureOnly bool) {
	xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20 + float64(r.listDepth)*20
	effectiveWidth := width - float64(r.listDepth)*20 - float64(r.blockquote)*20
	r.drawWrappedTextAt(dc, text, xStart, effectiveWidth, measureOnly)
}

// drawWrappedTextAt draws word-wrapped text starting at (x, current y).
func (r *renderer) drawWrappedTextAt(dc *gg.Context, text string, x, maxWidth float64, measureOnly bool) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}

	if !measureOnly {
		dc.SetColor(r.textColor())
	}

	lineSpacing := r.fontSize() * 1.5
	var line strings.Builder
	line.WriteString(words[0])

	for i := 1; i < len(words); i++ {
		test := line.String() + " " + words[i]
		w, _ := dc.MeasureString(test)
		if w > maxWidth && line.Len() > 0 {
			// Emit current line.
			if !measureOnly {
				dc.DrawString(line.String(), x, r.y+r.fontSize())
			}
			r.y += lineSpacing
			line.Reset()
			line.WriteString(words[i])
		} else {
			line.WriteString(" ")
			line.WriteString(words[i])
		}
	}
	// Emit last line.
	if line.Len() > 0 {
		if !measureOnly {
			dc.DrawString(line.String(), x, r.y+r.fontSize())
		}
		r.y += lineSpacing
	}
}

// findCheckbox looks for a TaskCheckBox node within a list item's first paragraph.
func (r *renderer) findCheckbox(listItem ast.Node) *east.TaskCheckBox {
	for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
		for inline := child.FirstChild(); inline != nil; inline = inline.NextSibling() {
			if cb, ok := inline.(*east.TaskCheckBox); ok {
				return cb
			}
		}
	}
	return nil
}

// tableRow holds the data for one row of a table.
type tableRow struct {
	cells    []*east.TableCell // AST nodes for styled rendering
	text     []string          // plain text for measurement
	isHeader bool
}

// renderTable renders a GFM table with borders and header styling.
func (r *renderer) renderTable(table *east.Table, dc *gg.Context, contentWidth float64, measureOnly bool) {
	r.y += r.fontSize() * 0.4

	xStart := float64(r.cfg.Padding) + float64(r.blockquote)*20 + float64(r.listDepth)*20
	effectiveWidth := contentWidth - float64(r.listDepth)*20 - float64(r.blockquote)*20

	// Collect table data: header row + body rows.
	var rows []tableRow
	var alignments []east.Alignment

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch section := child.(type) {
		case *east.TableHeader:
			// The header contains a single TableRow with the header cells.
			for row := section.FirstChild(); row != nil; row = row.NextSibling() {
				if tr, ok := row.(*east.TableRow); ok {
					cells, text := r.collectRowCellNodes(tr)
					rows = append(rows, tableRow{cells: cells, text: text, isHeader: true})
				}
			}
			// Collect alignments from the table.
			alignments = table.Alignments
		case *east.TableRow:
			// Body rows are direct TableRow children of the Table.
			cells, text := r.collectRowCellNodes(section)
			rows = append(rows, tableRow{cells: cells, text: text, isHeader: false})
		}
	}

	if len(rows) == 0 {
		return
	}

	// Determine number of columns.
	numCols := 0
	for _, row := range rows {
		if len(row.text) > numCols {
			numCols = len(row.text)
		}
	}
	if numCols == 0 {
		return
	}

	// Measure column widths using plain text.
	plainRows := make([][]string, len(rows))
	for i, row := range rows {
		plainRows[i] = row.text
	}
	colWidths := r.measureColumnWidths(dc, plainRows, numCols)

	// Scale columns to fit available width.
	totalWidth := 0.0
	cellPadding := r.fontSize() * 0.8
	for _, w := range colWidths {
		totalWidth += w + cellPadding*2
	}
	if totalWidth > effectiveWidth {
		scale := effectiveWidth / totalWidth
		for i := range colWidths {
			colWidths[i] *= scale
		}
		totalWidth = effectiveWidth
	}

	rowHeight := r.fontSize() * 2.0

	// Draw table.
	for _, row := range rows {
		rowY := r.y

		if !measureOnly {
			// Draw row background for header.
			if row.isHeader {
				dc.SetColor(r.codeBgColor())
				dc.DrawRectangle(xStart, rowY, totalWidth, rowHeight)
				dc.Fill()
			}

			// Draw cell text with inline styling.
			cellX := xStart
			for colIdx := 0; colIdx < numCols; colIdx++ {
				colW := colWidths[colIdx] + cellPadding*2

				// Determine alignment.
				align := east.AlignNone
				if colIdx < len(alignments) {
					align = alignments[colIdx]
				}

				if colIdx < len(row.cells) && row.cells[colIdx] != nil {
					r.renderTableCell(dc, row.cells[colIdx], cellX, rowY, colW, rowHeight, cellPadding, align, row.isHeader)
				}

				cellX += colW
			}

			// Draw horizontal border below this row.
			dc.SetColor(r.ruleColor())
			dc.SetLineWidth(1)
			if row.isHeader {
				dc.SetLineWidth(2)
			}
			dc.DrawLine(xStart, rowY+rowHeight, xStart+totalWidth, rowY+rowHeight)
			dc.Stroke()

			// Draw vertical borders.
			dc.SetLineWidth(1)
			cellX = xStart
			for colIdx := 0; colIdx <= numCols; colIdx++ {
				dc.DrawLine(cellX, rowY, cellX, rowY+rowHeight)
				dc.Stroke()
				if colIdx < numCols {
					cellX += colWidths[colIdx] + cellPadding*2
				}
			}
		}

		r.y += rowHeight
	}

	// Reset font after header bold.
	r.loadFont(dc, false, false)
	r.y += r.fontSize() * 0.4
}

// renderTableCell draws one cell's inline content with per-segment styling.
func (r *renderer) renderTableCell(dc *gg.Context, cell *east.TableCell, cellX, rowY, colW, rowHeight, cellPadding float64, align east.Alignment, isHeader bool) {
	segments := r.collectInlineSegments(cell)
	if len(segments) == 0 {
		return
	}

	textY := rowY + rowHeight*0.65

	// Measure total styled width and build styled words.
	type styledWord struct {
		text   string
		bold   bool
		italic bool
	}

	var words []styledWord
	for _, seg := range segments {
		segWords := strings.Fields(seg.text)
		for _, w := range segWords {
			words = append(words, styledWord{
				text:   w,
				bold:   seg.bold || isHeader,
				italic: seg.italic,
			})
		}
	}

	if len(words) == 0 {
		return
	}

	// Measure total width of all words with spaces.
	totalTextW := 0.0
	for i, word := range words {
		r.loadFont(dc, word.bold, word.italic)
		ww, _ := dc.MeasureString(word.text)
		totalTextW += ww
		if i > 0 {
			spW, _ := dc.MeasureString(" ")
			totalTextW += spW
		}
	}

	maxTextW := colW - cellPadding*2

	// Determine starting X based on alignment.
	var startX float64
	switch align {
	case east.AlignCenter:
		startX = cellX + (colW-math.Min(totalTextW, maxTextW))/2
	case east.AlignRight:
		startX = cellX + colW - cellPadding - math.Min(totalTextW, maxTextW)
	default:
		startX = cellX + cellPadding
	}

	// Draw each word with its style, truncating if needed.
	curX := startX
	for i, word := range words {
		r.loadFont(dc, word.bold, word.italic)
		ww, _ := dc.MeasureString(word.text)

		if i > 0 {
			spW, _ := dc.MeasureString(" ")
			curX += spW
		}

		// Check if we'd exceed the cell boundary.
		if curX+ww > cellX+colW-cellPadding {
			// Truncate with ellipsis.
			ellipsisW, _ := dc.MeasureString("…")
			remaining := (cellX + colW - cellPadding) - curX - ellipsisW
			if remaining > 0 {
				truncated := word.text
				for len(truncated) > 1 {
					truncated = truncated[:len(truncated)-1]
					tw, _ := dc.MeasureString(truncated)
					if tw <= remaining {
						dc.SetColor(r.textColor())
						dc.DrawString(truncated+"…", curX, textY)
						break
					}
				}
			}
			break
		}

		dc.SetColor(r.textColor())
		dc.DrawString(word.text, curX, textY)
		curX += ww
	}

	r.loadFont(dc, false, false)
}

// collectRowCellNodes collects cell AST nodes and their plain text from a table row.
func (r *renderer) collectRowCellNodes(row *east.TableRow) ([]*east.TableCell, []string) {
	var cells []*east.TableCell
	var texts []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if tc, ok := cell.(*east.TableCell); ok {
			cells = append(cells, tc)
			texts = append(texts, strings.TrimSpace(r.collectText(tc)))
		}
	}
	return cells, texts
}

// measureColumnWidths measures the natural width of each column.
func (r *renderer) measureColumnWidths(dc *gg.Context, rows [][]string, numCols int) []float64 {
	colWidths := make([]float64, numCols)
	for _, row := range rows {
		for colIdx := 0; colIdx < numCols && colIdx < len(row); colIdx++ {
			w, _ := dc.MeasureString(row[colIdx])
			colWidths[colIdx] = math.Max(colWidths[colIdx], w)
		}
	}
	// Ensure minimum column width.
	minWidth := r.fontSize() * 3
	for i := range colWidths {
		colWidths[i] = math.Max(colWidths[i], minWidth)
	}
	return colWidths
}
