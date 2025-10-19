// Raspberry Pi Zero support for tamago/arm
// https://github.com/karlo195/tamago
//
// Copyright (c) the pizero package authors
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package pizero provides hardware initialization, automatically on import,
// for the Raspberry Pi Zero single board computer.
//
// This package is only meant to be used with `GOOS=tamago GOARCH=arm` as
// supported by the TamaGo framework for bare metal Go, see
// https://github.com/karlo195/tamago.
package pizero

import (
	_ "unsafe"

	"github.com/karlo195/tamago/board/raspberrypi"
	"github.com/karlo195/tamago/soc/bcm2835"
)

const peripheralBase = 0x20000000

type board struct{}

// Board provides access to the capabilities of the Pi Zero.
var Board pi.Board = &board{}

// Init takes care of the lower level initialization triggered early in runtime
// setup (post World start).
//
//go:linkname Init runtime.hwinit1
func Init() {
	// Defer to generic BCM2835 initialization, with Pi Zero
	// peripheral base address.
	bcm2835.Init(peripheralBase)
}
