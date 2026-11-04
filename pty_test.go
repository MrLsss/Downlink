package main

import (
	"runtime"
	"testing"
	"time"
)

// endWithin waits for the session to end, so that a command that keeps running
// fails the test instead of hanging it.
func endWithin(t *testing.T, s *ptySession, d time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- s.wait() }()
	select {
	case <-done:
	case <-time.After(d):
		s.kill()
		t.Fatalf("命令没有在 %s 内结束", d)
	}
	s.close()
}

func TestInterruptStopsTheCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ctrl-C 通过终止进程实现，测试在 Unix 上跑")
	}
	s, err := startPTY("/bin/sh", []string{"-c", "sleep 30"}, func(string) {})
	if err != nil {
		t.Skipf("分配伪终端失败：%v", err)
	}
	s.interrupt()
	endWithin(t, s, 5*time.Second)
}

func TestKillStopsTheCommandAndWhatItStarted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("结束进程树的方式不同，测试在 Unix 上跑")
	}
	s, err := startPTY("/bin/sh", []string{"-c", "sleep 60 & wait"}, func(string) {})
	if err != nil {
		t.Skipf("分配伪终端失败：%v", err)
	}
	s.kill()
	endWithin(t, s, 5*time.Second)
}
