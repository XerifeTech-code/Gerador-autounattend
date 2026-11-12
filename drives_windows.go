//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	k32                  = syscall.NewLazyDLL("kernel32.dll")
	pGetLogicalDrives    = k32.NewProc("GetLogicalDrives")
	pGetDriveTypeW       = k32.NewProc("GetDriveTypeW")
	pGetVolumeInfoW      = k32.NewProc("GetVolumeInformationW")
	pGetDiskFreeSpaceExW = k32.NewProc("GetDiskFreeSpaceExW")
	pSetErrorMode        = k32.NewProc("SetErrorMode")
)

const (
	driveRemovable = 2
	driveFixed     = 3
)

// listDrives devolve pendrives (removíveis) e qualquer unidade (menos C:) que seja mídia de instalação do Windows.
func listDrives() []Drive {
	// sem janelas "Não há disco na unidade" para leitores de cartão vazios
	old, _, _ := pSetErrorMode.Call(0x0001 | 0x8000)
	defer pSetErrorMode.Call(old)

	mask, _, _ := pGetLogicalDrives.Call()
	var out []Drive
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		if root == `C:\` || root == `A:\` || root == `B:\` {
			continue
		}
		rp, _ := syscall.UTF16PtrFromString(root)
		t, _, _ := pGetDriveTypeW.Call(uintptr(unsafe.Pointer(rp)))
		if t != driveRemovable && t != driveFixed {
			continue
		}
		var free, total uint64
		r1, _, _ := pGetDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), 0)
		if r1 == 0 {
			continue // sem mídia
		}
		lbl := make([]uint16, 261)
		pGetVolumeInfoW.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&lbl[0])), uintptr(len(lbl)), 0, 0, 0, 0, 0)
		d := Drive{Root: root, Label: syscall.UTF16ToString(lbl), Removable: t == driveRemovable, Free: free, Total: total}
		d.Installer = isInstaller(root)
		d.Ventoy = isVentoy(root)
		d.HasXML = fileExists(root + "autounattend.xml")
		if d.Removable || d.Installer {
			out = append(out, d)
		}
	}
	return out
}
