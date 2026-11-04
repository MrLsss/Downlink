package main

import (
	"strings"
	"testing"
	"time"
)

func TestLinks(t *testing.T) {
	got := links("  https://t.me/a/1 \n\n# 注释\nhttps://t.me/b/2\n")
	want := []string{"https://t.me/a/1", "https://t.me/b/2"}
	if len(got) != len(want) {
		t.Fatalf("links %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("links[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:          "0 B",
		1023:       "1023 B",
		1024:       "1.0 KB",
		1536:       "1.5 KB",
		5 << 20:    "5.0 MB",
		3 << 30:    "3.0 GB",
		11 << 30:   "11.0 GB",
		1200 << 20: "1.2 GB",
	}
	for n, want := range cases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestHumanClock(t *testing.T) {
	cases := map[time.Duration]string{
		0:                             "0s",
		3 * time.Second:               "3s",
		65 * time.Second:              "1m05s",
		2*time.Hour + 300*time.Second: "2h05m",
	}
	for d, want := range cases {
		if got := humanClock(d); got != want {
			t.Errorf("humanClock(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	out := "Version: 0.20.4\nCommit: 9d7d49e\n\nBuilt with: go1.25.8 darwin/arm64\n"
	if got := parseVersion(out); got != "0.20.4" {
		t.Fatalf("parseVersion = %q", got)
	}
	if got := parseVersion("something else"); got != "" {
		t.Fatalf("parseVersion on other output = %q", got)
	}
	// tdl writes a logging warning before the version when its data
	// directory is not writable.
	warn := "write error: can't rename log file\nVersion: 0.20.4\n"
	if got := parseVersion(warn); got != "0.20.4" {
		t.Fatalf("parseVersion with a warning = %q", got)
	}
}

func TestDownloadArgs(t *testing.T) {
	a := newApp()
	a.cfg.Namespace = "work"
	a.cfg.Threads = 6
	a.cfg.Limit = 3
	a.cfg.Proxy = "socks5://127.0.0.1:1080"
	a.cfg.Include = "mp4,mkv"
	a.cfg.Exclude = "jpg"
	a.cfg.Template = "{{ .FileName }}"
	a.cfg.SkipSame = true
	a.cfg.RewriteExt = true

	got := strings.Join(a.downloadArgs([]string{"https://t.me/a/1", "https://t.me/a/2"}, "/tmp/out"), " ")
	want := "-n work -t 6 -l 3 --proxy socks5://127.0.0.1:1080 dl " +
		"-u https://t.me/a/1 -u https://t.me/a/2 -d /tmp/out " +
		"-i mp4,mkv -e jpg --template {{ .FileName }} --skip-same --rewrite-ext --continue"
	if got != want {
		t.Fatalf("downloadArgs\n got %q\nwant %q", got, want)
	}

	a.cfg.Restart = true
	if !strings.Contains(strings.Join(a.downloadArgs(nil, "/tmp/out"), " "), "--restart") {
		t.Fatal("restart is not passed")
	}
	if strings.Contains(strings.Join(a.downloadArgs(nil, "/tmp/out"), " "), "--continue") {
		t.Fatal("continue is passed with restart")
	}

	login := strings.Join(a.loginArgs(), " ")
	if login != "-n work -t 6 -l 3 --proxy socks5://127.0.0.1:1080 login -T code" {
		t.Fatalf("loginArgs %q", login)
	}
}

func TestJobProgress(t *testing.T) {
	j := newJob("/tmp/out", []string{"https://t.me/a/1"})

	j.consume("Movie One(123):456 -> /tmp/out/Movie One.mp4 ...  50.0% [######....] [57.22 MB in 301ms; 189.87 MB/s]\n")
	if len(j.files) != 1 {
		t.Fatalf("files %d", len(j.files))
	}
	fp := j.files["123:456"]
	if fp == nil {
		t.Fatalf("no file for 123:456: %+v", j.files)
	}
	if fp.name != "Movie One.mp4" || fp.path != "/tmp/out/Movie One.mp4" {
		t.Fatalf("file %+v", fp)
	}
	if fp.frac < 0.49 || fp.frac > 0.51 {
		t.Fatalf("frac %v", fp.frac)
	}
	if fp.value <= 0 || fp.total <= fp.value {
		t.Fatalf("value %d total %d", fp.value, fp.total)
	}
	if j.speed != 0 {
		t.Fatalf("speed %d from a file line", j.speed)
	}

	j.consume("[#######...] [12s; 194.28 MB/s]\n")
	if j.speed != parseBytes("194.28 MB") {
		t.Fatalf("overall speed %d", j.speed)
	}

	j.consume("Movie One(123):456 -> /tmp/out/Movie One.mp4 ... done! [1.34 MB in 401ms; 3.33 MB/s]\n")
	if !fp.done || fp.frac != 1 {
		t.Fatalf("file after done %+v", fp)
	}
	done, count, bytes, total := j.totals()
	if done != 1 || count != 1 {
		t.Fatalf("totals done %d count %d", done, count)
	}
	if bytes != fp.value || total < bytes {
		t.Fatalf("bytes %d total %d from %+v", bytes, total, fp)
	}
	if !strings.Contains(j.summary(), "1 / 1 个文件") {
		t.Fatalf("summary %q", j.summary())
	}
}

func TestJobFailure(t *testing.T) {
	j := newJob("/tmp/out", nil)
	j.consume("\x1b[31mError: failed to get message\x1b[0m\n")
	if j.errText != "failed to get message" {
		t.Fatalf("errText %q", j.errText)
	}
	if len(j.logTail) != 1 {
		t.Fatalf("logTail %q", j.logTail)
	}
}

func TestJobFailureTakesTheCauseUnderTheError(t *testing.T) {
	j := newJob("/tmp/out", nil)
	j.consume("\x1b[31mError: callback:\x1b[0m\n")
	j.consume("    github.com/gotd/td/telegram.(*Client).Run.func3\n")
	j.consume("        github.com/gotd/td@v0.140.0/telegram/connect.go:180\n")
	j.consume("  - not authorized. please login first\n")
	if j.errText != "not authorized. please login first" {
		t.Fatalf("errText %q", j.errText)
	}
}

func TestJobSummaryCountsRemainingDown(t *testing.T) {
	j := newJob("/tmp/out", nil)
	fp := j.file("Clip(9):10 -> /tmp/out/Clip.mov")
	fp.total = 100 * 1024 * 1024

	// The left half of the overall bracket is the time already spent, so the
	// remaining time has to fall as the file grows.
	for _, step := range []struct {
		percent int
		elapsed int
		want    string
	}{{10, 10, "1m30s"}, {30, 30, "1m10s"}, {60, 60, "40s"}, {90, 90, "10s"}} {
		fp.value = int64(step.percent) * 1024 * 1024
		j.consume("[#######...] [" + itoa(int64(step.elapsed)) + "s; 1.00 MB/s]\n")
		if got := j.remaining(); got != step.want {
			t.Fatalf("at %d%% the remaining time is %q, want %q", step.percent, got, step.want)
		}
		if s := j.summary(); !strings.Contains(s, "剩余 "+step.want) {
			t.Fatalf("at %d%% the summary is %q", step.percent, s)
		}
	}
}

func TestJobRemainingFollowsTheRestOfTheBytes(t *testing.T) {
	j := newJob("/tmp/out", nil)
	fp := j.file("Movie One(123):456 -> /tmp/out/Movie One.mp4")
	fp.total = 10 * 1024 * 1024
	j.speed = 1024 * 1024

	if got := j.remaining(); got != "10s" {
		t.Fatalf("remaining %q for 10 MB at 1 MB/s", got)
	}

	fp.value = 8 * 1024 * 1024
	if got := j.remaining(); got != "2s" {
		t.Fatalf("remaining %q with 2 MB left", got)
	}

	j.consume("[#######...] [40s; 1.00 MB/s]\n")
	if j.speed != 1024*1024 {
		t.Fatalf("speed %d", j.speed)
	}
	if got := j.remaining(); got != "2s" {
		t.Fatalf("remaining %q after the overall line", got)
	}

	fp.value = fp.total
	fp.done = true
	if got := j.remaining(); got != "" {
		t.Fatalf("remaining %q with nothing left", got)
	}
}

func TestJobPercentWithoutDetail(t *testing.T) {
	j := newJob("/tmp/out", nil)
	j.consume("Clip(9):10 -> /tmp/out/Clip.mov ...  25.0% [##......] [\n")
	fp := j.files["9:10"]
	if fp == nil || fp.frac != 0.25 {
		t.Fatalf("file %+v", fp)
	}
}

func TestLoginFlow(t *testing.T) {
	var f loginFlow
	f.consume("? Enter your phone number: ")
	if f.step != "phone" {
		t.Fatalf("step %q", f.step)
	}
	f.step = "sending"
	f.consume("Sending Code...")
	if f.status != "正在发送验证码…" {
		t.Fatalf("status %q after Sending Code", f.status)
	}
	f.consume("? Enter Code: ")
	if f.step != "code" {
		t.Fatalf("step %q after the code prompt", f.step)
	}
	f.consume("? Enter 2FA Password: ")
	if f.step != "twofactor" {
		t.Fatalf("step %q after the password prompt", f.step)
	}
	f.consume("Login successfully! ID: 12345, Username: alice\n")
	if f.step != "done" {
		t.Fatalf("step %q after success", f.step)
	}
	if f.Account.ID != 12345 || f.Account.Username != "alice" {
		t.Fatalf("account %+v", f.Account)
	}
}

func TestLoginFlowError(t *testing.T) {
	var f loginFlow
	f.consume("? Enter your phone number: ")
	f.consume("\x1b[31mError: phone number is invalid\x1b[0m\n")
	if f.errText != "phone number is invalid" {
		t.Fatalf("errText %q", f.errText)
	}
}

func TestTextInputPromptDetection(t *testing.T) {
	var f loginFlow
	f.consume("? Enter your phone number: (+86 12345678900)")
	if f.sawPhone != true || f.step != "phone" {
		t.Fatalf("prompt not detected: %+v", f)
	}
	f.consume("? Enter your phone number: (+86 12345678900)")
	if f.step != "phone" {
		t.Fatalf("step %q", f.step)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	cfg := defaultConfig()
	cfg.Namespace = "work"
	cfg.Proxy = "http://127.0.0.1:7890"
	cfg.Threads = 8
	cfg.Limit = 4
	cfg.Account = account{ID: 42, Username: "bob"}
	cfg.pushHistory(record{ID: "a", Dir: "/tmp/a", Status: statusDone})
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Namespace != "work" || got.Proxy != "http://127.0.0.1:7890" || got.Threads != 8 || got.Limit != 4 {
		t.Fatalf("loaded %+v", got)
	}
	if got.Account.Username != "bob" || len(got.History) != 1 {
		t.Fatalf("loaded %+v", got)
	}
}

func TestHistoryLimit(t *testing.T) {
	var cfg config
	for i := 0; i < historyLimit+5; i++ {
		cfg.pushHistory(record{ID: itoa(int64(i))})
	}
	if len(cfg.History) != historyLimit {
		t.Fatalf("history %d", len(cfg.History))
	}
	if cfg.History[0].ID != itoa(int64(historyLimit+4)) {
		t.Fatalf("newest entry is %q", cfg.History[0].ID)
	}
	cfg.removeHistory(itoa(int64(historyLimit + 4)))
	if len(cfg.History) != historyLimit-1 {
		t.Fatalf("history after remove %d", len(cfg.History))
	}
	for _, r := range cfg.History {
		if r.ID == itoa(int64(historyLimit+4)) {
			t.Fatal("the removed entry is still there")
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Namespace != "default" || cfg.Threads != 4 || cfg.Limit != 2 {
		t.Fatalf("defaults %+v", cfg)
	}
	if cfg.Restart || cfg.SkipSame || cfg.Takeout {
		t.Fatalf("switches start on: %+v", cfg)
	}
}

func TestConfigFix(t *testing.T) {
	cfg := config{}
	cfg.fix()
	if cfg.Namespace != "default" || cfg.Threads != 4 || cfg.Limit != 2 || cfg.Dir == "" {
		t.Fatalf("fixed %+v", cfg)
	}
}
