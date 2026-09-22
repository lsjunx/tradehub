package identity

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// LoadOrCreate 读取 path 中的 ID；不存在或为空则生成 UUID 并写入。
func LoadOrCreate(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if id != "" {
			return id, nil
		}
	}
	id := uuid.NewString()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o644); err != nil {
		return "", err
	}
	return id, nil
}
