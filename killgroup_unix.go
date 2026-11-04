//go:build !windows

package main

import "syscall"

// killGroup ends everything the command started, when the command leads its
// own process group: on a pseudo terminal it is a session leader, so its group
// id is its own pid. A command that shares this process's group is left to
// process.Kill, since signalling its group would end this program too.
func killGroup(pid int) {
	g, err := syscall.Getpgid(pid)
	if err != nil || g != pid {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
