package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// inside reports whether the element lies within the window.
func inside(t *testing.T, tt *ui.Tester, label string, w, h int) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("%q is not on screen：%q", label, tt.Texts())
	}
	if r.X < 0 || r.X+r.W > float32(w) || r.Y < 0 || r.Y+r.H > float32(h) {
		t.Fatalf("%q sits at %+v, outside the %dx%d window", label, r, w, h)
	}
	return r
}

func TestDownloadPageLayout(t *testing.T) {
	_, tt := newTester(t)
	tt.Frame()

	// The page starts to the right of the sidebar and below the title bar.
	subtitle := inside(t, tt, "粘贴 Telegram 消息链接，选好保存位置，剩下的交给 tdl。", 1120, 764)
	if subtitle.X < 250 {
		t.Fatalf("the page starts at %v, inside the sidebar", subtitle.X)
	}
	if subtitle.Y < 40 || subtitle.Y > 130 {
		t.Fatalf("the page title sits at %v", subtitle.Y)
	}

	// The composer asks first for links, then for a place to put them, then
	// for the one action.
	label := inside(t, tt, "链接", 1120, 764)
	if label.X < subtitle.X || label.Y <= subtitle.Y {
		t.Fatalf("the links field starts at %+v, above or left of the title", label)
	}
	path := inside(t, tt, "保存位置", 1120, 764)
	if path.Y <= label.Y {
		t.Fatalf("the place row sits above the links：%v <= %v", path.Y, label.Y)
	}
	start := inside(t, tt, "开始下载", 1120, 764)
	if start.X+start.W > 1086 {
		t.Fatalf("the primary action ends at %v, past the page margin", start.X+start.W)
	}
	if start.Y <= label.Y {
		t.Fatalf("the primary action sits above the links field：%v <= %v", start.Y, label.Y)
	}

	options := inside(t, tt, "下载选项", 1120, 764)
	if options.Y >= label.Y {
		t.Fatalf("the options button sits at %v, below the links field %v", options.Y, label.Y)
	}
	if options.X <= start.X {
		t.Fatalf("the options button starts at %v, left of the primary action %v", options.X, start.X)
	}
}

func TestSidebarStaysInItsColumn(t *testing.T) {
	_, tt := newTester(t)
	tt.Frame()
	for _, label := range []string{"记录", "设置", "未登录"} {
		if r := inside(t, tt, label, 1120, 764); r.X+r.W > 238 {
			t.Fatalf("%q reaches %v, past the sidebar", label, r.X+r.W)
		}
	}
}

func TestDownloadPageFitsTheSmallestWindow(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 940, 620)
	tt.Frame()
	for _, label := range []string{"下载", "链接", "粘贴", "清空", "开始下载", "下载选项"} {
		inside(t, tt, label, 940, 620)
	}
}

// TestRunningJobKeepsTheComposerInView keeps the tallest state of the page —
// a running download with many files — from pushing the action out of the
// window the app opens with.
func TestRunningJobKeepsTheComposerInView(t *testing.T) {
	a := newApp()
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	for i := 0; i < 8; i++ {
		j.consume(fmt.Sprintf("电影(123):%d -> tdl_123_%d_movie.mp4 ...  50.0%% [########....] [1.00 MB in 1s; 1.00 MB/s]\n", 456+i, 456+i))
	}
	j.running = true
	a.job = j
	a.notice = "无法创建保存目录：mkdir /Users/someone/Downloads/tdl: operation not permitted"
	a.noticeBad = true
	tt := ui.NewTester(a.view, 1120, 764)
	tt.Frame()
	for _, label := range []string{"正在下载", "链接", "开始下载", "还有 5 个文件", "下载选项"} {
		inside(t, tt, label, 1120, 764)
	}
}

