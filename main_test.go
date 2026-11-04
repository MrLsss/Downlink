package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// realHome is the home directory the user runs the program in, kept so a test
// that needs a written tdl session can point tdl back at it.
var realHome = os.Getenv("HOME")

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tdl-ui-test")
	if err != nil {
		panic(err)
	}
	// The tests keep their settings and their view of ~/.tdl away from the
	// user's own files.
	os.Setenv("TDL_UI_DATA_DIR", filepath.Join(dir, "app"))
	os.Setenv("HOME", filepath.Join(dir, "home"))
	if err := os.MkdirAll(os.Getenv("HOME"), 0o755); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func newTester(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	a := newApp()
	return a, ui.NewTester(a.view, 1120, 764)
}

func TestDownloadPage(t *testing.T) {
	_, tt := newTester(t)
	for _, want := range []string{"下载", "Telegram 下载器", "未登录", "链接", "保存位置", "选项", "开始下载"} {
		if !tt.HasText(want) {
			t.Errorf("缺少 %q：%q", want, tt.Texts())
		}
	}
}

func TestClearLinks(t *testing.T) {
	a, tt := newTester(t)
	a.links = "https://t.me/example/1\nhttps://t.me/example/2"
	tt.Frame()
	if err := tt.Click("清空"); err != nil {
		t.Fatal(err)
	}
	if a.links != "" {
		t.Fatalf("links %q after 清空", a.links)
	}
	tt.Frame()
	if tt.HasText("2 个链接") {
		t.Fatalf("the count stayed on screen：%q", tt.Texts())
	}
}

func TestNavigation(t *testing.T) {
	a, tt := newTester(t)
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if a.page != "settings" {
		t.Fatalf("page %q after clicking 设置", a.page)
	}
	if !tt.HasText("tdl 可执行文件") || !tt.HasText("连接") {
		t.Errorf("settings page shows %q", tt.Texts())
	}
	if err := tt.Click("记录"); err != nil {
		t.Fatal(err)
	}
	if a.page != "tasks" {
		t.Fatalf("page %q after clicking 记录", a.page)
	}
	if !tt.HasText("还没有下载记录") {
		t.Errorf("tasks page shows %q", tt.Texts())
	}
}

func TestStartWithoutLinks(t *testing.T) {
	a, tt := newTester(t)
	a.binPath = "/usr/bin/true"
	if err := tt.Click("开始下载"); err != nil {
		t.Fatal(err)
	}
	if a.notice != "粘贴至少一个消息链接" {
		t.Fatalf("notice %q", a.notice)
	}
	if a.job != nil {
		t.Fatal("a download started without links")
	}
	if !tt.HasText("粘贴至少一个消息链接") {
		t.Errorf("the page does not show %q", tt.Texts())
	}
}

func TestStartWithoutBinary(t *testing.T) {
	a, _ := newTester(t)
	a.links = "https://t.me/example/1"
	a.binPath = ""
	a.startDownload()
	if a.page != "settings" {
		t.Fatalf("page %q", a.page)
	}
	if a.job != nil {
		t.Fatal("a download started without the binary")
	}
}

func TestHistoryRow(t *testing.T) {
	a, tt := newTester(t)
	if err := tt.Click("记录"); err != nil {
		t.Fatal(err)
	}
	a.cfg.History = []record{{
		ID:     "1",
		Dir:    expandHome("~/Downloads/tdl"),
		URLs:   []string{"https://t.me/example/1"},
		Files:  3,
		Bytes:  5 << 20,
		Status: statusDone,
	}}
	tt.Frame()
	if !tt.HasText("3 个文件") || !tt.HasText("5.0 MB") {
		t.Errorf("history row shows %q", tt.Texts())
	}
}

