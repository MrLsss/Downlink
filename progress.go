package main

import (
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ansiRE matches the escape sequences a terminal program writes for colors,
// cursor moves and the like.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]|\x1b[()][A-Za-z0-9]|\x1b[=>78]")

var (
	// msgRE reads the tracker message tdl writes: the chat, the message and
	// the file it goes to.
	msgRE     = regexp.MustCompile(`^(.+)\((\d+)\):(\d+) -> (.+)$`)
	pctRE     = regexp.MustCompile(`^\s*([0-9]+(?:\.[0-9]+)?)%`)
	overallRE = regexp.MustCompile(`^\[[#.\s]*\] \[([^\]]*)\]$`)
)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// fileProgress is one file of a download.
type fileProgress struct {
	key    string
	name   string
	path   string
	value  int64
	total  int64
	speed  int64
	frac   float64
	done   bool
	failed bool
}

// job is one download run of tdl and what its progress lines reported.
type job struct {
	running    bool
	cancelling bool
	status     string
	startedAt  time.Time
	endedAt    time.Time
	dir        string
	urls       []string
	files      map[string]*fileProgress
	order      []string
	speed      int64
	errText    string
	logTail    []string
	pending    string
	sess       *ptySession
}

func newJob(dir string, urls []string) *job {
	return &job{
		running:   true,
		startedAt: time.Now(),
		dir:       dir,
		urls:      urls,
		files:     map[string]*fileProgress{},
	}
}

// consume reads a chunk of the command's output.
func (j *job) consume(chunk string) {
	j.pending += stripANSI(chunk)
	for {
		i := strings.IndexByte(j.pending, '\n')
		if i < 0 {
			break
		}
		line := j.pending[:i]
		j.pending = j.pending[i+1:]
		j.consumeLine(line)
	}
	if len(j.pending) > 1<<16 {
		j.pending = j.pending[len(j.pending)-1<<12:]
	}
}

func (j *job) consumeLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if m := overallRE.FindStringSubmatch(line); m != nil {
		j.setOverall(m[1])
		return
	}
	if i := strings.LastIndex(line, " ... "); i > 0 {
		j.consumeFile(strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+len(" ... "):]))
		return
	}
	j.note(line)
}

func (j *job) consumeFile(message, rest string) {
	fp := j.file(message)
	switch {
	case strings.HasPrefix(rest, "done!"):
		fp.done = true
		fp.frac = 1
		fp.speed = 0
		if v, _ := parseProgressDetail(lastBracket(rest)); v > 0 {
			fp.value, fp.total = v, v
		}
	case strings.HasPrefix(rest, "failed!"):
		fp.failed = true
		fp.speed = 0
	default:
		if m := pctRE.FindStringSubmatch(rest); m != nil {
			if pct, err := strconv.ParseFloat(m[1], 64); err == nil {
				fp.frac = math.Min(1, pct/100)
			}
		}
		value, speed := parseProgressDetail(lastBracket(rest))
		if value > 0 {
			fp.value = value
		}
		if speed > 0 {
			fp.speed = speed
		}
	}
	// tdl never prints the size of a file, only how much of it is here and
	// how far along it is. The percent keeps one decimal, so a sample below
	// one percent divides by a rounded fraction and lands far off; from one
	// percent up the estimate is good, and it gets better as the file fills.
	if !fp.done && fp.frac >= 0.01 && fp.value > 0 {
		fp.total = int64(float64(fp.value) / fp.frac)
	}
}

// file returns the progress of the file a tracker line is about.
func (j *job) file(message string) *fileProgress {
	key, name, path := message, message, ""
	if m := msgRE.FindStringSubmatch(message); m != nil {
		key = m[2] + ":" + m[3]
		path = strings.TrimSpace(m[4])
		name = filepath.Base(path)
	}
	if fp, ok := j.files[key]; ok {
		if fp.path == "" && path != "" {
			fp.path, fp.name = path, name
		}
		return fp
	}
	fp := &fileProgress{key: key, name: name, path: path}
	j.files[key] = fp
	j.order = append(j.order, key)
	return fp
}

