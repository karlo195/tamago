// x86-64 processor support
// https://github.com/karlo195/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package amd64

import (
	_ "unsafe"

	"github.com/karlo195/tamago/internal/rng"
)

// defined in rng.s
func rdrand() uint32

// GetRandomData returns len(b) random bytes gathered from the RDRAND instruction.
func GetRandomData(b []byte) {
	read := 0
	need := len(b)

	for read < need {
		read = rng.Fill(b, read, rdrand())
	}
}

//go:linkname initRNG runtime.initRNG
func initRNG() {
	rng.GetRandomDataFn = GetRandomData
}
