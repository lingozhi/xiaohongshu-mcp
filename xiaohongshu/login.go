package xiaohongshu

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LoginAction struct {
	page *rod.Page
}

func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	pp := a.page.Context(ctx)

	// 检查创作者平台的登录状态（和发布视频使用同一域名）
	creatorURL := "https://creator.xiaohongshu.com/publish/publish?source=official"
	logrus.Infof("检查登录状态: %s", creatorURL)

	wait := pp.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
	if err := pp.Navigate(creatorURL); err != nil {
		return false, errors.Wrap(err, "导航到创作者平台失败")
	}
	wait()

	time.Sleep(2 * time.Second)

	// 检查是否被重定向到登录页
	currentURL := pp.MustInfo().URL
	logrus.Infof("当前页面 URL: %s", currentURL)

	if strings.Contains(currentURL, "login") {
		return false, nil
	}

	// 检查是否有上传区域（说明已登录且在发布页面）
	selectors := []string{
		"div.upload-content",
		"div.creator-tab",
		".upload-wrapper",
	}

	for _, selector := range selectors {
		exists, _, err := pp.Has(selector)
		if err != nil {
			continue
		}
		if exists {
			logrus.Infof("检测到登录状态元素: %s", selector)
			return true, nil
		}
	}

	return false, nil
}

func (a *LoginAction) Login(ctx context.Context) error {
	pp := a.page.Context(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 等待一小段时间让页面完全加载
	time.Sleep(2 * time.Second)

	// 检查是否已经登录
	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，直接返回
		return nil
	}

	// 等待扫码成功提示或者登录完成
	// 这里我们等待登录成功的元素出现，这样更简单可靠
	pp.MustElement(".main-container .user .link-wrapper .channel")

	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := a.page.Context(ctx)

	// 导航到小红书首页
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 等待页面完全加载
	time.Sleep(3 * time.Second)

	// 检查是否已经登录
	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		return "", true, nil
	}

	// 尝试点击登录按钮触发二维码弹窗
	loginBtnSelectors := []string{
		".login-btn",
		".login-button",
		"[class*='login']",
		".side-bar .login-btn",
	}
	for _, selector := range loginBtnSelectors {
		if el, err := pp.Timeout(2 * time.Second).Element(selector); err == nil && el != nil {
			if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logrus.Warnf("点击登录按钮失败: %v", err)
				continue
			}
			time.Sleep(1 * time.Second)
			break
		}
	}

	// 等待二维码出现（最多30秒）
	qrcodeSelectors := []string{
		".login-container .qrcode-img",
		".qrcode-img",
		"[class*='qrcode'] img",
		".login-modal img",
	}

	var src *string
	var err error
	for _, selector := range qrcodeSelectors {
		el, e := pp.Timeout(10 * time.Second).Element(selector)
		if e == nil && el != nil {
			src, err = el.Attribute("src")
			if err == nil && src != nil && len(*src) > 0 {
				break
			}
		}
	}

	if src == nil || len(*src) == 0 {
		return "", false, errors.New("无法获取二维码，请检查页面是否正常加载")
	}

	return *src, false, nil
}

func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	pp := a.page.Context(ctx)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logrus.Info("开始等待扫码登录...")

	// 扫码成功后页面可能出现的元素
	successSelectors := []string{
		".main-container .user .link-wrapper .channel", // 已登录用户信息
		".login-success",     // 登录成功提示
		"[class*='success']", // 包含 success 的元素
	}

	// 二维码选择器（用于检测二维码是否消失）
	qrcodeSelectors := []string{
		".qrcode-img",
		"[class*='qrcode'] img",
	}

	checkCount := 0
	for {
		select {
		case <-ctx.Done():
			logrus.Warn("等待登录超时")
			return false
		case <-ticker.C:
			checkCount++

			// 检测登录成功元素
			for _, selector := range successSelectors {
				if exists, _, _ := pp.Has(selector); exists {
					logrus.Infof("检测到登录成功元素: %s", selector)
					return true
				}
			}

			// 检测二维码是否消失（扫码成功后二维码会消失）
			qrcodeExists := false
			for _, selector := range qrcodeSelectors {
				if exists, _, _ := pp.Has(selector); exists {
					qrcodeExists = true
					break
				}
			}

			if !qrcodeExists && checkCount > 5 {
				// 二维码消失超过 10 秒，认为扫码成功
				logrus.Info("二维码已消失，认为扫码成功")
				return true
			}

			// 每 30 秒输出一次日志
			if checkCount%15 == 0 {
				logrus.Infof("等待扫码中... (已等待 %d 秒)", checkCount*2)
			}
		}
	}
}
