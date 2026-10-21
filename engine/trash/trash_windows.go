//go:build windows

package trash

import (
	"fmt"
	"syscall"
	"unsafe"
)

// SHFileOperationW flags: delete with undo (Recycle Bin), no UI.
const (
	foDelete          = 0x0003
	fofSilent         = 0x0004
	fofNoConfirmation = 0x0010
	fofAllowUndo      = 0x0040
	fofNoErrorUI      = 0x0400
)

// shFileOpStruct mirrors SHFILEOPSTRUCTW (natural alignment on 64-bit Windows).
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperationW = syscall.NewLazyDLL("shell32.dll").NewProc("SHFileOperationW")

// Trash moves path to the Recycle Bin. With FOF_ALLOWUNDO the shell refuses
// rather than deleting permanently when the volume has no Recycle Bin.
func (System) Trash(path string) error {
	from, err := syscall.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0) // pFrom is a list ending with an extra NUL
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	if code, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op))); code != 0 {
		return fmt.Errorf("Recycle Bin refused the file (code %d)", code)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("moving to the Recycle Bin was cancelled")
	}
	return nil
}
