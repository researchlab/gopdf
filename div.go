package gopdf

import (
	"math"
	"strings"

	"github.com/researchlab/gopdf/core"
	"github.com/researchlab/gopdf/util"
)

const (
	DIV_STRAIGHT = 1 // 实线边框
	DIV_DASHED   = 2 // 虚线边框
	DIV_DOTTED   = 3 // 点状线的边框
	DIV_NONE     = 4 // 无边框
)

// 带有各种边框的内容, 可以自动换行
type Div struct {
	pdf       *core.Report
	font      core.Font
	frameType int // 边框类型, 默认是无边框
	contents  []string

	width, height float64
	lineHeight    float64
	lineSpace     float64

	fontColor string // 字体颜色
	backColor string // 背景颜色

	margin core.Scope
	border core.Scope

	horizontalCentered bool // 水平居中
	rightAlign         bool // 局右显示, 默认是居左显示
}

func NewDiv(lineHeight, lineSpce float64, pdf *core.Report) *Div {
	currX, _ := pdf.GetXY()
	endX := pdf.GetPageEndX()
	if endX-currX <= 0 {
		panic("please modify current X")
	}

	f := &Div{
		pdf:        pdf,
		frameType:  DIV_NONE,
		width:      endX - currX,
		height:     lineHeight,
		lineHeight: lineHeight,
		lineSpace:  lineSpce,
	}

	return f
}

func NewDivWithWidth(width float64, lineHeight, lineSpce float64, pdf *core.Report) *Div {
	currX, _ := pdf.GetXY()
	endX := pdf.GetPageEndX()
	if endX-currX <= 0 {
		panic("please modify current X")
	}

	if endX-currX <= width {
		width = endX - currX
	}

	f := &Div{
		pdf:        pdf,
		frameType:  DIV_NONE,
		width:      width,
		height:     lineHeight,
		lineHeight: lineHeight,
		lineSpace:  lineSpce,
	}

	return f
}

func (div *Div) Copy(content string) *Div {
	f := &Div{
		pdf:        div.pdf,
		frameType:  div.frameType,
		width:      div.width,
		lineHeight: div.lineHeight,
		lineSpace:  div.lineSpace,
		fontColor:  div.fontColor,
		backColor:  div.backColor,
	}

	f.SetBorder(div.border)
	f.SetFont(div.font)
	f.SetContent(content)

	return f
}

func (div *Div) SetFrameType(frameType int) *Div {
	if frameType < DIV_STRAIGHT || frameType > DIV_NONE {
		return div
	}

	div.frameType = frameType

	return div
}

func (div *Div) SetMarign(margin core.Scope) *Div {
	margin.ReplaceMarign()
	currX, _ := div.pdf.GetXY()
	endX := div.pdf.GetPageEndX()

	if endX-(currX+margin.Left) <= 0 {
		panic("the marign out of page boundary")
	}

	// 宽度检测
	if endX-(currX+margin.Left) <= div.width {
		div.width = endX - (currX + margin.Left)
	}

	div.margin = margin

	return div
}
func (div *Div) SetBorder(border core.Scope) *Div {
	border.ReplaceBorder()
	currX, _ := div.pdf.GetXY()
	endX := div.pdf.GetPageEndX()

	// 最大宽度检测
	if endX-(currX+div.margin.Left) >= div.width+border.Left+border.Right {
		div.width += border.Left + border.Right
	}

	div.border = border

	return div
}

func (div *Div) GetHeight() (height float64) {
	return div.height
}
func (div *Div) GetWidth() (width float64) {
	return div.width
}

func (div *Div) HorizontalCentered() *Div {
	div.horizontalCentered = true
	div.rightAlign = false
	return div
}
func (div *Div) RightAlign() *Div {
	div.rightAlign = true
	div.horizontalCentered = false
	return div
}

func (div *Div) SetFontColor(color string) *Div {
	util.CheckColor(color)
	div.fontColor = color
	return div
}
func (div *Div) SetBackColor(color string) *Div {
	util.CheckColor(color)
	div.backColor = color
	return div
}

func (div *Div) SetFont(font core.Font) *Div {
	div.font = font
	// 注册, 启动
	div.pdf.Font(font.Family, font.Size, font.Style)
	div.pdf.SetFontWithStyle(font.Family, font.Style, font.Size)

	return div
}
func (div *Div) SetFontWithColor(font core.Font, color string) *Div {
	div.SetFont(font)
	div.SetFontColor(color)
	return div
}

func (div *Div) SetContent(content string) *Div {
	convertStr := strings.Replace(content, "\t", "    ", -1)

	var (
		unit         = div.pdf.GetUnit()
		blocks       = strings.Split(convertStr, "\n") // 分行
		contentWidth = div.width - math.Abs(div.border.Left) - math.Abs(div.border.Right)
	)

	// 必须检查字体
	if util.IsEmpty(div.font) {
		panic("there no avliable font")
	}

	// 必须先进行注册, 才能设置
	div.pdf.Font(div.font.Family, div.font.Size, div.font.Style)
	div.pdf.SetFontWithStyle(div.font.Family, div.font.Style, div.font.Size)
	if len(blocks) == 1 {
		if div.pdf.MeasureTextWidth(convertStr)/unit < contentWidth {
			div.contents = []string{convertStr}
			div.height = math.Abs(div.border.Top) + math.Abs(div.border.Bottom) + div.lineHeight
			return div
		}
	}

	for i := range blocks {
		// 单独的一行
		if div.pdf.MeasureTextWidth(convertStr)/unit < contentWidth {
			div.contents = append(div.contents, blocks[i])
			continue
		}

		var (
			line []rune
		)
		// 单独的一行需要拆分
		for _, r := range []rune(blocks[i]) {
			line = append(line, r)
			lineLength := div.pdf.MeasureTextWidth(string(line))
			if lineLength/unit >= contentWidth {
				if lineLength-contentWidth/unit > unit*2 {
					div.contents = append(div.contents, string(line[0:len(line)-1]))
					line = line[len(line)-1:]
				} else {
					div.contents = append(div.contents, string(line))
					line = []rune{}
				}
			}
		}

		// 剩余单独成行
		if len(line) > 0 {
			div.contents = append(div.contents, string(line))
		}
	}

	// 重新计算 div 的高度
	length := float64(len(div.contents))
	div.height = div.border.Top + div.lineHeight*length + div.lineSpace*(length-1)

	return div
}

