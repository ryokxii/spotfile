//go:build !windows

package llamaserver

import (
	"io"
	"os/exec"
	"syscall"
)

// watchdogScript runs llama-server ("$@") under /bin/sh and ties its lifetime to
// Spotfile. The shell's stdin is a pipe whose only writer is Spotfile; when
// Spotfile exits for any reason — including a crash or SIGKILL — the kernel
// closes that pipe, the background read returns, and the server is terminated.
// Closing the pipe deliberately is also how Stop asks for a graceful exit.
//
// Background jobs in a non-interactive shell get /dev/null as stdin, so the
// pipe is duplicated to fd 3 first and the watcher reads from that explicitly.
const watchdogScript = `exec 3<&0
"$@" </dev/null 3<&- &
server=$!
( read -r _ <&3; kill -TERM "$server" 2>/dev/null ) &
watcher=$!
wait "$server"
status=$?
kill "$watcher" 2>/dev/null
exit $status`

type platform struct {
	stdin io.WriteCloser
}

func startCommand(binary string, args []string, out io.Writer) (*exec.Cmd, platform, error) {
	shArgs := append([]string{"-c", watchdogScript, "llama-server", binary}, args...)
	cmd := exec.Command("/bin/sh", shArgs...)
	cmd.Stdout, cmd.Stderr = out, out
	// Own process group, so a forced kill reaches the shell, the server and the
	// watcher together.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, platform{}, err
	}
	if err := cmd.Start(); err != nil {
		return nil, platform{}, err
	}
	return cmd, platform{stdin: stdin}, nil
}

// terminate closes the lifeline pipe; the watchdog sends SIGTERM to the server.
func (pl platform) terminate(*exec.Cmd) {
	_ = pl.stdin.Close()
}

// kill force-kills the whole process group.
func (pl platform) kill(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func (pl platform) release() {
	_ = pl.stdin.Close()
}
