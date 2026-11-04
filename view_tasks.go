package main

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) tasksPage(c *ui.Context, pal palette) {
	t := c.Theme()
	a.scrolled(c, "tasks", func() {
		a.header(c, pal, "记录", "最近 40 次下载，可以打开目录，或者按原链接重新下载。")
		if len(a.cfg.History) == 0 {
			ui.Column(c).FillWidth().Height(180).Gap(8).Center().Children(func() {
				ui.Icon(c, iconHistory).FontSize(22).TextColor(t.TextMuted)
				ui.Text(c, "还没有下载记录").FontSize(sizeLead).FontWeight(500)
				ui.Text(c, "第一次下载完成后会出现在这里。").FontSize(sizeBody).TextColor(t.TextMuted)
			})
			return
		}
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			for i := range a.cfg.History {
				a.historyRow(c, i, pal)
			}
		})
	})
}

func (a *app) historyRow(c *ui.Context, i int, pal palette) {
	t := c.Theme()
	rec := a.cfg.History[i]
	icon, ink, word := iconCheck, pal.ok, "完成"
	switch rec.Status {
	case statusFailed:
		icon, ink, word = iconAlert, pal.danger, "失败"
	case statusCancelled:
		icon, ink, word = iconStop, t.TextMuted, "已取消"
	}
	meta := rec.End.Format("2006-01-02 15:04") + " · " + word
	if rec.Files > 0 {
		meta += " · " + itoa(int64(rec.Files)) + " 个文件"
	}
	if rec.Bytes > 0 {
		meta += " · " + humanBytes(rec.Bytes)
	}
	if len(rec.URLs) > 1 {
		meta += " · " + itoa(int64(len(rec.URLs))) + " 个链接"
	}

	row := ui.Row(c).FillWidth().Height(72).Padding(0, 12).Radius(12).Gap(14).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(pal.hover)
	}
	row.Children(func() {
		ui.Box(c).Size(32, 32).Shrink(0).Radius(10).Background(pal.chip).Center().Children(func() {
			ui.Icon(c, icon).FontSize(14).TextColor(ink)
		})
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Text(c, shortenPath(rec.Dir)).FontSize(sizeBody).FontWeight(500).SingleLine()
			ui.Text(c, meta).FontSize(sizeMeta).TextColor(t.TextMuted).SingleLine()
		})
		secondaryButton(c, "打开").OnClick(func() { openPath(rec.Dir) })
		secondaryButton(c, "重新下载").OnClick(func() {
			a.links = strings.Join(rec.URLs, "\n")
			a.cfg.Dir = rec.Dir
			a.page = "download"
		})
		del := ui.ButtonBase(c).Size(28, 28).Radius(14).Label("删除记录").Tooltip("删除记录")
		if del.Hovered() {
			del.Background(pal.hover)
		}
		del.Children(func() {
			ui.Icon(c, iconTrash).FontSize(13).TextColor(t.TextMuted)
		})
		del.OnClick(func() {
			a.cfg.removeHistory(rec.ID)
			a.save()
		})
	})
}

// section is one titled group of settings, separated from its neighbours by
// space alone.
func section(c *ui.Context, title string, fn func()) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Text(c, title).FontSize(sizeLead).FontWeight(600).TextColor(t.Text)
		fn()
	})
}

func (a *app) settingsPage(c *ui.Context, pal palette) {
	t := c.Theme()
	a.scrolled(c, "settings", func() {
		a.header(c, pal, "设置", "程序自己不做网络请求，所有操作都交给你指定的 tdl 完成。")

		section(c, "tdl 可执行文件", func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				if ui.TextInput(c, &a.cfg.Binary).Grow(1).FontSize(sizeBody).
					Placeholder("留空时自动查找，或填写完整路径").Submitted() {
					a.save()
					a.probeBinary()
				}
				secondaryButton(c, "浏览…").OnClick(func() { a.browseBinary() })
			})
			switch {
			case a.binErr != "":
				ui.Row(c).FillWidth().Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlert).FontSize(13).TextColor(pal.danger)
					ui.Text(c, a.binErr).FontSize(sizeBody).TextColor(pal.danger).Grow(1)
					secondaryButton(c, "获取 tdl").OnClick(func() {
						c.OpenURL("https://github.com/iyear/tdl/releases")
					})
				})
			case a.binPath != "":
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(pal.ok)
					ui.Text(c, a.binVersion).FontSize(sizeBody).TextColor(t.TextMuted).Shrink(0)
					ui.Text(c, shortenPath(a.binPath)).FontSize(sizeBody).TextColor(t.TextMuted).
						SingleLine().Grow(1).Tooltip(a.binPath)
					secondaryButton(c, "在 Finder 中显示").OnClick(func() { revealPath(a.binPath) })
				})
			}
			hint(c, pal, "填写完整路径后按回车重新检测；留空时依次在 PATH、应用资源目录、可执行文件所在目录与工作目录下查找。")
		})

		section(c, "连接", func() {
			a.textField(c, "命名空间", "默认 default", &a.cfg.Namespace)
			if found := namespaces(); len(found) > 0 {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "本机已有").FontSize(sizeMeta).TextColor(pal.muted)
					for _, ns := range found {
						name := ns
						b := secondaryButton(c, name)
						if name == a.cfg.Namespace {
							b.Disabled(true)
						}
						b.OnClick(func() {
							a.cfg.Namespace = name
							a.save()
						})
					}
				})
			}
			hint(c, pal, "每个命名空间是一套独立的登录会话，和应用外使用的 tdl 保持一致。")
			a.textField(c, "代理", "例如 socks5://127.0.0.1:1080 或 http://127.0.0.1:7890", &a.cfg.Proxy)
			hint(c, pal, "本机无法直连 Telegram 时填写；留空表示直连。")
		})

		section(c, "下载", func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				if ui.TextInput(c, &a.cfg.Dir).Grow(1).FontSize(sizeBody).Changed() {
					a.save()
				}
				secondaryButton(c, "选择…").OnClick(func() { a.browseDir() })
				secondaryButton(c, "打开").OnClick(func() { openPath(expandHome(a.cfg.Dir)) })
			})
			ui.Row(c).Gap(34).AlignItems(ui.Center).Children(func() {
				a.numberField(c, "线程数", &a.threads, 1, 16, "threads")
				a.numberField(c, "并发任务", &a.limit, 1, 8, "limit")
			})
			a.machineHint(c)
		})

		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, "Downlink 0.0.2 · 用 MyGo 构建").FontSize(sizeMeta).TextColor(pal.muted)
			ui.Spacer(c)
			secondaryButton(c, "使用文档").OnClick(func() { c.OpenURL("https://docs.iyear.me/tdl/zh/") })
			secondaryButton(c, "打开 tdl 数据目录").OnClick(func() { openPath(tdlDataDir()) })
		})
	})
}
