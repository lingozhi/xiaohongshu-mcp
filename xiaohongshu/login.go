package xiaohongshu

import (
	"context"
	"time"

	"github.com/go-rod/rod"
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
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	time.Sleep(1 * time.Second)

	exists, _, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	if !exists {
		return false, errors.Wrap(err, "login status element not found")
	}

	return true, nil
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
			_ = el.MustClick()
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

	// 二维码选择器（与 FetchQrcodeImage 保持一致）
	qrcodeSelectors := []string{
		".login-container .qrcode-img",
		".qrcode-img",
		"[class*='qrcode'] img",
		".login-modal img",
	}

	// 登录成功的选择器
	loginSuccessSelectors := []string{
		".main-container .user .link-wrapper .channel",
		".user .avatar",
		".sidebar .user-info",
		".side-bar .user",
	}

	checkCount := 0
	qrcodeDisappeared := false

	for {
		select {
		case <-ctx.Done():
			logrus.Warn("等待登录超时")
			return false
		case <-ticker.C:
			checkCount++

			// 检测二维码是否存在
			qrcodeExists := false
			for _, selector := range qrcodeSelectors {
				if exists, _, _ := pp.Has(selector); exists {
					qrcodeExists = true
					break
				}
			}

			// 二维码消失时刷新页面（只刷新一次）
			if !qrcodeExists && !qrcodeDisappeared {
				qrcodeDisappeared = true
				logrus.Info("二维码已消失，刷新页面检测登录状态...")
				pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()
				time.Sleep(2 * time.Second)
			}

			// 检测登录成功元素
			for _, selector := range loginSuccessSelectors {
				if exists, _, _ := pp.Has(selector); exists {
					logrus.Info("检测到登录成功")
					return true
				}
			}

			// 每 10 次检测输出一次日志
			if checkCount%10 == 0 {
				logrus.Infof("等待登录中... (已检测 %d 次)", checkCount)
			}
		}
	}
}
