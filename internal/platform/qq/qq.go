// Package qq 实现 QQ 平台适配器（macOS）。
package qq

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/world-fish/emoji-transfer/internal/codec"
	"github.com/world-fish/emoji-transfer/internal/store"
)

const (
	name      = "qq"
	sandboxID = "com.tencent.qq"
)

// Platform 是 QQ 适配器。
type Platform struct{}

func New() *Platform { return &Platform{} }

func (p *Platform) Name() string { return name }

// Detect 检测 QQ 是否已安装。
func (p *Platform) Detect() (bool, error) {
	root, err := sandboxRoot()
	if err != nil {
		return false, err
	}
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// Export 读取 QQ 自定义表情目录（无加密，直接是标准图片文件）。
func (p *Platform) Export(s *store.Store) (int, error) {
	dirs, err := customFaceDirs()
	if err != nil {
		return 0, err
	}
	if len(dirs) == 0 {
		return 0, fmt.Errorf("no QQ custom face directories found; make sure QQ is installed and has been used")
	}

	added := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if !strings.HasSuffix(name, ".png") &&
				!strings.HasSuffix(name, ".gif") &&
				!strings.HasSuffix(name, ".jpg") &&
				!strings.HasSuffix(name, ".webp") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			format := codec.DetectFormat(data)
			if format == codec.FormatUnknown {
				continue
			}
			isNew, _, err := s.Add(data, format, "qq")
			if err != nil {
				return added, fmt.Errorf("store add: %w", err)
			}
			if isNew {
				added++
			}
		}
	}
	return added, nil
}

// Import QQ 导入暂未实现。
func (p *Platform) Import(emojis []*store.Emoji) error {
	return fmt.Errorf("QQ 导入功能暂未实现")
}

func sandboxRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Containers", sandboxID,
		"Data", "Library", "Application Support", "QQ"), nil
}

// customFaceDirs 枚举所有 QQ 账号下的 CustomFace 目录。
func customFaceDirs() ([]string, error) {
	root, err := sandboxRoot()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read QQ dir: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		faceDir := filepath.Join(root, entry.Name(), "CustomFace")
		if info, err := os.Stat(faceDir); err == nil && info.IsDir() {
			dirs = append(dirs, faceDir)
		}
	}
	return dirs, nil
}
