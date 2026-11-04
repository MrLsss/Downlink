package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// jobCard shows the download that runs, or the one that just ended: a pane of
// glass floating over the page.
func (a *app) jobCard(c *ui.Context, pal palette) {
	t := c.Theme()
	j := a.job
	title, ink := "正在下载", t.Text
	switch {
	case j.running && j.cancelling:
		title, ink = "正在停止…", pal.warn
	case !j.running && j.status == statusDone:
		title, ink = "下载完成", pal.ok
	case !j.running && j.status == statusFailed:
		title, ink = "下载失败", pal.danger
	case !j.running && j.status == statusCancelled:
		title, ink = "已取消", t.TextMuted
	}
	done, count, _, _ := j.totals()

	ui.Column(c).FillWidth().Padding(20).Radius(22).Gap(12).
		Material(glass.Glass{}).Shadow(0, 18, 44, 2, pal.shadow).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, title).FontSize(sizeBody).FontWeight(600).TextColor(ink)
			ui.Spacer(c)
			if j.running {
				if rate := humanRate(j.speed); rate != "" {
					ui.Text(c, rate).FontSize(sizeMeta).TextColor(t.TextMuted)
				}
			} else {
				ui.Text(c, humanClock(j.endedAt.Sub(j.startedAt))).FontSize(sizeMeta).TextColor(t.TextMuted)
			}
		})
		ui.Progress(c, j.fraction()).Height(6)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, j.summary()).FontSize(sizeMeta).TextColor(t.TextMuted).Grow(1).SingleLine()
			if j.running {
				secondaryButton(c, "取消").OnClick(func() { a.cancelDownload() })
			} else {
				secondaryButton(c, "打开目录").OnClick(func() { openPath(j.dir) })
				secondaryButton(c, "关闭").OnClick(func() { a.closeJob(j) })
			}
		})
		if j.running && time.Since(j.startedAt) > 15*time.Second {
			if _, _, got, _ := j.totals(); got == 0 {
				ui.Text(c, "已经开始下载，但这 15 秒里没有收到数据。").FontSize(sizeMeta).TextColor(pal.warn)
			}
		}
		if j.errText != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
				ui.Icon(c, iconAlert).FontSize(13).TextColor(pal.danger)
				ui.Text(c, j.errText).FontSize(sizeMeta).TextColor(pal.danger).Grow(1)
			})
		}
		switch {
		case len(j.order) > 0:
			// The card lists the files it has seen; the rest is a count, so the
			// page keeps one scroll instead of one per card.
			shown := len(j.order)
			if shown > 3 {
				shown = 3
			}
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, key := range j.order[:shown] {
					a.fileRow(c, key, pal)
				}
			})
			if rest := len(j.order) - shown; rest > 0 {
				ui.Text(c, "还有 "+itoa(int64(rest))+" 个文件").FontSize(sizeMeta).TextColor(t.TextMuted)
			}
		case j.running:
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(14, 14)
				ui.Text(c, "正在读取消息里的文件…").FontSize(sizeMeta).TextColor(t.TextMuted)
			})
		case done == 0 && count == 0:
			ui.Text(c, "这次没有可下载的文件。").FontSize(sizeMeta).TextColor(t.TextMuted)
		}
	})
}

func (a *app) fileRow(c *ui.Context, key string, pal palette) {
	fp := a.job.files[key]
	state := fmt.Sprintf("%.0f%%", fp.frac*100)
	ink := pal.muted
	switch {
	case fp.failed:
		state, ink = "失败", pal.danger
	case fp.done:
		state, ink = "完成", pal.ok
	}
	ui.Row(c).Height(24).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fp.name).FontSize(sizeMeta).SingleLine().Grow(1).Tooltip(fp.path)
		ui.Progress(c, fp.frac).Width(90).Height(4)
		ui.Text(c, state).FontSize(sizeMeta).Width(38).TextAlign(ui.End).TextColor(ink)
	})
}
