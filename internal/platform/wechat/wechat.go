// Package wechat 实现微信平台适配器（macOS）。
package wechat

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/world-fish/emoji-transfer/internal/codec"
	"github.com/world-fish/emoji-transfer/internal/store"
)

const (
	name      = "wechat"
	sandboxID = "com.tencent.xinWeChat"
)

// Platform 是微信适配器。
type Platform struct{}

func New() *Platform { return &Platform{} }

func (p *Platform) Name() string { return name }

// Detect 通过检查沙盒目录是否存在来判断微信是否已安装。
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

// Export 遍历所有微信账号目录，解密 .dat 文件并保存到本地表情库。
func (p *Platform) Export(s *store.Store) (int, error) {
	dirs, err := emojiDirs()
	if err != nil {
		return 0, err
	}
	if len(dirs) == 0 {
		return 0, fmt.Errorf("no WeChat emoji directories found; make sure WeChat is installed and has been used")
	}

	added := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dat") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			decrypted, format, err := codec.DecryptWechatDat(data)
			if err != nil {
				// 无法解密的文件跳过（可能不是表情文件）
				continue
			}
			isNew, _, err := s.Add(decrypted, format, name)
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

// Import 通过 AppleScript 将表情注入微信（仅 macOS）。
// 执行前微信必须已在运行。
func (p *Platform) Import(emojis []*store.Emoji) error {
	if len(emojis) == 0 {
		return nil
	}

	// 逐张通过 AppleScript 触发微信打开图片文件。
	// 微信 macOS 未暴露直接"添加表情"的 AppleScript 接口，
	// 这里采用 File > Open 作为尽力而为的方案，实际效果需真机验证。
	for _, e := range emojis {
		script := fmt.Sprintf(`
tell application "WeChat"
	activate
end tell
delay 0.5
tell application "System Events"
	tell process "WeChat"
		open POSIX file %q
	end tell
end tell
`, e.Path)

		cmd := exec.Command("osascript", "-e", script)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("applescript failed for %s: %w\n%s", e.Path, err, out)
		}
	}
	return nil
}

// sandboxRoot 返回微信 macOS 沙盒根目录路径。
func sandboxRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Containers", sandboxID,
		"Data", "Library", "Application Support", sandboxID), nil
}

// emojiDirs 枚举所有微信账号下的 Emoji 目录。
// 账号目录名为哈希字符串，需遍历探测。
func emojiDirs() ([]string, error) {
	root, err := sandboxRoot()
	if err != nil {
		return nil, err
	}

	accountsDir := filepath.Join(root, "Accounts")
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read accounts dir: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		emojiDir := filepath.Join(accountsDir, entry.Name(), "Emoji")
		if info, err := os.Stat(emojiDir); err == nil && info.IsDir() {
			dirs = append(dirs, emojiDir)
		}
	}
	return dirs, nil
}