// 自动分页
func (div *Div) GenerateAtomicCell() error {
	var (
		sx, sy   = div.pdf.GetXY()
		x, y     float64
		border   core.Scope
		pageEndY = div.pdf.GetPageEndY()
	)

	if util.IsEmpty(div.font) {
		panic("no font")
	}

	switch div.frameType {
	case DIV_STRAIGHT:
		div.pdf.LineType("straight", 0.01)
	case DIV_DASHED:
		div.pdf.LineType("dashed", 0.01)
	case DIV_DOTTED:
		div.pdf.LineType("dotted", 0.01)
	}

	div.drawLine(sx, sy)
	div.pdf.Font(div.font.Family, div.font.Size, div.font.Style)
	div.pdf.SetFontWithStyle(div.font.Family, div.font.Style, div.font.Size)
	border = div.border

	for i := 0; i < len(div.contents); i++ {
		// 水平居中, 只是对当前的行设置新的 Border
		if div.horizontalCentered {
			width := div.pdf.MeasureTextWidth(div.contents[i]) / div.pdf.GetUnit()
			if width < div.width {
				left := (div.width - width) / 2
				div.border = core.NewScope(left, border.Top, 0, border.Right)
			}
		}

		// 水平居右, 只是对当前的行设置新的 Border
		if div.rightAlign {
			width := div.pdf.MeasureTextWidth(div.contents[i]) / div.pdf.GetUnit()
			left := div.width - width
			div.border = core.NewScope(left, border.Top, 0, border.Right)
		}

		x, y = div.getContentPosition(sx, sy, i)

		// 换页
		if y+div.lineHeight > pageEndY {
			var newX, newY float64

			div.SetMarign(core.NewScope(div.margin.Left, 0, div.margin.Right, 0))
			div.SetBorder(core.NewScope(border.Left, div.lineSpace, border.Right, 0))
			div.contents = div.contents[i:]
			div.resetHeight()

			_, newY = div.pdf.GetPageStartXY()
			if len(div.contents) > 0 {
				newX, _ = div.pdf.GetXY()
			} else {
				newX, _ = div.pdf.GetPageStartXY()
			}

			div.pdf.AddNewPage(false)
			div.pdf.SetXY(newX, newY)

			return div.GenerateAtomicCell()
		}

		if !util.IsEmpty(div.fontColor) {
			div.pdf.TextColor(util.GetColorRGB(div.fontColor))
		}
		div.pdf.Font(div.font.Family, div.font.Size, div.font.Style) // 添加设置
		div.pdf.Cell(x, y, div.contents[i])
		if !util.IsEmpty(div.fontColor) {
			div.pdf.TextDefaultColor()
		}
	}

	x, _ = div.pdf.GetPageStartXY()
	div.pdf.SetXY(x, y+div.lineHeight+div.margin.Bottom) // 定格最终的位置

	return nil
}

func (div *Div) drawLine(sx, sy float64) {
	var (
		x, y     float64
		pageEndY = div.pdf.GetPageEndY()
	)

	if sy+div.height > pageEndY {
		x, y = sx+div.margin.Left, sy+div.margin.Top
		if !util.IsEmpty(div.backColor) {
			div.pdf.BackgroundColor(x, y, div.width, pageEndY-y, div.backColor, "0000")
		}

		y = sy + div.margin.Top
		// 两条竖线 + 一条横线
		if div.frameType != DIV_NONE {
			div.pdf.LineV(sx+div.margin.Left, y, pageEndY)
			div.pdf.LineV(sx+div.margin.Left+div.width, y, pageEndY)

			div.pdf.LineH(sx+div.margin.Left, y, sx+div.margin.Left+div.width)
			div.pdf.LineH(sx+div.margin.Left, pageEndY, sx+div.margin.Left+div.width)
		}

	} else {
		x, y = sx+div.margin.Left, sy+div.margin.Top
		if !util.IsEmpty(div.backColor) {
			div.pdf.BackgroundColor(x, y, div.width, div.height, div.backColor, "0000")
		}

		y = sy + div.margin.Top
		// 两条竖线 + 一条横线
		if div.frameType != DIV_NONE {
			div.pdf.LineV(sx+div.margin.Left, y, y+div.height)
			div.pdf.LineV(sx+div.margin.Left+div.width, y, y+div.height)

			div.pdf.LineH(sx+div.margin.Left, y, sx+div.margin.Left+div.width)
			div.pdf.LineH(sx+div.margin.Left, y+div.height, sx+div.margin.Left+div.width)
		}
	}
}

func (div *Div) resetHeight() {
	if len(div.contents) == 0 {
		div.height = 0
	}
	length := float64(len(div.contents))
	div.height = div.lineHeight*length + div.lineSpace*(length-1) + div.border.Top
}

func (div *Div) getContentPosition(sx, sy float64, index int) (x, y float64) {
	x = sx + div.margin.Left + div.border.Left
	y = sy + div.margin.Top + div.border.Top

	y += float64(index) * (div.lineHeight + div.lineSpace)

	return x, y
}
