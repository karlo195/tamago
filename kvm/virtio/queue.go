// VirtIO Virtual Queue support
// https://github.com/usbarmory/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package virtio

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/usbarmory/tamago/dma"
)

// Descriptor Flags
const (
	Next     = 1
	Write    = 2
	Indirect = 3
)

// Descriptor represents a VirtIO virtual queue descriptor.
//
// All exported fields are used one-time at initialization, fields requiring
// DMA are accessible through functions.
type Descriptor struct {
	Address uint64
	length  uint32
	Flags   uint16
	Next    uint16

	// DMA buffer (addressed data)
	buf []byte
}

// Bytes converts the descriptor structure to byte array format.
func (d *Descriptor) Bytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, d.Address)
	binary.Write(buf, binary.LittleEndian, d.length)
	binary.Write(buf, binary.LittleEndian, d.Flags)
	binary.Write(buf, binary.LittleEndian, d.Next)

	return buf.Bytes()
}

// Init initializes a virtual queue descriptor for the given reserved DMA
// buffer, which must have been prefiously created with dma.Reserve().
func (d *Descriptor) Init(buf []byte, flags uint16) {
	res, addr := dma.Reserved(buf)

	if !res {
		return
	}

	d.Address = uint64(addr)
	d.length = uint32(len(buf))
	d.Flags = flags

	d.buf = buf
}

// Destroy removes a virtual queue descriptor from physical memory.
func (d *Descriptor) Destroy() {
	dma.Release(uint(d.Address))
}

// Read copies the contents of the descriptor buffer to b.
func (d *Descriptor) Read(b []byte) {
	copy(b, d.buf)
}

// Write copies the contents of b to the descriptor buffer.
func (d *Descriptor) Write(b []byte) {
	d.length = uint32(len(b))
	copy(d.buf, b)
}

// Available represents a VirtIO virtual queue Available ring buffer.
//
// All exported fields are used one-time at initialization, fields requiring
// DMA are accessible through functions.
type Available struct {
	Flags      uint16
	index      uint16
	ring       []uint16
	EventIndex uint16

	// DMA buffer
	buf []byte
}

// Bytes converts the descriptor structure to byte array format.
func (d *Available) Bytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, d.Flags)
	binary.Write(buf, binary.LittleEndian, d.index)

	for _, ring := range d.ring {
		binary.Write(buf, binary.LittleEndian, ring)
	}

	binary.Write(buf, binary.LittleEndian, d.EventIndex)

	return buf.Bytes()
}

// SetIndex updates the descriptor index field.
func (d *Available) SetIndex(index uint16) {
	off := 2
	binary.LittleEndian.PutUint16(d.buf[off:], index)

	d.index = index
}

// Ring returns a ring buffer at the given position.
func (d *Available) Ring(n uint16) uint16 {
	off := 4 + n*2
	d.ring[n] = binary.LittleEndian.Uint16(d.buf[off:])

	return d.ring[n]
}

// SetRingIndex updates the index value of a ring buffer.
func (d *Available) SetRingIndex(n uint16, index uint16) {
	off := 4 + n*2
	binary.LittleEndian.PutUint16(d.buf[off:], index)

	d.ring[n] = index
}

// Ring represents a VirtIO virtual queue buffer index
type Ring struct {
	Index  uint32
	Length uint32
}

// Bytes converts the descriptor structure to byte array format.
func (d *Ring) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, d)
	return buf.Bytes()
}

// Used represents a VirtIO virtual queue Used ring buffer.
//
// All exported fields are used one-time at initialization, fields requiring
// DMA are accessible through functions.
type Used struct {
	Flags      uint16
	index      uint16
	ring       []*Ring
	AvailEvent uint16

	// DMA buffer
	buf []byte

	last uint16
}

// Bytes converts the descriptor structure to byte array format.
func (d *Used) Bytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, d.Flags)
	binary.Write(buf, binary.LittleEndian, d.index)

	for _, ring := range d.ring {
		buf.Write(ring.Bytes())
	}

	binary.Write(buf, binary.LittleEndian, d.AvailEvent)

	return buf.Bytes()
}

// Index returns the descriptor index field.
func (d *Used) Index() uint16 {
	off := 2
	d.index = binary.LittleEndian.Uint16(d.buf[off:])

	return d.index
}

