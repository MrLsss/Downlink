package main

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// The window is a frame with one page in it. Type has four sizes: the page
// title, the sentence under it, the words that carry content, and the small
// print that qualifies them.
const (
	sizeDisplay = 34
	sizeLead    = 15
	sizeBody    = 13
	sizeMeta    = 11.5
)

// view builds a frame of the window from the app's state.
func (a *app) view(c *ui.Context) {
	t := c.Theme()
	pal := paletteFor(t)
	root := c.Root().Background(t.Background)
	if c.Vibrancy() {
		// The window material is the ground; the page must not cover it.
		root.Background(ui.Transparent)
	}
	ui.Row(c).Fill().Children(func() {
		a.sidebar(c, pal)
		a.main(c, pal)
	})
	a.loginModal(c, pal)
	a.optionsModal(c, pal)
}

func (a *app) sidebar(c *ui.Context, pal palette) {
	t := c.Theme()
	ui.Column(c).Width(248).FillHeight().Padding(0, 16, 18, 16).Background(pal.scrim).Children(func() {
		ui.Box(c).Height(52).Shrink(0).DragWindow()
		ui.Row(c).Gap(10).AlignItems(ui.Center).Shrink(0).Padding(6, 8, 26, 8).Children(func() {
			ui.Box(c).Size(30, 30).Shrink(0).Radius(9).Background(t.Accent).Center().Children(func() {
				ui.Icon(c, iconLogo).FontSize(17).TextColor(t.AccentText)
			})
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, "Downlink").FontSize(15).FontWeight(600).LetterSpacing(0.2)
				ui.Text(c, "Telegram 下载器").FontSize(sizeMeta).TextColor(pal.muted)
			})
		})
		ui.Column(c).Gap(2).Shrink(0).Children(func() {
			a.navRow(c, "download", "下载", iconDownload, pal)
			a.navRow(c, "tasks", "记录", iconHistory, pal)
			a.navRow(c, "settings", "设置", iconSettings, pal)
		})
		ui.Spacer(c)
		a.accountCard(c, pal)
	})
}

func (a *app) navRow(c *ui.Context, id, label string, ic *ui.SVG, pal palette) {
	t := c.Theme()
	selected := a.page == id
	row := ui.ButtonBase(c.Key("nav-"+id)).FillWidth().Height(44).Radius(10).Padding(0, 10).Gap(10).
		AlignItems(ui.Center).Justify(ui.Start).Label(label)
	switch {
	case selected:
		row.Background(pal.selected)
	case row.Hovered():
		row.Background(pal.hover)
	}
	ink := t.TextMuted
	weight := 400
	if selected {
		ink = t.Text
		weight = 600
	}
	row.Children(func() {
		ui.Icon(c, ic).FontSize(16).TextColor(ink)
		ui.Text(c, label).FontSize(sizeBody).FontWeight(weight).TextColor(ink)
	})
	row.OnClick(func() { a.page = id })
}

func (a *app) accountCard(c *ui.Context, pal palette) {
	label, note, dot, action := "未登录", "还没有 tdl 会话", pal.muted, "登录 Telegram"
	switch {
	case a.cfg.Account.Username != "" || a.cfg.Account.ID != 0:
		label, note, dot, action = a.cfg.Account.label(), "tdl 会话 · "+a.cfg.Namespace, pal.ok, "重新登录"
	case hasSession(a.cfg.Namespace):
		label, note, dot, action = "已登录", "tdl 会话 · "+a.cfg.Namespace, pal.ok, "重新登录"
	}
	ui.Column(c).FillWidth().Shrink(0).Gap(3).Padding(8, 10, 4, 10).Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(dot)
			ui.Text(c, label).FontSize(sizeBody).FontWeight(500).SingleLine().Grow(1)
		})
		ui.Text(c, note).FontSize(sizeMeta).TextColor(pal.muted).SingleLine()
		ui.Row(c).Padding(8, 0, 0, 0).Children(func() {
			secondaryButton(c, action).FillWidth().OnClick(func() { a.openLogin() })
		})
	})
}

func (a *app) main(c *ui.Context, pal palette) {
	ui.Box(c).Grow(1).FillHeight().Background(c.Theme().Background).Children(func() {
		switch a.page {
		case "tasks":
			a.tasksPage(c, pal)
		case "settings":
			a.settingsPage(c, pal)
		default:
			a.downloadPage(c, pal)
		}
		a.topBar(c)
	})
}

func (a *app) topBar(c *ui.Context) {
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(46).PassThrough().
		Material(glass.Blur{
			Radius: 6,
			Mask:   &ui.LinearGradient{From: ui.RGBA(0, 0, 0, 1), To: ui.Transparent, Angle: 180},
		})
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(46).DragWindow()
}

