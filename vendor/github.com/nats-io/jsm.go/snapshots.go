// Copyright 2020 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nats-io/jsm.go/api"
)

type snapshotOptions struct {
	file      string
	scb       func(SnapshotProgress)
	rcb       func(RestoreProgress)
	debug     bool
	consumers bool
	jsck      bool
	chunkSz   int
}

type SnapshotOption func(o *snapshotOptions)

// SnapshotConsumers includes consumer configuration and state in backups
func SnapshotConsumers() SnapshotOption {
	return func(o *snapshotOptions) {
		o.consumers = true
	}
}

// SnapshotHealthCheck performs a health check prior to starting the snapshot
func SnapshotHealthCheck() SnapshotOption {
	return func(o *snapshotOptions) {
		o.jsck = true
	}
}

// SnapshotNotify notifies cb about progress of the snapshot operation
func SnapshotNotify(cb func(SnapshotProgress)) SnapshotOption {
	return func(o *snapshotOptions) {
		o.scb = cb
	}
}

// RestoreNotify notifies cb about progress of the restore operation
func RestoreNotify(cb func(RestoreProgress)) SnapshotOption {
	return func(o *snapshotOptions) {
		o.rcb = cb
	}
}

// SnapshotDebug enables logging using the standard go logging library
func SnapshotDebug() SnapshotOption {
	return func(o *snapshotOptions) {
		o.debug = true
	}
}

type SnapshotProgress interface {
	// StartTime is when the process started
	StartTime() time.Time
	// EndTime is when the process ended - zero when not completed
	EndTime() time.Time
	// ChunkSize is the size of the data packets sent over NATS
	ChunkSize() int
	// ChunksReceived is how many chunks of ChunkSize were received
	ChunksReceived() uint32
	// BytesReceived is how many Bytes have been received
	BytesReceived() uint64
	// BlocksReceived is how many data storage Blocks have been received
	BlocksReceived() int
	// BlockSize is the size in bytes of each data storage Block
	BlockSize() int
	// BlocksExpected is the number of blocks expected to be received
	BlocksExpected() int
	// BlockBytesReceived is the size of uncompressed block data received
	BlockBytesReceived() uint64
	// HasMetadata indicates if the metadata blocks have been received yet
	HasMetadata() bool
	// HasData indicates if all the data blocks have been received
	HasData() bool
	// BytesPerSecond is the number of bytes received in the last second, 0 during the first second
	BytesPerSecond() uint64
	// HealthCheck indicates if health checking was requested
	HealthCheck() bool
}

type RestoreProgress interface {
	// StartTime is when the process started
	StartTime() time.Time
	// EndTime is when the process ended - zero when not completed
	EndTime() time.Time
	// ChunkSize is the size of the data packets sent over NATS
	ChunkSize() int
	// ChunksSent is the number of chunks of size ChunkSize that was sent
	ChunksSent() uint32
	// ChunksToSend number of chunks of ChunkSize expected to be sent
	ChunksToSend() int
	// BytesSent is the number of bytes sent so far
	BytesSent() uint64
	// BytesPerSecond is the number of bytes received in the last second, 0 during the first second
	BytesPerSecond() uint64
}

type snapshotProgress struct {
	startTime          time.Time
	endTime            time.Time
	healthCheck        bool
	chunkSize          int
	chunksReceived     uint32
	chunksSent         uint32
	chunksToSend       int
	bytesReceived      uint64
	bytesSent          uint64
	blocksReceived     int32
	blockSize          int
	blocksExpected     int
	blockBytesReceived uint64
	metadataDone       bool
	dataDone           bool
	sending            bool   // if we are sending data, this is a hint for bps calc
	bps                uint64 // Bytes per second
	scb                func(SnapshotProgress)
	rcb                func(RestoreProgress)

	sync.Mutex
}

func (sp *snapshotProgress) HealthCheck() bool {
	return sp.healthCheck
}

func (sp *snapshotProgress) BlockBytesReceived() uint64 {
	return sp.blockBytesReceived
}

func (sp *snapshotProgress) ChunksReceived() uint32 {
	return sp.chunksReceived
}

func (sp *snapshotProgress) BytesReceived() uint64 {
	return sp.bytesReceived
}

func (sp *snapshotProgress) BlocksReceived() int {
	return int(sp.blocksReceived)
}

func (sp *snapshotProgress) BlockSize() int {
	return sp.blockSize
}

func (sp *snapshotProgress) BlocksExpected() int {
	return sp.blocksExpected
}

func (sp *snapshotProgress) HasMetadata() bool {
	return sp.metadataDone
}

