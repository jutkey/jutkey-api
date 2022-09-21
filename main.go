package main

import (
	"jutkey-server/cmd"
	"runtime"
)

func main() {
	runtime.LockOSThread()
	cmd.Execute()
}