func TestLoginModal(t *testing.T) {
	a, tt := newTester(t)
	a.login = loginFlow{open: true, step: "phone", status: "输入手机号，带国家区号"}
	tt.Frame()
	if !tt.HasText("登录 Telegram") || !tt.HasText("发送验证码") {
		t.Errorf("login dialog shows %q", tt.Texts())
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	if a.login.open {
		t.Fatal("the dialog stayed open after 关闭")
	}
}

// progressLines is what tdl writes for one 50% file and a finished download.
const (
	halfLine = `电影(123):456 -> tdl_123_456_movie.mp4 ...  50.0% [########....] [1.00 MB in 1s; 1.00 MB/s]`
	doneLine = `电影(123):456 -> tdl_123_456_movie.mp4 ... done! [2.00 MB in 2s; 1.00 MB/s]`
)

func TestJobCardShowsTheRunningDownload(t *testing.T) {
	a, tt := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	j.consume(halfLine + "\n")
	a.job = j
	tt.Frame()
	for _, want := range []string{"正在下载", "tdl_123_456_movie.mp4", "50%", "1.0 MB / 2.0 MB"} {
		if !tt.HasText(want) {
			t.Errorf("the job card does not show %q：%q", want, tt.Texts())
		}
	}
}

func TestFinishedDownloadLandsInTheHistory(t *testing.T) {
	a, _ := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	j.consume(halfLine + "\n" + doneLine + "\n")
	a.job = j
	a.finishDownload(j, nil)

	if len(a.cfg.History) != 1 {
		t.Fatalf("history has %d records", len(a.cfg.History))
	}
	rec := a.cfg.History[0]
	if rec.Status != statusDone || rec.Files != 1 || rec.Bytes != 2<<20 {
		t.Fatalf("record = %+v", rec)
	}
}

func TestCancelledDownloadLandsInTheHistory(t *testing.T) {
	a, _ := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	j.consume(halfLine + "\n")
	j.cancelling = true
	a.job = j
	a.finishDownload(j, nil)
	if a.cfg.History[0].Status != statusCancelled {
		t.Fatalf("record = %+v", a.cfg.History[0])
	}
}

func TestStaleJobDoesNotFinishTheNewOne(t *testing.T) {
	a, _ := newTester(t)
	old := newJob("/tmp/tdl", []string{"https://t.me/example/1"})
	cur := newJob("/tmp/tdl", []string{"https://t.me/example/2"})
	a.job = cur
	a.finishDownload(old, nil)
	if !cur.running {
		t.Fatal("the stale run finished the current one")
	}
}

func TestFinishedJobCanBeClosed(t *testing.T) {
	a, tt := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	a.job = j
	a.finishDownload(j, nil)
	tt.Frame()
	if !tt.HasText("关闭") {
		t.Fatalf("the finished card offers no way out：%q", tt.Texts())
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	if a.job != nil {
		t.Fatalf("job still on the page after 关闭：%+v", a.job)
	}
	tt.Frame()
	if tt.HasText("下载完成") {
		t.Fatalf("the closed card still shows：%q", tt.Texts())
	}
}

func TestCancelledJobCanBeClosedAndStartedAgain(t *testing.T) {
	a, tt := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	j.cancelling = true
	a.job = j
	a.finishDownload(j, nil)
	a.links = "https://t.me/example/456"
	tt.Frame()
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("开始下载"); err != nil {
		t.Fatal(err)
	}
	if a.notice != "先指定 tdl 可执行文件" {
		t.Fatalf("开始下载 did nothing: notice %q, page %q", a.notice, a.page)
	}
}

func TestRunningJobSaysWhenNoBytesArrive(t *testing.T) {
	a, tt := newTester(t)
	j := newJob("/tmp/tdl", []string{"https://t.me/example/456"})
	j.startedAt = time.Now().Add(-20 * time.Second)
	a.job = j
	tt.Frame()
	if !tt.HasText("已经开始下载，但这 15 秒里没有收到数据。") {
		t.Fatalf("the stalled download is not reported：%q", tt.Texts())
	}
}
