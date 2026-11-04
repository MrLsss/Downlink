package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestE2ELiveDownload downloads a real message with the tdl session the user
// already has, into the directory TDL_UI_E2E_DIR names:
//
//	TDL_UI_E2E=1 TDL_UI_E2E_LIVE=1 TDL_UI_E2E_NS=mrlsss \
//	TDL_UI_E2E_DIR=$PWD/.scratch/live TDL_UI_E2E_PROXY=http://127.0.0.1:7890 \
//	CGO_ENABLED=0 go test -run E2ELive -v .
func TestE2ELiveDownload(t *testing.T) {
	bin := e2eBinary(t)
	if os.Getenv("TDL_UI_E2E_LIVE") == "" {
		t.Skip("设置 TDL_UI_E2E_LIVE=1 后运行，会真的下载文件")
	}
	ns := os.Getenv("TDL_UI_E2E_NS")
	dir := os.Getenv("TDL_UI_E2E_DIR")
	if ns == "" || dir == "" {
		t.Skip("需要 TDL_UI_E2E_NS 与 TDL_UI_E2E_DIR")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The download has to read the session the user wrote with the command
	// line, so tdl looks at the real home directory here.
	t.Setenv("HOME", realHome)
	t.Setenv("TDL_UI_DATA_DIR", filepath.Join(dir, "app"))

	a := newApp()
	a.binPath = bin
	a.cfg.Namespace = ns
	a.cfg.Proxy = os.Getenv("TDL_UI_E2E_PROXY")
	a.cfg.Threads = 2
	a.cfg.Limit = 1

	urls := []string{"https://t.me/JBCATrepository/3571"}
	j := newJob(dir, urls)
	sess, err := startPTY(bin, a.downloadArgs(urls, dir), j.consume)
	if err != nil {
		t.Fatalf("启动 tdl 失败：%v", err)
	}
	runErr := sess.wait()
	sess.close()

	if runErr != nil {
		t.Fatalf("下载失败：%v\n输出：\n%s", runErr, joinLines(j.logTail))
	}
	done, count, bytes, totalBytes := j.totals()
	if count == 0 {
		t.Fatalf("没有读到文件名与进度，输出：\n%s", joinLines(j.logTail))
	}
	if done != count {
		t.Fatalf("只完成了 %d/%d 个文件，输出：\n%s", done, count, joinLines(j.logTail))
	}
	if bytes == 0 {
		t.Fatalf("没有读到下载的字节数，输出：\n%s", joinLines(j.logTail))
	}
	for _, key := range j.order {
		fp := j.files[key]
		path := fp.path
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("文件 %s 不在磁盘上：%v", path, err)
		}
		if st.Size() != fp.total {
			t.Fatalf("%s 的大小是 %d，进度行报告 %d", path, st.Size(), fp.total)
		}
	}
	t.Logf("下载了 %d 个文件，共 %s（进度行总计 %s），用时 %s",
		count, humanBytes(bytes), humanBytes(totalBytes), humanClock(time.Since(j.startedAt)))
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// TestE2ECancelStopsTheDownload cancels a transfer that has started and waits
// for the tdl process to end, which is what the app's 取消 button does:
//
//	TDL_UI_E2E=1 TDL_UI_E2E_LIVE=1 TDL_UI_E2E_NS=mrlsss \
//	TDL_UI_E2E_PROXY=http://127.0.0.1:7890 CGO_ENABLED=0 go test -run E2ECancel -v .
func TestE2ECancelStopsTheDownload(t *testing.T) {
	bin := e2eBinary(t)
	if os.Getenv("TDL_UI_E2E_LIVE") == "" {
		t.Skip("设置 TDL_UI_E2E_LIVE=1 后运行")
	}
	ns := os.Getenv("TDL_UI_E2E_NS")
	if ns == "" {
		t.Skip("需要 TDL_UI_E2E_NS")
	}
	dir := t.TempDir()
	t.Setenv("HOME", realHome)
	t.Setenv("TDL_UI_DATA_DIR", filepath.Join(dir, "app"))

	a := newApp()
	a.binPath = bin
	a.cfg.Namespace = ns
	a.cfg.Proxy = os.Getenv("TDL_UI_E2E_PROXY")
	a.cfg.Threads = 1
	a.cfg.Limit = 1

	// This message stays at zero bytes on this machine, which is what makes it
	// a good subject: the transfer is still running when it is cancelled.
	urls := []string{"https://t.me/SWKGGX/42651"}
	j := newJob(dir, urls)
	sess, err := startPTY(bin, a.downloadArgs(urls, dir), j.consume)
	if err != nil {
		t.Fatalf("启动 tdl 失败：%v", err)
	}
	j.sess = sess
	a.job = j

	ended := make(chan error, 1)
	go func() { ended <- sess.wait() }()

	// Let the command reach the transfer before cancelling it.
	time.Sleep(8 * time.Second)
	a.cancelDownload()

	select {
	case <-ended:
	case <-time.After(20 * time.Second):
		sess.kill()
		t.Fatalf("取消 20 秒后 tdl 仍在运行，输出：\n%s", joinLines(j.logTail))
	}
	sess.close()
	t.Logf("取消后 tdl 已退出，输出：\n%s", joinLines(j.logTail))
}