func (sp *snapshotProgress) HasData() bool {
	return sp.dataDone
}

func (sp *snapshotProgress) BytesPerSecond() uint64 {
	if sp.bps > 0 {
		return sp.bps
	}

	if sp.sending {
		return sp.bytesSent
	}

	return sp.bytesReceived
}

func (sp *snapshotProgress) StartTime() time.Time {
	return sp.startTime
}

func (sp *snapshotProgress) EndTime() time.Time {
	return sp.endTime
}

func (sp *snapshotProgress) ChunkSize() int {
	return sp.chunkSize
}

func (sp *snapshotProgress) ChunksToSend() int {
	return sp.chunksToSend
}

func (sp *snapshotProgress) ChunksSent() uint32 {
	return sp.chunksSent
}

func (sp *snapshotProgress) BytesSent() uint64 {
	return sp.bytesSent
}

func (sp *snapshotProgress) notify() {
	if sp.scb != nil {
		sp.scb(sp)
	}
	if sp.rcb != nil {
		sp.rcb(sp)
	}
}

// the tracker will gunzip and untar the stream as it passes by looking
// for the file names in the tar data and based on these will notify the
// caller about blocks received etc
func (sp *snapshotProgress) trackBlockProgress(r io.Reader, debug bool, errc chan error) {
	seenMetaSum := false
	seenMetaInf := false

	zr, err := gzip.NewReader(r)
	if err != nil {
		errc <- fmt.Errorf("progress tracker failed to start gzip: %s", err)
		return
	}
	defer zr.Close()

	tr := tar.NewReader(zr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			errc <- fmt.Errorf("progress tracker received EOF")
			return
		}
		if err != nil {
			errc <- fmt.Errorf("progress tracker received an unexpected error: %s", err)
			return
		}

		if debug {
			log.Printf("Received file %s", hdr.Name)
		}

		if !sp.metadataDone {
			if hdr.Name == "meta.sum" {
				seenMetaSum = true
			}
			if hdr.Name == "meta.inf" {
				seenMetaInf = true
			}
			if seenMetaInf && seenMetaSum {
				sp.metadataDone = true

				// tell the caller soon as we are done with metadata
				sp.notify()
			}
		}

		if strings.HasSuffix(hdr.Name, "blk") {
			sp.Lock()
			br := sp.blocksReceived
			sp.blocksReceived++
			sp.blockBytesReceived += uint64(hdr.Size)
			sp.Unlock()

			// notify before setting done so callers can easily print the
			// last progress for blocks
			sp.notify()

			if int(br) == sp.blocksExpected {
				sp.dataDone = true
			}
		}
	}
}

