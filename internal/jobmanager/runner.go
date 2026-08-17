package jobmanager

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// RawEvent is one line of subprocess output, already classified by which
// byte terminated it: Kind "progress" for a bare '\r' redraw (tqdm), "log"
// for a real '\n'-terminated line or an unterminated fragment flushed at EOF.
type RawEvent struct {
	Stream string // "stdout" | "stderr"
	Text   string
	Kind   string // "log" | "progress"
}

// Runner runs mlx_whisper (or any command) as a subprocess, streaming its
// stdout/stderr line-by-line to onEvent, and supports cancellation.
type Runner struct {
	cmd  *exec.Cmd
	done chan struct{}
}

// NewRunner builds a Runner. args must NOT include the binary path itself.
// Passing args as a []string (never through a shell) means filenames with
// spaces or special characters (e.g. "My Recording 2026-08-17.mov") are
// handled safely by the OS -- no manual escaping, no shell-injection risk.
func NewRunner(binPath string, args []string, dir string) *Runner {
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &Runner{cmd: cmd, done: make(chan struct{})}
}

// Run starts the subprocess and blocks until it exits, calling onEvent for
// every line/redraw seen on stdout or stderr. It returns the process's exit
// error (nil on a clean exit).
func (r *Runner) Run(onEvent func(RawEvent)) error {
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := r.cmd.Start(); err != nil {
		close(r.done)
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	// tqdm writes its progress bar to stderr by default (not stdout), so both
	// pipes get the exact same line-splitting + classification treatment --
	// never assume progress only arrives on one particular stream.
	go streamPipe(stdout, "stdout", onEvent, &wg)
	go streamPipe(stderr, "stderr", onEvent, &wg)
	wg.Wait()

	err = r.cmd.Wait()
	close(r.done)
	return err
}

func streamPipe(pipe io.Reader, stream string, onEvent func(RawEvent), wg *sync.WaitGroup) {
	defer wg.Done()
	s := &lineSplitter{}
	scanner := bufio.NewScanner(pipe)
	scanner.Split(s.split)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" && s.lastTerm == '\r' {
			// tqdm sometimes emits a bare '\r' with nothing new to show; skip
			// the empty redraw rather than publishing a no-op progress event.
			continue
		}
		kind := "log"
		if s.lastTerm == '\r' {
			kind = "progress"
		}
		onEvent(RawEvent{Stream: stream, Text: text, Kind: kind})
	}
}

// ForceKill sends SIGKILL to the process group immediately, no grace period.
// Used on server shutdown (the process itself is exiting right now, so
// there's no point waiting) -- day-to-day user-initiated cancellation should
// use Cancel instead, which gives the subprocess a chance to exit cleanly.
func (r *Runner) ForceKill() {
	if r.cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL)
}

// Cancel sends SIGTERM to the whole process group (so any child processes
// mlx_whisper/python spawn die too, not just the parent), then escalates to
// SIGKILL after a grace period if the process hasn't exited by then. It
// returns immediately; the escalation runs in its own goroutine.
func (r *Runner) Cancel() {
	if r.cmd.Process == nil {
		return
	}
	pgid := r.cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	go func() {
		select {
		case <-r.done:
			return
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
	}()
}
