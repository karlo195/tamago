// Linux user space support
// https://github.com/karlo195/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package linux_user provides support for using `GOOS=tamago` in Linux user
// space.
//
// This package is only meant to be used with `GOOS=tamago` as supported by the
// TamaGo framework for bare metal Go, see https://github.com/karlo195/tamago.
package linux_user

import (
	"runtime"
	_ "unsafe"
)

//go:linkname ramStart runtime.ramStart
var ramStart uint64 = 0x80000000

//go:linkname ramSize runtime.ramSize
var ramSize uint64 = 0x20000000 // 512MB

//go:linkname ramStackOffset runtime.ramStackOffset
var ramStackOffset uint64 = 0x100

// defined in syscall_*.s
func sys_exit(code int32)
func sys_write(c *byte)
func sys_clock_gettime() (ns int64)
func sys_getrandom(b []byte, n int)

//go:linkname nanotime1 runtime.nanotime1
func nanotime1() int64 {
	return sys_clock_gettime()
}

//go:linkname initRNG runtime.initRNG
func initRNG() {}

//go:linkname getRandomData runtime.getRandomData
func getRandomData(b []byte) {
	sys_getrandom(b, len(b))
}

// preallocated memory to avoid malloc during panic
var a [1]byte

//go:linkname printk runtime.printk
func printk(c byte) {
	a[0] = c
	sys_write(&a[0])
}

//go:linkname hwinit0 runtime.hwinit0
func hwinit0() {
	runtime.Bloc = uintptr(ramStart)
}

//go:linkname hwinit1 runtime.hwinit1
func hwinit1() {
	runtime.Exit = sys_exit
}
