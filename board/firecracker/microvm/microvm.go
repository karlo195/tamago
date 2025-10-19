// Firecracker microvm support for tamago/amd64
// https://github.com/karlo195/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package microvm provides hardware initialization, automatically on import,
// for a Firecracker microvm configured with a single x86_64 core.
//
// This package is only meant to be used with `GOOS=tamago GOARCH=amd64` as
// supported by the TamaGo framework for bare metal Go, see
// https://github.com/karlo195/tamago.
package microvm

import (
	"runtime"
	_ "unsafe"

	"github.com/karlo195/tamago/amd64"
	"github.com/karlo195/tamago/dma"
	"github.com/karlo195/tamago/kvm/pvclock"
	"github.com/karlo195/tamago/soc/intel/ioapic"
	"github.com/karlo195/tamago/soc/intel/uart"
)

const (
	dmaStart = 0x50000000
	dmaSize  = 0x10000000 // 256MB
)

// Peripheral registers
const (
	// Communication port
	COM1 = 0x3f8

	// Intel I/O Programmable Interrupt Controller
	IOAPIC0_BASE = 0xfec00000

	// VirtIO Memory-mapped I/O
	VIRTIO_MMIO_BASE = 0xc0000000

	// VirtIO Networking
	VIRTIO_NET0_BASE = VIRTIO_MMIO_BASE + 0x2000
	VIRTIO_NET0_IRQ  = 6
)

// Peripheral instances
var (
	// CPU instance(s)
	AMD64 = &amd64.CPU{
		// required before Init()
		TimerMultiplier: 1,
	}

	// I/O APIC - GSI 0-23
	IOAPIC0 = &ioapic.IOAPIC{
		Base: IOAPIC0_BASE,
	}

	// Serial port
	UART0 = &uart.UART{
		Index: 1,
		Base:  COM1,
	}
)

//go:linkname nanotime1 runtime.nanotime1
func nanotime1() int64 {
	return AMD64.GetTime()
}

// Init takes care of the lower level initialization triggered early in runtime
// setup (post World start).
//
//go:linkname Init runtime.hwinit1
func Init() {
	// initialize CPU
	AMD64.Init()

	// initialize I/O APIC
	IOAPIC0.Init()
	// initialize serial console
	UART0.Init()

	runtime.Exit = func(_ int32) {
		AMD64.Reset()
	}
}

func init() {
	// trap CPU exceptions
	AMD64.EnableExceptions()

	// initialize APs
	AMD64.InitSMP(-1)

	// allocate global DMA region
	dma.Init(dmaStart, dmaSize)

	// initialize KVM pvclock as needed
	pvclock.Init(AMD64)
}