// scrolled wraps a page's content in the column that scrolls.
func (a *app) scrolled(c *ui.Context, key string, fn func()) {
	ui.Scroll(c).Key("page-"+key).Fill().Padding(46, 40, 44, 40).ScrollbarInsets(46, 0, 0).Children(func() {
		ui.Column(c).FillWidth().Gap(34).Children(fn)
	})
}

// header is the page's title, the sentence that says what the page is for,
// and what the last action had to say.
func (a *app) header(c *ui.Context, pal palette, title, subtitle string) {
	ui.Column(c).Gap(8).FillWidth().Shrink(0).Children(func() {
		ui.Text(c, title).FontSize(sizeDisplay).FontWeight(600).LetterSpacing(-0.8)
		ui.Text(c, subtitle).FontSize(sizeLead).TextColor(pal.muted)
		a.noticeRow(c, pal)
	})
}

// noticeRow shows what the last action had to say, in the colour that says
// whether it worked.
func (a *app) noticeRow(c *ui.Context, pal palette) {
	if a.notice == "" {
		return
	}
	ink := pal.muted
	if a.noticeBad {
		ink = pal.warn
	}
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(ink)
		ui.Text(c, a.notice).FontSize(sizeMeta).TextColor(ink)
	})
}

// glassButton is a pill of glass: it keeps the keyboard focus a button has,
// and the glass grows a little under the pointer.
// secondaryButton is every action that is not the page's main one: a pill of
// the chip tone, with no edge, that answers the pointer as soon as it lands.
func secondaryButton(c *ui.Context, label string) ui.Element {
	pal := paletteFor(c.Theme())
	b := ui.ButtonBase(c).Height(34).Padding(0, 16).Radius(17).Label(label).Background(pal.chip)
	switch {
	case b.Pressed():
		b.Background(pal.selected)
	case b.Hovered():
		b.Background(pal.hover)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(12.5)
	})
	return b
}

// primaryButton is the one action the page asks for, filled with the accent.
func primaryButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Height(40).Padding(0, 22).Radius(20).Label(label).Background(t.Accent)
	switch {
	case b.Pressed():
		b.Background(t.AccentPressed)
	case b.Hovered():
		b.Background(t.AccentHover)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(sizeBody).FontWeight(600).TextColor(t.AccentText)
	})
	return b
}

func quietLabel(c *ui.Context, pal palette, s string) {
	ui.Text(c, s).FontSize(sizeMeta).FontWeight(600).TextColor(pal.muted)
}

func fieldLabel(c *ui.Context, t *ui.Theme, s string) {
	ui.Text(c, s).FontSize(sizeMeta).TextColor(t.TextMuted)
}

func hint(c *ui.Context, pal palette, s string) {
	ui.Text(c, s).FontSize(sizeMeta).TextColor(pal.muted)
}

func (a *app) downloadPage(c *ui.Context, pal palette) {
	ui.Scroll(c).Key("page-download").Fill().Padding(46, 40, 44, 40).ScrollbarInsets(46, 0, 0).
		Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(20).Children(func() {
					ui.Column(c).Grow(1).Gap(8).Children(func() {
						ui.Text(c, "下载").FontSize(sizeDisplay).FontWeight(600).LetterSpacing(-0.8)
						ui.Text(c, "粘贴 Telegram 消息链接，选好保存位置，剩下的交给 tdl。").
							FontSize(sizeLead).TextColor(pal.muted)
						a.noticeRow(c, pal)
					})
					secondaryButton(c, "下载选项").OnClick(func() { a.showAdvanced = true })
				})
				if a.job != nil {
					ui.Column(c).Padding(26, 0, 0, 0).Children(func() { a.jobCard(c, pal) })
				}
				ui.Box(c).Height(28).Shrink(0)
				a.composer(c, pal)
			})
		})
}

// composer takes the links and starts the download: the one place the page
// asks something of the user.
func (a *app) composer(c *ui.Context, pal palette) {
	t := c.Theme()
	running := a.job != nil && a.job.running
	n := len(links(a.links))
	ui.Column(c).FillWidth().Shrink(0).Gap(0).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, "链接").FontSize(sizeMeta).FontWeight(600).TextColor(pal.muted)
			if n > 0 {
				ui.Text(c, itoa(int64(n))+" 个").FontSize(sizeMeta).TextColor(pal.muted)
			}
			ui.Spacer(c)
			secondaryButton(c, "粘贴").OnClick(func() {
				if s := strings.TrimSpace(c.ReadClipboard()); s != "" {
					a.links = s
				}
			})
			secondaryButton(c, "清空").Disabled(a.links == "").OnClick(func() { a.links = "" })
		})
		ui.Box(c).Height(10).Shrink(0)
		ui.Column(c).Height(fieldHeight(c, a.job != nil)).Radius(16).Padding(16).Background(pal.chip).Children(func() {
			ui.TextAreaBase(c, &a.links).Grow(1).FontSize(sizeBody).
				Placeholder("https://t.me/example/123\nhttps://t.me/c/1234567890/456")
		})
		ui.Box(c).Height(18).Shrink(0)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, "保存位置").FontSize(sizeMeta).TextColor(pal.muted).Shrink(0)
			ui.Text(c, shortenPath(expandHome(a.cfg.Dir))).FontSize(sizeBody).Grow(1).SingleLine().
				TextColor(t.Text).Tooltip(expandHome(a.cfg.Dir))
			secondaryButton(c, "更改…").OnClick(func() { a.browseDir() })
			primaryButton(c, "开始下载").Disabled(running).OnClick(func() { a.startDownload() })
		})
	})
}

