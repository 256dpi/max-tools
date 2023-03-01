package main

import "syscall"

func sysProcAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
