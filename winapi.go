package main

import (
	"log"
	"syscall"
	"unsafe"

	_ "net/http/pprof"
)

type (
	HANDLE  uintptr
	HWND    HANDLE
	HGLOBAL HANDLE
)

type WinApi struct {
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEXW struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSm     syscall.Handle
}

const (
	WM_DESTROY         = 0x0002
	WM_CLOSE           = 0x0010
	WM_CLIPBOARDUPDATE = 0x031D
	WS_MINIMIZE        = 0x20000000
	CF_TEXT            = 1
	CF_UNICODETEXT     = 13
	IDC_ARROW          = 32512
	COLOR_WINDOW       = 5
	PM_REMOVE          = 0x001
	SW_USE_DEFAULT     = 0x80000000
	HWND_MESSAGE       = ^HWND(2)
	GMEM_MOVEABLE      = 0x0002
)

var (
	user32                         = syscall.MustLoadDLL("user32")
	procCreateWindowExW            = user32.MustFindProc("CreateWindowExW")
	procDestroyWindow              = user32.MustFindProc("DestroyWindow")
	procAddClipboardFormatListener = user32.MustFindProc("AddClipboardFormatListener")
	procGetMessageW                = user32.MustFindProc("GetMessageW")
	procOpenClipboard              = user32.MustFindProc("OpenClipboard")
	procCloseClipboard             = user32.MustFindProc("CloseClipboard")
	procGetClipboardData           = user32.MustFindProc("GetClipboardData")
	procRegisterClassExW           = user32.MustFindProc("RegisterClassExW")
	procLoadCursorW                = user32.MustFindProc("LoadCursorW")
	procDispatchMessageW           = user32.MustFindProc("DispatchMessageW")
	procDefWindowProcW             = user32.MustFindProc("DefWindowProcW")
	procGetClipboardFormatName     = user32.MustFindProc("GetClipboardFormatNameW")
	procEmptyClipboard             = user32.MustFindProc("EmptyClipboard")
	procSetClipboardData           = user32.MustFindProc("SetClipboardData")
)

var (
	kernel32            = syscall.MustLoadDLL("kernel32.dll")
	procGetModuleHandle = kernel32.MustFindProc("GetModuleHandleW")
	procGlobalLock      = kernel32.MustFindProc("GlobalLock")
	procGlobalUnlock    = kernel32.MustFindProc("GlobalUnlock")
	procGlobalAlloc     = kernel32.MustFindProc("GlobalAlloc")
	procMoveMemory      = kernel32.MustFindProc("RtlMoveMemory")
	procGlobalFree      = kernel32.MustFindProc("GlobalFree")
)

func (winApi *WinApi) globalFree(hMem HGLOBAL) {
	ret, _, _ := procGlobalFree.Call(uintptr(hMem))

	if ret != 0 {
		panic("GlobalFree: failed")
	}
}

func (winApi *WinApi) moveMemory(destination, source unsafe.Pointer, length uintptr) {
	procMoveMemory.Call(
		uintptr(unsafe.Pointer(destination)),
		uintptr(source),
		length)
}

func (winApi *WinApi) globalAlloc(uFlags uint, dwBytes uintptr) HGLOBAL {
	ret, _, _ := procGlobalAlloc.Call(uintptr(uFlags), dwBytes)

	if ret == 0 {
		panic("GlobalAlloc failed")
	}

	return HGLOBAL(ret)
}

func (winApi *WinApi) setClipboardData(uFormat uint, hMem HANDLE) HANDLE {
	ret, _, _ := procSetClipboardData.Call(
		uintptr(uFormat),
		uintptr(hMem))
	return HANDLE(ret)
}

func (winApi *WinApi) emptyClipboard() bool {
	ret, _, _ := procEmptyClipboard.Call()
	return ret != 0
}

func (winApi *WinApi) GetClipboardFormatName(format uint) (string, bool) {
	cchMaxCount := 255
	buf := make([]uint16, cchMaxCount)
	ret, _, err := procGetClipboardFormatName.Call(
		uintptr(format),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(cchMaxCount))

	if err != nil {
		log.Fatalln(err)
		return "Error", false
	}

	if ret > 0 {
		return syscall.UTF16ToString(buf), true
	}

	return "Requested format does not exist or is predefined", false
}

func (winApi *WinApi) defWindowProc(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(
		uintptr(hwnd),
		uintptr(msg),
		uintptr(wparam),
		uintptr(lparam),
	)
	return uintptr(ret)
}

func (winApi *WinApi) dispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessageW.Call(
		uintptr(unsafe.Pointer(msg)))

	return ret

}

func (winApi *WinApi) loadCursorResource(cursorName uint32) (syscall.Handle, error) {
	ret, _, err := procLoadCursorW.Call(
		uintptr(0),
		uintptr(uint16(cursorName)),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func (winApi *WinApi) registerClassEx(wcx *WNDCLASSEXW) (uint16, error) {
	ret, _, err := procRegisterClassExW.Call(
		uintptr(unsafe.Pointer(wcx)),
	)
	if ret == 0 {
		return 0, err
	}
	return uint16(ret), nil
}

func (winApi *WinApi) globalLock(hMem HGLOBAL) unsafe.Pointer {
	ret, _, _ := procGlobalLock.Call(uintptr(hMem))

	if ret == 0 {
		panic("GlobalLock failed")
	}

	return unsafe.Pointer(ret)
}

func (winApi *WinApi) globalUnlock(hMem HGLOBAL) bool {
	ret, _, _ := procGlobalUnlock.Call(uintptr(hMem))
	return ret != 0
}

func (winApi *WinApi) openClipboard() bool {
	ret, _, _ := procOpenClipboard.Call()
	return ret != 0
}

func (winApi *WinApi) closeClipboard() bool {
	ret, _, _ := procCloseClipboard.Call()
	return ret != 0
}

func (winApi *WinApi) getClipboardData(uFormat int) (syscall.Handle, error) {
	ret, _, err := procGetClipboardData.Call(uintptr(uFormat))
	return syscall.Handle(ret), err
}

func (winApi *WinApi) getModuleHandle() (syscall.Handle, error) {
	ret, _, err := procGetModuleHandle.Call(uintptr(0))
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func (winApi *WinApi) createWindowExW(className, windowName string, style uint32, x, y, width, height uint32, parent HWND, menu, instance syscall.Handle) (HWND, error) {
	classNamePtr, classNameErr := syscall.UTF16PtrFromString(className)
	if classNameErr != nil {
		log.Fatalln("createWindowExW: failed to create utf16ptr from className")
	}

	windowNamePtr, windowNameErr := syscall.UTF16PtrFromString(windowName)
	if windowNameErr != nil {
		log.Fatalln("createWindowExW: failed to create utf16ptr from modulename")
	}

	ret, _, err := procCreateWindowExW.Call(
		uintptr(0),
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(windowNamePtr)),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		uintptr(0),
	)
	if ret == 0 {
		return 0, err
	}
	return HWND(ret), nil
}

func (winApi *WinApi) destroyWindow(hwnd HWND) bool {
	ret, _, _ := procDestroyWindow.Call(
		uintptr(hwnd))

	return ret != 0
}

func (winApi *WinApi) addClipboardFormatListener(hwnd HWND) bool {
	ret, _, _ := procAddClipboardFormatListener.Call(
		uintptr(hwnd))
	return ret != 0
}

func (winApi *WinApi) getMessage(msg *MSG, hwnd HWND, msgFilterMin, msgFilterMax uint32) bool {
	ret, _, _ := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))

	return ret != 0
}
