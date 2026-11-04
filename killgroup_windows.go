//go:build windows

package main

// killGroup has nothing extra to do on Windows: killing the process ends it.
func killGroup(int) {}
