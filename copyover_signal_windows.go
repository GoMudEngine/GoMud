//go:build windows

// Windows does not support the signal being used, let alone the rest of copyover functionality.

package main

func startCopyoverSignalHandler() {}
