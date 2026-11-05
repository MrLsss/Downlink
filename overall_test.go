package main

import (
	"strings"
	"testing"
)

const (
	chatLine = "丝袜控(4385158554):"
	half     = 1986422374 // 1.85 GiB
)

func overallOf(t *testing.T, lines ...string) (*job, float64) {
	t.Helper()
	j := newJob("/tmp", []string{"https://t.me/a/1"})
	j.running = true
	for _, line := range lines {
		j.consume(line + "\n")
	}
	return j, j.fraction()
}

// TestOverallProgressAddsUpTheFiles is the two file case: the first file is
// done, the second one is at 66%, and the bar has to sit around 83% instead of
// jumping to the end.
func TestOverallProgressAddsUpTheFiles(t *testing.T) {
	j, got := overallOf(t,
		chatLine+"13964 -> a.mp4 ... done! [1.85 GB in 40s; 46.25 MB/s]",
		chatLine+"13965 -> b.mp4 ...  0.1% [..........] [1.22 MB in 1s; 1.22 MB/s]",
		chatLine+"13965 -> b.mp4 ... 66.0% [######....] [1.22 GB in 10s; 122.00 MB/s]",
	)
	if got < 0.82 || got > 0.84 {
		t.Fatalf("总进度是 %v，应该约 0.83", got)
	}
	if s := j.summary(); !strings.Contains(s, "1 / 2 个文件") || !strings.Contains(s, "3.1 GB / 3.7 GB") {
		t.Fatalf("汇总行是 %q", s)
	}
}

// TestOverallProgressLeavesRoomForAFileThatHasNotStarted keeps the bar away
// from the end while a file of the message has not reported anything yet.
func TestOverallProgressLeavesRoomForAFileThatHasNotStarted(t *testing.T) {
	j := newJob("/tmp", []string{"https://t.me/a/1"})
	j.running = true
	j.consumeLine(chatLine + "13964 -> a.mp4 ... done! [1.85 GB in 40s; 46.25 MB/s]")
	j.files["4385158554:13965"] = &fileProgress{key: "4385158554:13965", name: "b.mp4"}
	j.order = append(j.order, "4385158554:13965")

	if got := j.fraction(); got < 0.49 || got > 0.51 {
		t.Fatalf("总进度是 %v，一个文件完成、另一个还没开始应该是 0.5", got)
	}
	if _, _, bytes, total := j.totals(); bytes != half || total != 2*half {
		t.Fatalf("已传 %d，总量 %d", bytes, total)
	}
}

// TestOverallProgressReachesTheEndWhenEveryFileIsDone is the bookkeeping the
// bar relies on once the download is over.
func TestOverallProgressReachesTheEndWhenEveryFileIsDone(t *testing.T) {
	j := newJob("/tmp", []string{"https://t.me/a/1"})
	j.running = true
	j.consumeLine(chatLine + "13964 -> a.mp4 ... done! [1.85 GB in 40s; 46.25 MB/s]")
	j.consumeLine(chatLine + "13965 -> b.mp4 ... done! [600.00 MB in 20s; 30.00 MB/s]")

	j.running = false
	if got := j.fraction(); got != 1 {
		t.Fatalf("全部完成之后是 %v", got)
	}
}

// TestOverallProgressWithoutFiles stays at zero before the first tracker line.
func TestOverallProgressWithoutFiles(t *testing.T) {
	j := newJob("/tmp", []string{"https://t.me/a/1"})
	j.running = true
	if got := j.fraction(); got != 0 {
		t.Fatalf("没有文件时是 %v", got)
	}
}

// TestFileSizeIsReestimatedAsThePercentGetsPrecise guards the estimate itself:
// an early one percent sample lands within a few percent of the final size.
func TestFileSizeIsReestimatedAsThePercentGetsPrecise(t *testing.T) {
	j := newJob("/tmp", []string{"https://t.me/a/1"})
	j.consumeLine(chatLine + "13965 -> b.mp4 ...  0.1% [..........] [1.22 MB in 1s; 1.22 MB/s]")
	j.consumeLine(chatLine + "13965 -> b.mp4 ... 10.0% [##........] [185.00 MB in 10s; 18.50 MB/s]")
	fp := j.files["4385158554:13965"]
	if fp.total < 1900*1000*1000 || fp.total > 2000*1000*1000 {
		t.Fatalf("总量估计是 %d，应该在 1.85 GiB 附近", fp.total)
	}
}
