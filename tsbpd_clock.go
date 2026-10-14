package srt

import "github.com/datarhei/gosrt/packet"

// Clock handling for TSBPD, following libsrt (srtcore/tsbpd_time.cpp).

const (
	driftWindow  = 1000
	driftMaxStep = 5_000 // microseconds
	wrapGuard    = 30_000_000
)

// tsbpdStartCorrection is how far the peer's clock origin lies after our own start, in microseconds.
// arrivedUs is our elapsed time when the peer's handshake arrived, peerTimestampUs the timestamp it carried.
func tsbpdStartCorrection(arrivedUs int64, peerTimestampUs uint32) int64 {
	return arrivedUs - int64(peerTimestampUs)
}

// tsbpdReleaseTime adds the signed corrections to an unsigned release time, never going below zero.
func tsbpdReleaseTime(base, delay uint64, shift, drift int64) uint64 {
	v := int64(base+delay) + shift + drift
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// unwrapTimestamp places a 32-bit timestamp on the connection's 64-bit timeline.
func unwrapTimestamp(ts uint32, baseOffset uint64, wrapPeriod bool) uint64 {
	if wrapPeriod && ts < wrapGuard {
		baseOffset += uint64(packet.MAX_TIMESTAMP) + 1
	}
	return baseOffset + uint64(ts)
}

type driftTracer struct {
	sum      int64
	n        int64
	firstRTT int64
	drift    int64 // microseconds, applied on top of shift
	shift    int64 // microseconds, the base moved in steps of driftMaxStep
}

func newDriftTracer() *driftTracer {
	return &driftTracer{firstRTT: -1}
}

// add takes one sample, measured against the base moved by shift, and the RTT sample of the same exchange.
// It reports whether a full window was averaged and the correction updated.
func (d *driftTracer) add(sampleUs, rttUs int64) bool {
	if d.firstRTT < 0 {
		d.firstRTT = rttUs
	}
	d.sum += sampleUs - (rttUs-d.firstRTT)/2
	d.n++
	if d.n < driftWindow {
		return false
	}
	avg := d.sum / d.n
	d.sum, d.n = 0, 0
	var over int64
	if avg > driftMaxStep {
		over = driftMaxStep
	} else if avg < -driftMaxStep {
		over = -driftMaxStep
	}
	d.shift += over
	d.drift = avg - over
	return true
}

func (d *driftTracer) total() int64 {
	return d.shift + d.drift
}
