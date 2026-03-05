// Package codec 提供图片格式识别和微信 .dat 文件解密功能。
package codec

import (
	"errors"
	"fmt"
)

// Format 表示图片格式。
type Format string

const (
	FormatPNG     Format = "png"
	FormatJPEG    Format = "jpg"
	FormatGIF     Format = "gif"
	FormatWEBP    Format = "webp"
	FormatUnknown Format = ""
)

// 各图片格式的文件头魔数
var magics = []struct {
	magic  []byte
	format Format
}{
	{[]byte{0x89, 0x50, 0x4E, 0x47}, FormatPNG},  // PNG
	{[]byte{0xFF, 0xD8, 0xFF}, FormatJPEG},        // JPEG
	{[]byte{0x47, 0x49, 0x46}, FormatGIF},         // GIF
	{[]byte{0x52, 0x49, 0x46, 0x46}, FormatWEBP},  // WEBP（RIFF 头）
}

// DetectFormat 通过文件头魔数识别图片格式。
func DetectFormat(data []byte) Format {
	if len(data) < 4 {
		return FormatUnknown
	}
	for _, m := range magics {
		if len(data) >= len(m.magic) {
			match := true
			for i, b := range m.magic {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				// WEBP 在偏移 8 处还需校验 "WEBP" 标识
				if m.format == FormatWEBP {
					if len(data) >= 12 && string(data[8:12]) == "WEBP" {
						return FormatWEBP
					}
					continue
				}
				return m.format
			}
		}
	}
	return FormatUnknown
}

// DecryptWechatDat 解密微信 .dat 文件，返回原始图片字节和格式。
// 微信对图片每个字节做单字节 XOR 加密，密钥通过文件首字节与已知魔数异或推导。
func DecryptWechatDat(data []byte) ([]byte, Format, error) {
	if len(data) == 0 {
		return nil, FormatUnknown, errors.New("数据为空")
	}

	key, format := findXORKey(data[0])
	if format == FormatUnknown {
		return nil, FormatUnknown, fmt.Errorf("无法从首字节 0x%02X 推导 XOR 密钥", data[0])
	}

	decrypted := make([]byte, len(data))
	for i, b := range data {
		decrypted[i] = b ^ key
	}

	// 验证解密后的数据与预期格式一致
	if DetectFormat(decrypted) != format {
		return nil, FormatUnknown, fmt.Errorf("解密校验失败，格式：%s", format)
	}

	return decrypted, format, nil
}

// findXORKey 用加密首字节与各格式魔数逐一异或，推导出 XOR 密钥。
func findXORKey(encryptedByte byte) (byte, Format) {
	candidates := []struct {
		firstByte byte
		format    Format
	}{
		{0x89, FormatPNG},
		{0xFF, FormatJPEG},
		{0x47, FormatGIF},
		{0x52, FormatWEBP},
	}
	for _, c := range candidates {
		key := encryptedByte ^ c.firstByte
		if key != 0 {
			return key, c.format
		}
	}
	return 0, FormatUnknown
}
