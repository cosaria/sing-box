package tui

import (
	"strings"

	"github.com/skip2/go-qrcode"
)

// RenderQR 将字符串内容渲染为终端 Unicode 半块字符 QR 码。
// 使用上半块（▀）、下半块（▄）、全块（█）和空格来压缩高度，
// 每两行像素合并为一行字符输出。
// 若 content 为空或生成失败，返回空字符串。
func RenderQR(content string) string {
	if content == "" {
		return ""
	}
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return ""
	}
	bitmap := q.Bitmap()
	var sb strings.Builder
	for y := 0; y < len(bitmap); y += 2 {
		for x := 0; x < len(bitmap[y]); x++ {
			top := bitmap[y][x]
			bot := y+1 < len(bitmap) && bitmap[y+1][x]
			switch {
			case top && bot:
				sb.WriteRune('█')
			case top:
				sb.WriteRune('▀')
			case bot:
				sb.WriteRune('▄')
			default:
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}