// fieldHeight gives the links field the room the window has left: the page
// above it takes what it needs, the field takes the rest within reason.
func fieldHeight(c *ui.Context, busy bool) float32 {
	_, h := c.Size()
	if busy {
		return 190
	}
	f := h - 330
	if f < 170 {
		return 170
	}
	if f > 420 {
		return 420
	}
	return f
}

func (a *app) optionsModal(c *ui.Context, pal palette) {
	ui.Modal(c, &a.showAdvanced, func() {
		ui.Column(c).Width(580).Padding(28).Radius(22).Gap(22).
			Material(glass.Glass{}).Shadow(0, 24, 60, 4, pal.shadow).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "下载选项").FontSize(17).FontWeight(600).Grow(1)
				secondaryButton(c, "关闭").OnClick(func() { a.showAdvanced = false })
			})
			ui.Row(c).Gap(34).AlignItems(ui.Start).Children(func() {
				a.numberField(c, "线程数", &a.threads, 1, 16, "threads")
				a.numberField(c, "并发任务", &a.limit, 1, 8, "limit")
				ui.Column(c).Gap(6).Children(func() {
					quietLabel(c, pal, "未完成的下载")
					if ui.Segmented(c.Key("resume"), &a.resume, "继续", "重新开始").Changed() {
						a.cfg.Restart = a.resume == 1
						a.save()
					}
				})
			})
			a.machineHint(c)
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				ui.Row(c).Gap(34).Children(func() {
					a.switchField(c, "跳过同名文件", "已存在时不再下载", &a.cfg.SkipSame)
					a.switchField(c, "改写扩展名", "按文件类型补齐后缀", &a.cfg.RewriteExt)
				})
				ui.Row(c).Gap(34).Children(func() {
					a.switchField(c, "按对话分组", "每个对话放进子目录", &a.cfg.Group)
					a.switchField(c, "使用 Takeout", "导出会话，绕开限流", &a.cfg.Takeout)
				})
			})
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				a.textField(c, "只下载这些扩展名", "用逗号分隔，例如 mp4,mkv，留空表示不筛选", &a.cfg.Include)
				a.textField(c, "跳过这些扩展名", "用逗号分隔，例如 jpg,png", &a.cfg.Exclude)
				a.textField(c, "文件名模板", "默认 "+defaultTemplate(), &a.cfg.Template)
			})
		})
	})
}

func defaultTemplate() string {
	return "{{ .DialogID }}_{{ .MessageID }}_{{ filenamify .FileName }}"
}

func (a *app) numberField(c *ui.Context, label string, v *float64, lo, hi float64, key string) {
	ui.Column(c).Gap(8).Shrink(0).Children(func() {
		fieldLabel(c, c.Theme(), label)
		if ui.NumberInput(c.Key(key), v, lo, hi, 1).Changed() {
			a.syncedSettings()
			a.save()
		}
	})
}

func (a *app) switchField(c *ui.Context, label, detail string, v *bool) {
	t := c.Theme()
	ui.Row(c).Grow(1).Gap(10).AlignItems(ui.Center).Children(func() {
		if ui.Switch(c, v).Label(label).Changed() {
			a.save()
		}
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, label).FontSize(sizeBody).SingleLine()
			ui.Text(c, detail).FontSize(sizeMeta).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// machineHint says how many cores this machine has and what that suggests for
// the two numbers, so the fields start from a reason instead of a guess.
func (a *app) machineHint(c *ui.Context) {
	t := c.Theme()
	cores := runtime.NumCPU()
	ui.Row(c).Grow(1).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "这台机器有 %d 个核心，建议线程数 %d、并发任务 %d。",
			cores, suggestedThreads(cores), suggestedLimit(cores)).
			FontSize(sizeMeta).TextColor(t.TextMuted).Grow(1)
		secondaryButton(c, "用建议值").OnClick(func() { a.applySuggestion() })
	})
}

func (a *app) textField(c *ui.Context, label, placeholder string, v *string) {
	t := c.Theme()
	ui.Column(c).Gap(7).FillWidth().Children(func() {
		quietLabel(c, paletteFor(t), label)
		if ui.TextInput(c, v).Placeholder(placeholder).FontSize(sizeBody).Changed() {
			a.save()
		}
	})
}
