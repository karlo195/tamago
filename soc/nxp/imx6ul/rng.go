// NXP i.MX6UL RNG initialization
// https://github.com/karlo195/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package imx6ul

import (
	"encoding/binary"
	"time"
	_ "unsafe"

	"github.com/karlo195/tamago/dma"
	"github.com/karlo195/tamago/internal/rng"
	"github.com/karlo195/tamago/soc/nxp/caam"
	"github.com/karlo195/tamago/soc/nxp/rngb"
)

//go:linkname initRNG runtime.initRNG
func initRNG() {
	_, fam, revMajor, revMinor := SiliconVersion()
	Family = fam

	if revMajor != 0 || revMinor != 0 {
		Native = true
	}

	if !Native {
		drbg := &rng.DRBG{}
		binary.LittleEndian.PutUint64(drbg.Seed[:], uint64(time.Now().UnixNano()))
		rng.GetRandomDataFn = drbg.GetRandomData
		return
	}

	switch Family {
	case IMX6UL:
		// Cryptographic Acceleration and Assurance Module
		CAAM = &caam.CAAM{
			Base:            CAAM_BASE,
			CCGR:            CCM_CCGR0,
			CG:              CCGRx_CG5,
			DeriveKeyMemory: dma.Default(),
		}
		CAAM.Init()

		// The CAAM TRNG is too slow for direct use, therefore
		// we use it to seed an AES-CTR based DRBG.
		drbg := &rng.DRBG{}
		CAAM.GetRandomData(drbg.Seed[:])

		rng.GetRandomDataFn = drbg.GetRandomData
	case IMX6ULL:
		// True Random Number Generator
		RNGB = &rngb.RNGB{
			Base: RNGB_BASE,
		}
		RNGB.Init()

		rng.GetRandomDataFn = RNGB.GetRandomData
	}
}
