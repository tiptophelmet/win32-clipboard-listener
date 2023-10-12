package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"
	"unsafe"
)

type WindowsClipboardListener struct {
	winApi                      *WinApi
	moduleHandler               syscall.Handle
	cursorHandler               syscall.Handle
	wndClassName                string
	wndHandler                  *HWND
	message                     MSG
	wmClipboardUpdateHandlerPtr uintptr
	copied                      string
}

func initWindowsClipboardListener() *WindowsClipboardListener {
	wcl := &WindowsClipboardListener{
		winApi: &WinApi{},
	}

	wcl.wmClipboardUpdateHandlerPtr = syscall.NewCallback(func(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case WM_CLIPBOARDUPDATE:
			wcl.tryOpenWinApiClipboard()
			defer wcl.winApi.closeClipboard()

			if wcl.message.WParam == 0x4 {
				// Handle SetClipboardData case
				// Since SetClipboardData is also a WM_CLIPBOARDUPDATE,
				// we need to ignore it, since there is no user-copied data.
				//
				// We called SetClipbardData to bring back originally copied item,
				// that was erased by EmptyClipboard (for decreasing memory consumption)
				return 0
			}

			if wcl.message.WParam == 0x0 {
				// Handle EmptyClipboard case
				// Since EmptyClipboard is also a WM_CLIPBOARDUPDATE,
				// we need to ignore it, since there is no user-copied data.
				return 0
			}

			hData, err := wcl.winApi.getClipboardData(CF_UNICODETEXT)
			if hData == 0 {
				// TODO: If user manually deletes an item from clipboard, 
				// app should smoothly handle this case & proceed with next messages,
				// since it is not a fatal error.
				//
				// WM_CLIPBOARDUPDATE is triggered both when items are added
				// or deleted from the clipboard.
				fmt.Println("GetClipboardData failed: ", err)
				return 0
			}

			dataPtr := wcl.winApi.globalLock(HGLOBAL(hData))

			wcl.copied = syscall.UTF16ToString((*[1 << 20]uint16)(dataPtr)[:])
			wcl.winApi.globalUnlock(HGLOBAL(hData))

			wcl.winApi.emptyClipboard()

			utf16, err := syscall.UTF16FromString(wcl.copied)
			if err != nil {
				return 0
			}

			hMem := wcl.winApi.globalAlloc(GMEM_MOVEABLE, uintptr(len(utf16)*2))
			dataPtr = wcl.winApi.globalLock(HGLOBAL(hMem))

			wcl.winApi.moveMemory(dataPtr, unsafe.Pointer(&utf16[0]), uintptr(len(utf16)*2))
			wcl.winApi.globalUnlock(HGLOBAL(hMem))

			if wcl.winApi.setClipboardData(CF_UNICODETEXT, HANDLE(hMem)) == 0 {
				defer wcl.winApi.globalFree(hMem)

				log.Fatalln("SetClipboardData: error")
			} else {
				fmt.Println("SetClipboardData: OK")
			}

			fmt.Printf("Copied at %v\n\r", time.Now())
			return 0
		default:
			return wcl.winApi.defWindowProc(hwnd, msg, wparam, lparam)
		}
	})

	return wcl
}

func (wcl *WindowsClipboardListener) tryOpenWinApiClipboard() {
	if !wcl.winApi.openClipboard() {
		wcl.winApi.closeClipboard()
		wcl.tryOpenWinApiClipboard()
	}
}

func (wcl *WindowsClipboardListener) initModuleHandler() {
	moduleHandler, err := wcl.winApi.getModuleHandle()
	if err != nil {
		log.Fatalln("initModuleHandler: failed to init module handler")
	}

	wcl.moduleHandler = moduleHandler
}

func (wcl *WindowsClipboardListener) initCursor() {
	cursorHandler, err := wcl.winApi.loadCursorResource(IDC_ARROW)
	if err != nil {
		log.Fatalln(err)
	}

	wcl.cursorHandler = cursorHandler
}

func (wcl *WindowsClipboardListener) setWndClassName(className string) {
	wcl.wndClassName = className
}

func (wcl *WindowsClipboardListener) registerWndClass() {
	wndClassNamePtr, err := syscall.UTF16PtrFromString(wcl.wndClassName)
	if err != nil {
		log.Fatalln("registerWndClass: failed to create utf16ptr from wndClassName")
	}

	wndClass := WNDCLASSEXW{
		wndProc:    wcl.wmClipboardUpdateHandlerPtr,
		instance:   wcl.moduleHandler,
		cursor:     wcl.cursorHandler,
		background: COLOR_WINDOW + 1,
		className:  wndClassNamePtr,
	}
	wndClass.size = uint32(unsafe.Sizeof(wndClass))

	if _, err := wcl.winApi.registerClassEx(&wndClass); err != nil {
		log.Fatalln("registerClassEx: ", err)
	}
}

func (wcl *WindowsClipboardListener) initWindow() {
	hWnd, err := wcl.winApi.createWindowExW(
		wcl.wndClassName,
		"Simple Go Window!",
		WS_MINIMIZE,
		SW_USE_DEFAULT,
		SW_USE_DEFAULT,
		SW_USE_DEFAULT,
		SW_USE_DEFAULT,
		HWND_MESSAGE,
		0,
		wcl.moduleHandler,
	)

	if err != nil {
		log.Fatalln("CreateWindowExW: ", err)
	}

	wcl.wndHandler = &hWnd
}

func (wcl *WindowsClipboardListener) initClipboardFormatListener() {
	wcl.winApi.addClipboardFormatListener(*wcl.wndHandler)
}

func setupProfiler() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}

func (wcl *WindowsClipboardListener) setupListener() {
	wcl.initModuleHandler()
	wcl.initCursor()
	wcl.setWndClassName("goClass")
	wcl.registerWndClass()
	wcl.initWindow()
	wcl.initClipboardFormatListener()
}

func (wcl *WindowsClipboardListener) listen() {
	var newMessage bool

	for {
		newMessage = wcl.winApi.getMessage(&wcl.message, *wcl.wndHandler, WM_CLIPBOARDUPDATE, WM_CLIPBOARDUPDATE)
		if newMessage {
			wcl.winApi.dispatchMessage(&wcl.message)
		}
	}
}

func main() {
	go setupProfiler()

	wcl := initWindowsClipboardListener()
	wcl.setupListener()
	defer wcl.winApi.destroyWindow(*wcl.wndHandler)

	wcl.listen()
}
