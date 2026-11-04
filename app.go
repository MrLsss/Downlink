package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// app is the state the window shows. The view builds the interface from it on
// the main thread, whenever the window needs a frame.
type app struct {
	win *mygo.Window
	cfg config

	page  string
	links string

	threads float64
	limit   float64
	resume  int

	job *job

	login loginFlow

	binPath    string
	binVersion string
	binErr     string

	showAdvanced bool
	picking      bool

	notice    string
	noticeBad bool
	noticeSeq int

	closed atomic.Bool
}

func newApp() *app {
	a := &app{page: "download"}
	a.cfg = defaultConfig()
	a.threads = float64(a.cfg.Threads)
	a.limit = float64(a.cfg.Limit)
	return a
}

// load reads the settings and finds the tdl binary.
func (a *app) load() {
	cfg, err := loadConfig()
	if err != nil {
		// A settings file that cannot be read leaves the program on its
		// defaults, with the reason on screen instead of a window that never
		// appears.
		log.Printf("读取设置失败：%v", err)
		a.notifyProblem("设置文件读不出来，已改用默认设置：" + err.Error())
	}
	// The session tdl already keeps is the one to use, so a login made with
	// the command line keeps working here: take it when the namespace the
	// settings name has no session of its own and there is only one that does.
	if found := namespaces(); !hasSession(cfg.Namespace) && len(found) == 1 {
		cfg.Namespace = found[0]
	}
	a.cfg = cfg
	a.threads = float64(cfg.Threads)
	a.limit = float64(cfg.Limit)
	if cfg.Restart {
		a.resume = 1
	}
	a.probeBinary()
}

// probeBinary finds the tdl binary and asks it for its version.
func (a *app) probeBinary() {
	self, _ := os.Executable()
	path, err := findBinary(a.cfg, self)
	if err != nil {
		a.binPath, a.binVersion, a.binErr = "", "", err.Error()
		return
	}
	v, err := probeVersion(path)
	if err != nil {
		// A binary that answers nothing is no better than a missing one, so
		// downloads wait until the path the user gave works.
		a.binPath, a.binVersion, a.binErr = "", "", err.Error()
		return
	}
	a.binPath, a.binVersion, a.binErr = path, v, ""
}

func (a *app) save() {
	if err := saveConfig(a.cfg); err != nil {
		log.Printf("保存设置失败：%v", err)
		a.notifyProblem("设置没有存下来：" + err.Error())
	}
}

// update runs fn on the main thread, unless the window is gone.
func (a *app) update(fn func()) {
	if a.closed.Load() || a.win == nil {
		return
	}
	a.win.Update(fn)
}

// shutdown stops the commands that still run when the window closes.
func (a *app) shutdown() {
	a.closed.Store(true)
	if a.job != nil && a.job.running && a.job.sess != nil {
		// Ctrl-C lets tdl write down where it stopped; SIGKILL after a moment
		// keeps it from holding the database once this process is gone.
		a.job.sess.interrupt()
		sess := a.job.sess
		go func() {
			time.Sleep(250 * time.Millisecond)
			sess.kill()
		}()
	}
	if a.login.sess != nil {
		a.login.sess.interrupt()
		a.login.sess.close()
		a.login.sess = nil
	}
}

// notify shows a short message under the page's title.
func (a *app) notify(text string) {
	a.showNotice(text, false)
}

// notifyProblem shows a message about something that did not work.
func (a *app) notifyProblem(text string) {
	a.showNotice(text, true)
}

func (a *app) showNotice(text string, bad bool) {
	a.noticeSeq++
	seq := a.noticeSeq
	a.notice = text
	a.noticeBad = bad
	time.AfterFunc(4*time.Second, func() {
		a.update(func() {
			if a.noticeSeq == seq {
				a.notice = ""
				a.noticeBad = false
			}
		})
	})
}

// syncedSettings writes the numbers the widgets edit back into the settings.
func (a *app) syncedSettings() {
	a.cfg.Threads = int(a.threads)
	a.cfg.Limit = int(a.limit)
}