// setOverall reads the line of the download as a whole. The left half of the
// bracket is the time already spent, so the speed is all this line gives us.
func (j *job) setOverall(detail string) {
	_, tail, ok := strings.Cut(detail, "; ")
	if !ok {
		return
	}
	if v := parseBytes(strings.TrimSuffix(strings.TrimSpace(tail), "/s")); v > 0 {
		j.speed = v
	}
}

// remaining is how long the download still needs at the current speed.
func (j *job) remaining() string {
	if !j.running || j.speed <= 0 {
		return ""
	}
	_, _, bytes, totalBytes := j.totals()
	if totalBytes <= 0 || bytes >= totalBytes {
		return ""
	}
	return humanClock(time.Duration(float64(totalBytes-bytes) / float64(j.speed) * float64(time.Second)))
}

func (j *job) note(line string) {
	j.logTail = append(j.logTail, line)
	if len(j.logTail) > 80 {
		j.logTail = j.logTail[len(j.logTail)-80:]
	}
	// tdl prints the cause under the "Error: " line, one "  - " line per
	// level, so the reason to show comes from the whole report.
	if e := lastError(strings.Join(j.logTail, "\n")); e != "" {
		j.errText = e
	}
}

// totals counts the files and the bytes of the download. A file never counts
// for more than its own size, so a size estimate that came out small cannot
// make the download look finished. A file whose size is not known yet takes
// the size of an average file, so the bar does not sit at the end while more
// files are still waiting.
func (j *job) totals() (done, count int, bytes, totalBytes int64) {
	var sizedBytes int64
	var sized int
	for _, key := range j.order {
		fp := j.files[key]
		count++
		if fp.done {
			done++
		}
		if fp.total > 0 {
			sized++
			sizedBytes += fp.total
			totalBytes += fp.total
			if fp.value < fp.total {
				bytes += fp.value
			} else {
				bytes += fp.total
			}
			continue
		}
		bytes += fp.value
	}
	if waiting := count - sized; waiting > 0 && sized > 0 {
		totalBytes += sizedBytes / int64(sized) * int64(waiting)
	}
	return done, count, bytes, totalBytes
}

// fraction is how much of the download finished, from 0 to 1. Files are
// weighed by their size, so one file of two finished is halfway only when both
// are the same size.
func (j *job) fraction() float64 {
	done, count, bytes, totalBytes := j.totals()
	if count == 0 {
		return 0
	}
	if done == count && !j.running {
		return 1
	}
	if totalBytes > 0 {
		return math.Min(1, float64(bytes)/float64(totalBytes))
	}
	return float64(done) / float64(count)
}

// summary is the line under the download's progress bar.
func (j *job) summary() string {
	done, count, bytes, totalBytes := j.totals()
	s := itoa(int64(done)) + " / " + itoa(int64(count)) + " 个文件"
	if totalBytes > 0 {
		s += " · " + humanBytes(bytes) + " / " + humanBytes(totalBytes)
	}
	if j.running {
		if r := humanRate(j.speed); r != "" {
			s += " · " + r
		}
		if r := j.remaining(); r != "" {
			s += " · 剩余 " + r
		}
	}
	return s
}

// lastBracket returns what the last bracketed group of a line holds.
func lastBracket(s string) string {
	end := strings.LastIndexByte(s, ']')
	if end < 0 {
		return ""
	}
	start := strings.LastIndexByte(s[:end], '[')
	if start < 0 {
		return ""
	}
	return s[start+1 : end]
}

// parseProgressDetail reads the value and the speed of a tracker's detail,
// such as "57.22 MB in 301ms; 189.87 MB/s".
func parseProgressDetail(detail string) (value, speed int64) {
	if detail == "" {
		return 0, 0
	}
	head, tail, ok := strings.Cut(detail, " in ")
	if !ok {
		return 0, 0
	}
	value = parseBytes(head)
	if i := strings.LastIndex(tail, "; "); i >= 0 {
		speed = parseBytes(strings.TrimSuffix(strings.TrimSpace(tail[i+2:]), "/s"))
	}
	return value, speed
}

// parseBytes reads a size such as "1.34 MB" or "900 B".
func parseBytes(s string) int64 {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) != 2 {
		return 0
	}
	n, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	mult := map[string]float64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
		"PB": 1024 * 1024 * 1024 * 1024 * 1024,
	}[strings.ToUpper(fields[1])]
	if mult == 0 {
		return 0
	}
	return int64(n * mult)
}
