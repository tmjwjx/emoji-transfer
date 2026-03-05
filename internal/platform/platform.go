// Package platform 定义所有平台适配器必须实现的统一接口。
package platform

import "github.com/world-fish/emoji-transfer/internal/store"

// Platform 是每个平台适配器必须实现的接口。
type Platform interface {
	// Name 返回平台标识符，如 "wechat"、"qq"。
	Name() string

	// Detect 检测该平台是否已安装且表情缓存目录可访问。
	Detect() (bool, error)

	// Export 读取平台表情缓存并保存到本地表情库，返回新增数量。
	Export(s *store.Store) (int, error)

	// Import 将本地表情库中的表情注入目标平台。
	Import(emojis []*store.Emoji) error
}
