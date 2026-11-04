package main

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
)

// The terminal tdl runs on is wide, so that its progress lines keep the whole
// file name instead of cutting it short at the width of a narrow one.
const (
	ptyCols = 1000
	ptyRows = 60
)

// ptySession runs a command on a pseudo terminal, which tdl's interactive
// prompts need.
type ptySession struct {
	p    pty.Pty
	cmd  *pty.Cmd
	done chan struct{}
	once sync.Once
}

func startPTY(bin string, args []string, onChunk func(string)) (*ptySession, error) {
	p, err := pty.New()
	if err != nil {
		return nil, err
	}
	if err := p.Resize(ptyCols, ptyRows); err != nil {
		p.Close()
		return nil, err
	}
	cmd := p.Command(bin, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err := cmd.Start(); err != nil {
		p.Close()
		return nil, err
	}
	s := &ptySession{p: p, cmd: cmd, done: make(chan struct{})}
	go s.read(onChunk)
	return s, nil
}

// read hands every chunk the command writes to onChunk.
func (s *ptySession) read(onChunk func(string)) {
	defer close(s.done)
	buf := make([]byte, 32*1024)
	// survey asks the terminal where the cursor is, and waits for the answer
	// before it reads what the user types. The query arrives in one piece on
	// nearly every read; the three bytes kept here catch it when it does not.
	const query = "\x1b[6n"
	tail := ""
	for {
		n, err := s.p.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			if strings.Contains(tail+chunk, query) {
				s.p.Write([]byte("\x1b[24;1R"))
			}
			tail = lastBytes(chunk, len(query)-1)
			onChunk(chunk)
		}
		if err != nil {
			return
		}
	}
}

func lastBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func (s *ptySession) write(text string) error {
	_, err := s.p.Write([]byte(text))
	return err
}

// interrupt types Ctrl-C, which the terminal turns into a signal.
func (s *ptySession) interrupt() { s.write("\x03") }

// wait waits for the command to exit and for the rest of its output to arrive.
func (s *ptySession) wait() error {
	err := s.cmd.Wait()
	select {
	case <-s.done:
	case <-time.After(200 * time.Millisecond):
		s.close()
		<-s.done
	}
	return err
}

func (s *ptySession) kill() {
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
}

func (s *ptySession) close() {
	s.once.Do(func() { s.p.Close() })
}
