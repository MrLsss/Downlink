package main

import (
	"regexp"
	"strconv"
	"strings"
)

var loginOKRE = regexp.MustCompile(`Login successfully! ID: (\d+), Username: (.*)`)

// loginFlow drives `tdl login -T code` on a pseudo terminal and asks the user
// for what that command prompts.
type loginFlow struct {
	gen       int
	open      bool
	step      string
	phone     string
	code      string
	twoFactor string
	status    string
	errText   string
	sess      *ptySession
	buf       string
	Account   account

	sawPhone    bool
	sawCode     bool
	sawPassword bool
	sawDone     bool
}

// consume reads a chunk of the login command's output and moves the step
// forward when it prompts for something.
func (f *loginFlow) consume(chunk string) {
	f.buf += stripANSI(chunk)
	if len(f.buf) > 1<<15 {
		f.buf = f.buf[len(f.buf)-1<<13:]
	}
	b := f.buf

	if !f.sawPhone && strings.Contains(b, "Enter your phone number:") {
		f.sawPhone = true
		f.step = "phone"
		f.status = "输入手机号，带国家区号"
	}
	if strings.Contains(b, "Sending Code...") && (f.step == "phone" || f.step == "sending") {
		f.status = "正在发送验证码…"
	}
	if !f.sawCode && strings.Contains(b, "Enter Code:") {
		f.sawCode = true
		f.step = "code"
		f.status = "输入 Telegram 收到的验证码"
	}
	if !f.sawPassword && strings.Contains(b, "Enter 2FA Password:") {
		f.sawPassword = true
		f.step = "twofactor"
		f.status = "输入两步验证密码"
	}
	if !f.sawDone {
		if m := loginOKRE.FindStringSubmatch(b); m != nil {
			f.sawDone = true
			f.step = "done"
			f.status = "已登录"
			f.errText = ""
			id, _ := strconv.ParseInt(m[1], 10, 64)
			f.Account = account{ID: id, Username: strings.TrimSpace(m[2])}
		}
	}
	if e := lastError(b); e != "" && !f.sawDone {
		f.errText = e
		f.step = "failed"
		f.status = "登录没有完成"
	}
}

// lastError is the reason the command reported, preferring the cause tdl lists
// last under "Error: "; empty while it reported nothing of the sort.
func lastError(s string) string {
	i := strings.LastIndex(s, "Error: ")
	if i < 0 {
		return ""
	}
	tail := s[i+len("Error: "):]
	cause := ""
	for _, line := range strings.Split(tail, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") {
			cause = strings.TrimSpace(t[2:])
		}
	}
	if cause != "" {
		return cause
	}
	return strings.TrimSpace(strings.SplitN(tail, "\n", 2)[0])
}