// suggestedThreads is how many connections one file gets: half the cores, at
// least two, at most eight, which is what tdl's own default sits in.
func suggestedThreads(cores int) int {
	n := cores / 2
	if n < 2 {
		n = 2
	}
	if n > 8 {
		n = 8
	}
	return n
}

// suggestedLimit is how many files download at once: a quarter of the cores,
// between one and four.
func suggestedLimit(cores int) int {
	n := cores / 4
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// applySuggestion sets both numbers to what this machine suggests.
func (a *app) applySuggestion() {
	cores := runtime.NumCPU()
	a.threads = float64(suggestedThreads(cores))
	a.limit = float64(suggestedLimit(cores))
	a.syncedSettings()
	a.save()
}

func (a *app) startDownload() {
	if a.job != nil && a.job.running {
		return
	}
	if a.binPath == "" {
		a.notify("先指定 tdl 可执行文件")
		a.page = "settings"
		return
	}
	urls := links(a.links)
	if len(urls) == 0 {
		a.notify("粘贴至少一个消息链接")
		return
	}
	dir := expandHome(strings.TrimSpace(a.cfg.Dir))
	if dir == "" {
		a.notify("选择保存位置")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.notifyProblem("无法创建保存目录：" + err.Error())
		return
	}
	a.cfg.Dir = dir
	a.syncedSettings()
	a.save()

	j := newJob(dir, urls)
	a.job = j
	sess, err := startPTY(a.binPath, a.downloadArgs(urls, dir), func(chunk string) {
		a.update(func() {
			if a.job == j {
				j.consume(chunk)
			}
		})
	})
	if err != nil {
		j.running, j.status, j.errText = false, statusFailed, err.Error()
		j.endedAt = time.Now()
		return
	}
	j.sess = sess
	go func() {
		err := sess.wait()
		sess.close()
		a.update(func() { a.finishDownload(j, err) })
	}()
}

// finishDownload records what the run ended with.
func (a *app) finishDownload(j *job, err error) {
	if a.job != j || !j.running {
		return
	}
	j.running = false
	j.endedAt = time.Now()
	switch {
	case j.cancelling:
		j.status = statusCancelled
	case err != nil || j.errText != "":
		j.status = statusFailed
		if j.errText == "" {
			j.errText = err.Error()
		}
	default:
		j.status = statusDone
	}
	done, count, bytes, _ := j.totals()
	if done < count {
		count = done
	}
	a.cfg.pushHistory(record{
		ID:     itoa(j.startedAt.UnixNano()),
		Start:  j.startedAt,
		End:    j.endedAt,
		Dir:    j.dir,
		URLs:   j.urls,
		Files:  count,
		Bytes:  bytes,
		Status: j.status,
		Error:  j.errText,
	})
	a.syncedSettings()
	a.save()
}

func (a *app) cancelDownload() {
	j := a.job
	if j == nil || !j.running || j.sess == nil || j.cancelling {
		return
	}
	j.cancelling = true
	j.sess.interrupt()
	time.AfterFunc(6*time.Second, func() {
		if a.win == nil {
			return
		}
		a.update(func() {
			if j.running && j.cancelling {
				j.sess.kill()
			}
		})
	})
}

// openLogin starts a login and opens the dialog that asks what it needs.
func (a *app) openLogin() {
	a.login = loginFlow{open: true, step: "start", status: "正在连接 Telegram…", gen: a.login.gen + 1}
	if a.binPath == "" {
		a.login.step = "failed"
		a.login.errText = "先指定 tdl 可执行文件"
		return
	}
	a.startLogin()
}

func (a *app) startLogin() {
	gen := a.login.gen
	sess, err := startPTY(a.binPath, a.loginArgs(), func(chunk string) {
		a.update(func() {
			if a.login.gen != gen {
				return
			}
			f := &a.login
			f.consume(chunk)
			if f.Account.Username != "" && a.cfg.Account != f.Account {
				a.cfg.Account = f.Account
				a.save()
			}
		})
	})
	if err != nil {
		a.login.step = "failed"
		a.login.errText = err.Error()
		return
	}
	a.login.sess = sess
	go func() {
		err := sess.wait()
		sess.close()
		a.update(func() {
			if a.login.gen == gen {
				a.loginExited(err)
			}
		})
	}()
}

func (a *app) loginExited(err error) {
	f := &a.login
	if f.step == "done" {
		return
	}
	if f.errText == "" && err != nil {
		f.errText = err.Error()
	}
	if f.errText == "" {
		f.errText = "登录进程已退出"
	}
	f.step = "failed"
}

func (a *app) closeLogin() {
	f := &a.login
	if f.sess != nil {
		f.sess.interrupt()
		f.sess.close()
		f.sess = nil
	}
	f.gen++
	f.open = false
}

// submitLogin sends what the dialog holds for the step the login waits on.
func (a *app) submitLogin() {
	f := &a.login
	if f.sess == nil {
		return
	}
	var value string
	switch f.step {
	case "phone":
		value = strings.TrimSpace(f.phone)
		if value == "" {
			f.errText = "输入手机号"
			return
		}
		f.step = "sending"
		f.status = "正在发送验证码…"
	case "code":
		value = strings.TrimSpace(f.code)
		if value == "" {
			f.errText = "输入验证码"
			return
		}
		f.step = "verifying"
		f.status = "正在验证验证码…"
	case "twofactor":
		value = f.twoFactor
		if value == "" {
			f.errText = "输入两步验证密码"
			return
		}
		f.step = "verifying"
		f.status = "正在验证密码…"
	default:
		return
	}
	f.errText = ""
	f.sess.write(value + "\r")
}

// pick asks for a path in a native dialog and hands it back on the UI thread.
// The dialog runs on its own goroutine so the window keeps drawing and the
// click that opened it returns at once.
func (a *app) pick(opts mygo.OpenDialogOptions, apply func(string)) {
	if a.picking {
		return
	}
	a.picking = true
	opts.Parent = a.win
	if a.win != nil {
		a.win.Focus()
	}
	go func() {
		paths, err := mygo.Dialog.Open(opts)
		a.update(func() {
			a.picking = false
			if err != nil {
				a.notifyProblem("打不开选择窗口：" + err.Error())
				return
			}
			if len(paths) == 0 {
				return
			}
			apply(paths[0])
		})
	}()
}

func (a *app) browseDir() {
	a.pick(mygo.OpenDialogOptions{
		Title:       "选择保存位置",
		DefaultPath: expandHome(a.cfg.Dir),
		Directory:   true,
	}, func(path string) {
		a.cfg.Dir = path
		a.save()
	})
}

func (a *app) browseBinary() {
	home, err := homeDir()
	if err != nil {
		a.notifyProblem("找不到用户目录：" + err.Error())
		return
	}
	a.pick(mygo.OpenDialogOptions{
		Title:       "选择 tdl 可执行文件",
		DefaultPath: home,
		ButtonLabel: "选择",
		Message:     "选择从 tdl 发布页下载的可执行文件。",
	}, func(path string) {
		a.cfg.Binary = path
		a.save()
		a.probeBinaryIn(path)
	})
}

// probeBinaryIn asks the binary for its version without holding the window,
// and drops the path when it does not answer like tdl does.
func (a *app) probeBinaryIn(path string) {
	go func() {
		v, err := probeVersion(path)
		a.update(func() {
			if err != nil {
				a.cfg.Binary = ""
				a.save()
				a.binPath, a.binVersion = "", ""
				a.binErr = "这个文件不是 tdl：" + err.Error()
				return
			}
			a.binPath, a.binVersion, a.binErr = path, v, ""
		})
	}()
}

// expandHome turns a path starting with ~ into one in the home directory.
func expandHome(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
