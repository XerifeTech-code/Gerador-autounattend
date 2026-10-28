//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

func launchWindow(url string) *exec.Cmd {
	fmt.Println("Interface em:", url)
	return nil
}

func killWindow(cmd *exec.Cmd) {}

func revealFile(path string) {}

func openURL(url string) { fmt.Println("Abrir:", url) }
