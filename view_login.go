package main

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// loginModal asks for what the login command waits on.
func (a *app) loginModal(c *ui.Context, pal palette) {
	t := c.Theme()
	f := &a.login
	m := ui.Modal(c, &f.open, func() {
		ui.Column(c).Width(440).Padding(28).Radius(22).Gap(20).
			Material(glass.Glass{}).Shadow(0, 24, 60, 4, pal.shadow).Children(func() {
			ui.Row(c).Gap(13).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(34, 34).Shrink(0).Radius(10).Background(t.Accent).Center().
					Children(func() {
						ui.Icon(c, iconLogo).FontSize(17).TextColor(t.AccentText)
					})
				ui.Column(c).Gap(3).Grow(1).Children(func() {
					ui.Text(c, "登录 Telegram").FontSize(17).FontWeight(600)
					ui.Text(c, f.status).FontSize(sizeMeta).TextColor(t.TextMuted).SingleLine()
				})
			})
			a.loginStep(c, pal)
			if f.errText != "" {
				ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Icon(c, iconAlert).FontSize(13).TextColor(pal.danger)
					ui.Text(c, f.errText).FontSize(sizeMeta).TextColor(pal.danger).Grow(1)
				})
			}
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				hint(c, pal, "会话保存在 tdl 自己的数据目录里")
				ui.Spacer(c)
				secondaryButton(c, "关闭").OnClick(func() { a.closeLogin() })
			})
		})
	})
	if m.Dismissed() {
		a.closeLogin()
	}
}

func (a *app) loginStep(c *ui.Context, pal palette) {
	t := c.Theme()
	f := &a.login
	switch f.step {
	case "phone":
		in := ui.TextInput(c, &f.phone).Placeholder("+8613800138000").FontSize(sizeBody).AutoFocus()
		if in.Submitted() {
			a.submitLogin()
		}
		primaryButton(c, "发送验证码").FillWidth().OnClick(func() { a.submitLogin() })
	case "code":
		in := ui.TextInput(c, &f.code).Placeholder("Telegram 收到的验证码").FontSize(sizeBody).AutoFocus()
		if in.Submitted() {
			a.submitLogin()
		}
		primaryButton(c, "验证").FillWidth().OnClick(func() { a.submitLogin() })
	case "twofactor":
		in := ui.TextInput(c, &f.twoFactor).Placeholder("两步验证密码").FontSize(sizeBody).AutoFocus()
		if in.Submitted() {
			a.submitLogin()
		}
		primaryButton(c, "验证").FillWidth().OnClick(func() { a.submitLogin() })
	case "done":
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconCheck).FontSize(15).TextColor(pal.ok)
			ui.Text(c, "已登录 "+f.Account.label()).FontSize(sizeBody).Grow(1)
		})
	case "failed":
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconAlert).FontSize(15).TextColor(pal.danger)
			ui.Text(c, "登录没有完成").FontSize(sizeBody).Grow(1)
		})
		primaryButton(c, "重新登录").FillWidth().OnClick(func() {
			a.closeLogin()
			a.openLogin()
		})
	default:
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(15, 15)
			ui.Text(c, "正在与 Telegram 建立连接…").FontSize(sizeMeta).TextColor(t.TextMuted)
		})
	}
}