// Ring returns a ring buffer at the given position.
func (d *Used) Ring(n uint16) Ring {
	off := 4 + n*8
	binary.Decode(d.buf[off:], binary.LittleEndian, d.ring[n])

	return *d.ring[n]
}

// VirtualQueue represents a VirtIO split virtual queue Descriptor
type VirtualQueue struct {
	sync.Mutex

	Descriptors []*Descriptor
	Available   Available
	Used        Used

	// DMA buffer
	buf    []byte
	desc   uint // physical address for QueueDesc
	driver uint // phusical address for QueueDriver
	device uint // physical address for QueueDevice

	size uint16
}

// Bytes converts the descriptor structure to byte array format, the device
// area and driver area location offsets are also returned.
func (d *VirtualQueue) Bytes() ([]byte, int, int) {
	buf := new(bytes.Buffer)

	for _, desc := range d.Descriptors {
		buf.Write(desc.Bytes())
	}

	driver := buf.Len()
	buf.Write(d.Available.Bytes())

	// Used ring requires 4 bytes alignment
	if r := buf.Len() % 4; r != 0 {
		buf.Write(make([]byte, 4-r))
	}

	device := buf.Len()
	buf.Write(d.Used.Bytes())

	return buf.Bytes(), driver, device
}

// Init initializes a split virtual queue for the given size.
func (d *VirtualQueue) Init(size int, length int, flags uint16) {
	d.Lock()
	defer d.Unlock()

	// To avoid excessive DMA region fragmentation a single allocation
	// reserves all descriptor buffers.
	_, buf := dma.Reserve(size*length, 0)

	for i := 0; i < size; i++ {
		off := size * i

		desc := &Descriptor{}
		desc.Init(buf[off:off+length], flags)

		ring := &Ring{}

		d.Descriptors = append(d.Descriptors, desc)
		d.Available.ring = append(d.Available.ring, uint16(i))
		d.Used.ring = append(d.Used.ring, ring)
	}

	if flags == Write {
		// make all buffers immediately available
		d.Available.index = uint16(size)
	}

	// allocate DMA buffer
	buf, driver, device := d.Bytes()
	d.desc, d.buf = dma.Reserve(len(buf), 16)
	copy(d.buf, buf)

	// calculate area pointers
	d.driver = d.desc + uint(driver)
	d.device = d.desc + uint(device)
	d.size = uint16(size)

	// assign DMA slices
	d.Available.buf = d.buf[driver:device]
	d.Used.buf = d.buf[device:]
}

// Destroy removes a split virtual queue from physical memory.
func (d *VirtualQueue) Destroy() {
	for _, d := range d.Descriptors {
		d.Destroy()
	}

	d.Available.buf = nil
	d.Used.buf = nil

	dma.Release(d.desc)
}

// Address returns the virtual queue physical address.
func (d *VirtualQueue) Address() (desc uint, driver uint, device uint) {
	return d.desc, d.driver, d.device
}

// Pop receives a single used buffer from the virtual queue,
func (d *VirtualQueue) Pop() (buf []byte) {
	d.Lock()
	defer d.Unlock()

	if d.Used.Index() == d.Used.last {
		return
	}

	avail := d.Used.Ring(d.Used.last % d.size)
	buf = make([]byte, avail.Length)

	d.Descriptors[avail.Index].Read(buf)

	d.Available.index += 1
	d.Available.SetRingIndex(d.Available.index%d.size, uint16(avail.Index))

	d.Available.SetIndex(d.Available.index)
	d.Used.last += 1

	return
}

// Push supplies a single available buffer to the virtual queue.
func (d *VirtualQueue) Push(buf []byte) {
	d.Lock()
	defer d.Unlock()

	index := d.Available.Ring(d.Available.index % d.size)
	used := d.Used.Index() - d.Used.last

	off := 8 + index*16
	binary.LittleEndian.PutUint32(d.buf[off:], uint32(len(buf)))

	d.Descriptors[index].Write(buf)
	d.Available.SetIndex(d.Available.index + 1)

	for i := used; i > 0; i-- {
		n := d.Available.index % d.size
		avail := d.Used.Ring(i - 1)

		d.Available.SetRingIndex(n, uint16(avail.Index))
	}

	d.Used.last += used
}
