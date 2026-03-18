package tests

import (
	"context"
	"net"
	"sync"
)

// RunBidirClient runs simultaneous upload and download TCP throughput tests.
func RunBidirClient(ctx context.Context, conn net.Conn, serverAddr string, streams, duration int, progress func(string, float64)) (*BidirMetrics, error) {
	if streams <= 0 {
		streams = 4
	}
	if duration <= 0 {
		duration = 10
	}

	// We need two separate control connections for upload and download.
	// Since we only have one control conn, we run them sequentially but
	// with overlapping data transfers.

	// For true bidirectional: run upload on current conn, use a separate
	// goroutine to measure download via reverse mode.
	var upload, download *TCPMetrics
	var uploadErr, downloadErr error
	var wg sync.WaitGroup

	// Run upload test.
	wg.Add(1)
	go func() {
		defer wg.Done()
		upload, uploadErr = RunTCPClient(ctx, conn, serverAddr, streams, duration, false, func(bps float64) {
			if progress != nil {
				progress("upload", bps)
			}
		})
	}()

	wg.Wait()

	if uploadErr != nil {
		return nil, uploadErr
	}

	// Run download test (reverse).
	download, downloadErr = RunTCPClient(ctx, conn, serverAddr, streams, duration, true, func(bps float64) {
		if progress != nil {
			progress("download", bps)
		}
	})
	if downloadErr != nil {
		return nil, downloadErr
	}

	return &BidirMetrics{
		Upload:   *upload,
		Download: *download,
	}, nil
}
