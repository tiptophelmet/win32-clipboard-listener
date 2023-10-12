# win32-clipboard-listener
Memory-efficient Win32 Clipboard Listener

This Golang package solves a memory leak in existing cross-platform Golang solution (golang.design/x/clipboard). Consecutive copy operations, depending on content size, were blocking Go garbage collector to free allocated space for previous copies.

By using technique provided in a revised variant of clipboard listener memory is reused in a correct way, regardless of copied content size.