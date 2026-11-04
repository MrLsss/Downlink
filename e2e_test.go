package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// The tests below run the real tdl binary over a pseudo terminal. They need a
// network path to Telegram, so they run only when TDL_UI_E2E is set:
//
//	TDL_UI_E2E=1 TDL_UI_E2E_PROXY=http://127.0.0.1:7890 go test -run E2E -v .
func e2eBinary(t *testing.T) string {
	t.Helper()
	if os.Getenv("TDL_UI_E2E") == "" {
		t.Skip("设置 TDL_UI_E2E=1 后运行")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("这些用例需要 macOS 的 tdl 可执行文件")
	}
	bin, err := filepath.Abs(filepath.Join("tdl_MacOS_arm64", exeName()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("缺少 %s", bin)
	}
	return bin
}

// e2eHome points tdl at a temporary data directory, so its session, lock and
// log files stay out of the user's own ~/.tdl.
func e2eHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TDL_UI_DATA_DIR", filepath.Join(home, "app"))
	return home
}

// e2eApp is the state a download or a login needs, without a window.
func e2eApp(bin string) *app {
	a := newApp()
	a.binPath = bin
	a.cfg.Proxy = os.Getenv("TDL_UI_E2E_PROXY")
	a.cfg.Threads = 1
	a.cfg.Limit = 1
	return a
}

func TestE2EDownloadReportsMissingLogin(t *testing.T) {
	bin := e2eBinary(t)
	home := e2eHome(t)
	a := e2eApp(bin)

	dir := filepath.Join(home, "out")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	j := newJob(dir, []string{"https://t.me/tdl_e2e_does_not_exist/1"})
	sess, err := startPTY(bin, a.downloadArgs(j.urls, dir), j.consume)
	if err != nil {
		t.Fatalf("启动 tdl 失败：%v", err)
	}
	runErr := sess.wait()
	sess.close()

	if runErr == nil {
		t.Fatalf("下载命令没有失败：%q", joinLines(j.logTail))
	}
	if j.errText == "" {
		t.Fatalf("界面上没有错误文本，命令输出：%q", joinLines(j.logTail))
	}
}

// loginRun drives a login flow from a test, where the reader goroutine and the
// test itself look at the same state.
type loginRun struct {
	mu sync.Mutex
	f  loginFlow
}

func (r *loginRun) consume(chunk string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.f.consume(chunk)
}

func (r *loginRun) snapshot() loginFlow {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.f
}

// waitFor polls the flow until cond holds, the login reports an error, or the
// deadline passes.
func (r *loginRun) waitFor(t *testing.T, d time.Duration, cond func(loginFlow) bool) loginFlow {
	t.Helper()
	deadline := time.Now().Add(d)
	for {
		f := r.snapshot()
		if cond(f) || f.errText != "" || time.Now().After(deadline) {
			return f
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestE2ELoginAsksForThePhoneNumber(t *testing.T) {
	bin := e2eBinary(t)
	e2eHome(t)
	a := e2eApp(bin)

	var run loginRun
	sess, err := startPTY(bin, a.loginArgs(), run.consume)
	if err != nil {
		t.Fatalf("启动 tdl 登录失败：%v", err)
	}
	f := run.waitFor(t, 90*time.Second, func(f loginFlow) bool { return f.step == "phone" })
	sess.interrupt()
	sess.wait()
	sess.close()

	if f.errText != "" {
		t.Skipf("登录命令报告了错误，跳过：%s", f.errText)
	}
	if f.step != "phone" {
		t.Fatalf("登录没有走到手机号提示，step=%q 输出=%q", f.step, f.buf)
	}
}

func TestE2ELoginSendsThePhoneNumberItGets(t *testing.T) {
	bin := e2eBinary(t)
	e2eHome(t)
	a := e2eApp(bin)

	var run loginRun
	sess, err := startPTY(bin, a.loginArgs(), run.consume)
	if err != nil {
		t.Fatalf("启动 tdl 登录失败：%v", err)
	}
	defer func() {
		sess.interrupt()
		sess.wait()
		sess.close()
	}()

	if f := run.waitFor(t, 90*time.Second, func(f loginFlow) bool { return f.step == "phone" }); f.step != "phone" {
		t.Skipf("登录没有走到手机号提示，step=%q 错误=%q", f.step, f.errText)
	}

	// This number cannot exist, so Telegram answers without sending a code.
	sess.write("+9996612345678\r")
	f := run.waitFor(t, 90*time.Second, func(f loginFlow) bool {
		return f.step == "failed" || f.errText != ""
	})
	if f.errText == "" {
		t.Fatalf("没有读到 Telegram 的拒绝，step=%q 输出=%q", f.step, f.buf)
	}
	t.Logf("tdl 的回复：%s", f.errText)
}
