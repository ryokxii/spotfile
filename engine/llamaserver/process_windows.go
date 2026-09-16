//go:build windows

package llamaserver

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// platform holds a Job Object configured with KILL_ON_JOB_CLOSE. Spotfile owns
// the only handle, so when Spotfile exits for any reason — including a crash —
// Windows closes the handle and terminates llama-server with it.
type platform struct {
	job windows.Handle
}

func startCommand(binary string, args []string, out io.Writer) (*exec.Cmd, platform, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, platform{}, fmt.Errorf("create job object: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return nil, platform{}, fmt.Errorf("configure job object: %w", err)
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout, cmd.Stderr = out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		_ = windows.CloseHandle(job)
		return nil, platform{}, err
	}

	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err == nil {
		err = windows.AssignProcessToJobObject(job, proc)
		_ = windows.CloseHandle(proc)
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = windows.CloseHandle(job)
		return nil, platform{}, fmt.Errorf("assign llama-server to job object: %w", err)
	}
	return cmd, platform{job: job}, nil
}

// terminate stops the server. Windows has no SIGTERM equivalent for console-less
// processes, so this is the same as kill.
func (pl platform) terminate(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}

func (pl platform) kill(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}

func (pl platform) release() {
	_ = windows.CloseHandle(pl.job)
}
