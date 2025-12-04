package browser

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
)

// Browser 浏览器包装
type Browser struct {
	browser *rod.Browser
}

type browserConfig struct {
	binPath string
}

type Option func(*browserConfig)

func WithBinPath(binPath string) Option {
	return func(c *browserConfig) {
		c.binPath = binPath
	}
}

// NewBrowser 创建浏览器实例
func NewBrowser(headless bool, options ...Option) *Browser {
	cfg := &browserConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	// 使用 go-rod launcher 配置浏览器启动参数
	l := launcher.New()

	// 设置浏览器路径
	if cfg.binPath != "" {
		l = l.Bin(cfg.binPath)
	}

	// 设置无头模式
	l = l.Headless(headless)

	// 容器环境必须禁用沙箱
	if os.Getenv("ROD_NO_SANDBOX") != "" || isContainerEnv() {
		l = l.NoSandbox(true)
	}

	// 禁用 /dev/shm 使用（容器环境需要）
	l = l.Set("disable-dev-shm-usage")

	controlURL := l.MustLaunch()
	rodBrowser := rod.New().ControlURL(controlURL).MustConnect()

	// 加载 cookies
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)

	if data, err := cookieLoader.LoadCookies(); err == nil {
		if err := setCookiesFromJSON(rodBrowser, string(data)); err != nil {
			logrus.Warnf("failed to set cookies: %v", err)
		} else {
			logrus.Debugf("loaded cookies from file successfully")
		}
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}

	return &Browser{browser: rodBrowser}
}

// NewPage 创建新页面（带 stealth 模式）
func (b *Browser) NewPage() *rod.Page {
	page := stealth.MustPage(b.browser)
	return page
}

// Close 关闭浏览器
func (b *Browser) Close() {
	if b.browser != nil {
		_ = b.browser.Close()
	}
}

// isContainerEnv 检测是否在容器环境中
func isContainerEnv() bool {
	// 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// 检查 cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "kubepods") || strings.Contains(content, "containerd") {
			return true
		}
	}
	return false
}

// setCookiesFromJSON 从 JSON 字符串设置 cookies
func setCookiesFromJSON(b *rod.Browser, jsonStr string) error {
	var cookies []*proto.NetworkCookieParam
	if err := json.Unmarshal([]byte(jsonStr), &cookies); err != nil {
		return err
	}
	return b.SetCookies(cookies)
}
