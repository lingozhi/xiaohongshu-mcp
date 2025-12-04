package cookies

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Cookier interface {
	LoadCookies() ([]byte, error)
	SaveCookies(data []byte) error
	DeleteCookies() error
}

type localCookie struct {
	path string
}

func NewLoadCookie(path string) Cookier {
	if path == "" {
		panic("path is required")
	}

	return &localCookie{
		path: path,
	}
}

// LoadCookies 从文件中加载 cookies。
func (c *localCookie) LoadCookies() ([]byte, error) {
	logrus.Infof("加载 cookies: %s", c.path)
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, errors.Wrapf(err, "读取 cookies 失败: %s", c.path)
	}

	return data, nil
}

// SaveCookies 保存 cookies 到文件中。
func (c *localCookie) SaveCookies(data []byte) error {
	logrus.Infof("保存 cookies: %s", c.path)
	// 确保父目录存在
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "failed to create cookies directory")
	}
	return os.WriteFile(c.path, data, 0644)
}

// DeleteCookies 删除 cookies 文件。
func (c *localCookie) DeleteCookies() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		// 文件不存在，返回 nil（认为已经删除）
		return nil
	}
	return os.Remove(c.path)
}

// GetCookiesFilePath 获取 cookies 文件路径。
// 优先使用环境变量 COOKIES_PATH，否则使用当前目录下的 cookies.json
func GetCookiesFilePath() string {
	path := os.Getenv("COOKIES_PATH")
	if path == "" {
		path = "cookies.json"
	}
	logrus.Debugf("cookies 路径: %s", path)
	return path
}