func TestSettingsPageLayout(t *testing.T) {
	a, tt := newTester(t)
	a.page = "settings"
	tt.Frame()
	for _, label := range []string{"设置", "tdl 可执行文件", "浏览…", "连接", "代理", "下载", "用建议值"} {
		inside(t, tt, label, 1120, 764)
	}
	noOverlap(t, tt, "线程数", "并发任务")
}

// TestSettingsPageOpensOnItsFirstSection keeps the settings page usable at the
// smallest window the app opens: the top of the page lands inside it, and the
// rest is reached by scrolling.
func TestSettingsPageOpensOnItsFirstSection(t *testing.T) {
	a := newApp()
	a.page = "settings"
	tt := ui.NewTester(a.view, 940, 620)
	tt.Frame()
	for _, label := range []string{"设置", "tdl 可执行文件", "浏览…", "连接"} {
		inside(t, tt, label, 940, 620)
	}
	noOverlap(t, tt, "线程数", "并发任务")
}

// TestOptionsDialogLayout keeps the download options on one column each and
// inside the window.
func TestOptionsDialogLayout(t *testing.T) {
	a := newApp()
	a.showAdvanced = true
	tt := ui.NewTester(a.view, 1120, 764)
	tt.Frame()
	for _, label := range []string{"下载选项", "线程数", "并发任务", "未完成的下载", "文件名模板", "关闭"} {
		inside(t, tt, label, 1120, 764)
	}
	noOverlap(t, tt, "线程数", "并发任务")
	noOverlap(t, tt, "并发任务", "未完成的下载")

	// Each switch stands beside the text that names it, and the switches of one
	// row share a line while the rows stack.
	first := inside(t, tt, "跳过同名文件", 1120, 764)
	second := inside(t, tt, "改写扩展名", 1120, 764)
	below := inside(t, tt, "按对话分组", 1120, 764)
	if second.Y != first.Y {
		t.Fatalf("the two switches stand at y=%v and y=%v", first.Y, second.Y)
	}
	if second.X-first.X < 160 {
		t.Fatalf("the second switch starts %v after the first", second.X-first.X)
	}
	if below.Y <= first.Y || below.X != first.X {
		t.Fatalf("the second row stands at %v, the first at %v", below, first)
	}
	detail := inside(t, tt, "已存在时不再下载", 1120, 764)
	if detail.X < first.X+first.W+8 || detail.Y <= first.Y {
		t.Fatalf("the description stands at %v, its switch at %v", detail, first)
	}
}

// noOverlap says the control that starts at the second label begins after the
// one at the first label ended, which a column too narrow for its control
// breaks.
func noOverlap(t *testing.T, tt *ui.Tester, first, second string) {
	t.Helper()
	a := inside(t, tt, first, 1120, 764)
	b := inside(t, tt, second, 1120, 764)
	if b.X-a.X < 160 {
		t.Fatalf("%q starts %v after %q, closer than one control", second, b.X-a.X, first)
	}
	if b.Y != a.Y {
		t.Fatalf("%q sits at y=%v while %q sits at y=%v", second, b.Y, first, a.Y)
	}
}

func TestTasksPageLayout(t *testing.T) {
	a, tt := newTester(t)
	a.page = "tasks"
	tt.Frame()
	empty := inside(t, tt, "还没有下载记录", 1120, 764)
	title := inside(t, tt, "最近 40 次下载，可以打开目录，或者按原链接重新下载。", 1120, 764)
	if empty.Y <= title.Y {
		t.Fatalf("the empty state sits above the page title：%v <= %v", empty.Y, title.Y)
	}
}

// TestSettingsSectionTitlesStandOut keeps the three section titles larger than
// the field labels under them.
func TestSettingsSectionTitlesStandOut(t *testing.T) {
	a, tt := newTester(t)
	a.page = "settings"
	tt.Frame()
	heading := inside(t, tt, "连接", 1120, 764)
	label := inside(t, tt, "代理", 1120, 764)
	if heading.H <= label.H {
		t.Fatalf("标题 %v 高，标签 %v 高，标题不再突出", heading.H, label.H)
	}
}
