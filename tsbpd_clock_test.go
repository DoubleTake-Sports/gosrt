package srt

import "testing"

func TestStartCorrectionAnchorsToThePeersHandshakeTime(t *testing.T) {
	// Dial began 300 ms before the response; the peer had been running for 2 ms when it answered.
	if got := tsbpdStartCorrection(300_000, 2_000); got != 298_000 {
		t.Fatalf("got %d", got)
	}
	// A peer clock that started before our dial gives a negative correction.
	if got := tsbpdStartCorrection(5_000, 40_000); got != -35_000 {
		t.Fatalf("got %d", got)
	}
}

func TestReleaseTimeNeverGoesBelowZero(t *testing.T) {
	if got := tsbpdReleaseTime(1_000, 10_000, -50_000, 0); got != 0 {
		t.Fatalf("got %d", got)
	}
	if got := tsbpdReleaseTime(1_000, 10_000, 2_000, -500); got != 12_500 {
		t.Fatalf("got %d", got)
	}
}

func TestDriftIsOnlyAppliedOnceAFullWindowIsAveraged(t *testing.T) {
	d := newDriftTracer()
	for i := 0; i < driftWindow-1; i++ {
		if d.add(3_000, 1_000) {
			t.Fatal("updated before the window was full")
		}
	}
	if !d.add(3_000, 1_000) {
		t.Fatal("did not update on a full window")
	}
	if d.drift != 3_000 || d.shift != 0 {
		t.Fatalf("drift %d shift %d", d.drift, d.shift)
	}
}

func TestDriftBeyondTheLimitMovesTheBaseAndKeepsTheRemainder(t *testing.T) {
	d := newDriftTracer()
	for i := 0; i < driftWindow; i++ {
		d.add(-7_000, 1_000)
	}
	if d.shift != -driftMaxStep || d.drift != -2_000 {
		t.Fatalf("drift %d shift %d", d.drift, d.shift)
	}
	if d.total() != -7_000 {
		t.Fatalf("total %d", d.total())
	}
}

func TestDriftSamplesAreMeasuredAgainstTheShiftedBase(t *testing.T) {
	d := newDriftTracer()
	for i := 0; i < driftWindow; i++ {
		d.add(-7_000, 1_000)
	}
	// The peer has not drifted any further: the samples now read only the remainder kept in drift.
	for i := 0; i < driftWindow; i++ {
		d.add(-7_000-d.shift, 1_000)
	}
	if d.total() != -7_000 {
		t.Fatalf("total %d", d.total())
	}
}

func TestAChangeInRoundTripTimeIsNotTakenAsDrift(t *testing.T) {
	d := newDriftTracer()
	d.add(0, 1_000)
	// RTT grew by 4 ms: the one-way delay grew by about 2 ms, which is not clock drift.
	for i := 1; i < driftWindow; i++ {
		d.add(2_000, 5_000)
	}
	if d.drift > 10 || d.drift < -10 {
		t.Fatalf("drift %d", d.drift)
	}
}

func TestUnwrappedTimestampFollowsTheDataPathsWrapState(t *testing.T) {
	const span = uint64(1) << 32
	if got := unwrapTimestamp(5, 0, false); got != 5 {
		t.Fatalf("got %d", got)
	}
	if got := unwrapTimestamp(10, 0, true); got != span+10 {
		t.Fatalf("got %d", got)
	}
	if got := unwrapTimestamp(4_000_000_000, 0, true); got != 4_000_000_000 {
		t.Fatalf("got %d", got)
	}
}
