//go:build !windows

package main

// fora do Windows não há unidades com letra: a lista fica vazia
func listDrives() []Drive { return nil }