func (sp *snapshotProgress) trackBps(ctx context.Context) {
	var lastBytes uint64 = 0

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			sp.Lock()
			if sp.sending {
				sent := sp.bytesSent
				sp.bps = sent - lastBytes
				lastBytes = sent
			} else {
				received := sp.bytesReceived
				sp.bps = received - lastBytes
				lastBytes = received
			}
			sp.Unlock()

			sp.notify()

		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) RestoreSnapshotFromFile(ctx context.Context, stream string, file string, opts ...SnapshotOption) (RestoreProgress, *api.StreamState, error) {
	sopts := &snapshotOptions{
		file:    file,
		chunkSz: 512 * 1024,
	}

	for _, opt := range opts {
		opt(sopts)
	}

	fstat, err := os.Stat(file)
	if err != nil {
		return nil, nil, err
	}

	inf, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer inf.Close()

	var resp api.JSApiStreamRestoreResponse
	err = m.jsonRequest(fmt.Sprintf(api.JSApiStreamRestoreT, stream), map[string]string{}, &resp)
	if err != nil {
		return nil, nil, err
	}

	progress := &snapshotProgress{
		startTime:    time.Now(),
		chunkSize:    sopts.chunkSz,
		chunksToSend: 1 + int(fstat.Size())/sopts.chunkSz,
		sending:      true,
		rcb:          sopts.rcb,
		scb:          sopts.scb,
	}
	defer func() { progress.endTime = time.Now() }()
	go progress.trackBps(ctx)

	if sopts.debug {
		log.Printf("Starting restore of %q from %s using %d chunks", stream, file, progress.chunksToSend)
	}

	// in debug notify ~20ish times
	notifyInterval := uint32(1)
	if progress.chunksToSend >= 20 {
		notifyInterval = uint32(math.Ceil(float64(progress.chunksToSend) / 20))
	}

	// send initial notify to inform what to expect
	progress.notify()

	nc := m.nc
	var chunk [512 * 1024]byte
	var cresp *nats.Msg

	for {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		n, err := inf.Read(chunk[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		cresp, err = nc.Request(resp.DeliverSubject, chunk[:n], m.timeout)
		if err != nil {
			return nil, nil, err
		}
		if IsErrorResponse(cresp) {
			return nil, nil, fmt.Errorf("restore failed: %q", cresp.Data)
		}

		if sopts.debug && progress.chunksSent > 0 && progress.chunksSent%notifyInterval == 0 {
			log.Printf("Sent %d chunks", progress.chunksSent)
		}

		progress.Lock()
		progress.chunksSent++
		progress.bytesSent += uint64(n)
		progress.Unlock()

		progress.notify()
	}

	if sopts.debug {
		log.Printf("Sent %d chunks, server will now restore the snapshot, this might take a long time", progress.chunksSent)
	}

	// very long timeout as the server is doing the restore here and might take any mount of time
	cresp, err = nc.Request(resp.DeliverSubject, nil, time.Hour)
	if err != nil {
		return nil, nil, err
	}
	if IsErrorResponse(cresp) {
		return nil, nil, fmt.Errorf("restore failed: %q", cresp.Data)
	}

	kind, finalr, err := api.ParseMessage(cresp.Data)
	if err != nil {
		return nil, nil, err
	}

	if kind != "io.nats.jetstream.api.v1.stream_create_response" {
		return nil, nil, fmt.Errorf("invalid final response, expected a io.nats.jetstream.api.v1.stream_create_response message but got %q", kind)
	}

	createResp, ok := finalr.(*api.JSApiStreamCreateResponse)
	if !ok {
		return nil, nil, fmt.Errorf("invalid final response type")
	}
	if createResp.IsError() {
		return nil, nil, createResp.ToError()
	}

	return progress, &createResp.State, nil
}

// SnapshotToFile creates a backup into gzipped tar file
func (s *Stream) SnapshotToFile(ctx context.Context, file string, opts ...SnapshotOption) (SnapshotProgress, error) {
	sopts := &snapshotOptions{
		file:      file,
		jsck:      false,
		consumers: false,
		chunkSz:   512 * 1024,
	}

	for _, opt := range opts {
		opt(sopts)
	}

	if sopts.debug {
		log.Printf("Starting backup of %q to %q", s.Name(), file)
	}

	of, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	defer of.Close()

	ib := nats.NewInbox()
	req := api.JSApiStreamSnapshotRequest{
		DeliverSubject: ib,
		NoConsumers:    !sopts.consumers,
		CheckMsgs:      sopts.jsck,
		ChunkSize:      sopts.chunkSz,
	}

	var resp api.JSApiStreamSnapshotResponse
	err = s.mgr.jsonRequest(fmt.Sprintf(api.JSApiStreamSnapshotT, s.Name()), req, &resp)
	if err != nil {
		return nil, err
	}

	errc := make(chan error)
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	progress := &snapshotProgress{
		startTime:      time.Now(),
		chunkSize:      req.ChunkSize,
		blockSize:      resp.BlkSize,
		blocksExpected: resp.NumBlks,
		scb:            sopts.scb,
		rcb:            sopts.rcb,
		healthCheck:    sopts.jsck,
	}
	defer func() { progress.endTime = time.Now() }()
	go progress.trackBps(sctx)

	// set up a multi writer that writes to file and the progress monitor
	// if required else we write directly to the file and be done with it
	trackingR, trackingW := net.Pipe()
	defer trackingR.Close()
	defer trackingW.Close()
	go progress.trackBlockProgress(trackingR, sopts.debug, errc)

	writer := io.MultiWriter(of, trackingW)

	// tell the caller we are starting and what to expect
	progress.notify()

	sub, err := s.mgr.nc.Subscribe(ib, func(m *nats.Msg) {
		if len(m.Data) == 0 {
			m.Sub.Unsubscribe()
			cancel()
			return
		}

		progress.Lock()
		progress.bytesReceived += uint64(len(m.Data))
		progress.chunksReceived++
		progress.Unlock()

		n, err := writer.Write(m.Data)
		if err != nil {
			errc <- err
			return
		}
		if n != len(m.Data) {
			errc <- fmt.Errorf("failed to write %d bytes to %s, only wrote %d", len(m.Data), file, n)
			return
		}
	})
	if err != nil {
		return progress, err
	}
	defer sub.Unsubscribe()

	select {
	case err := <-errc:
		if sopts.debug {
			log.Printf("Snapshot Error: %s", err)
		}

		return progress, err
	case <-sctx.Done():
		return progress, nil
	}
}
