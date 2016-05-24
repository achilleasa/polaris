package tracer

import (
	"testing"
	"time"
)

func TestNaiveScheduler(t *testing.T) {
	type spec struct {
		speed1   uint32
		speed2   uint32
		frameH   uint32
		expRows1 uint32
		expRows2 uint32
	}
	specs := []spec{
		spec{1, 2, 10, 4, 6},
		spec{2, 1, 10, 7, 3},
		spec{1, 1000, 10, 1, 9},
	}

	for index, s := range specs {
		tr1 := makeMockTracer("mock-1", s.speed1)
		tr2 := makeMockTracer("mock-2", s.speed2)
		tracers := []Tracer{tr1, tr2}

		sch := NaiveScheduler()
		blockAssignment := sch.Schedule(tracers, s.frameH)

		if blockAssignment[0] != s.expRows1 {
			t.Fatalf("[spec %d] expected tracer 0 to be assigned %d rows; got %d", index, s.expRows1, blockAssignment[0])
		}

		if blockAssignment[1] != s.expRows2 {
			t.Fatalf("[spec %d] expected tracer 1 to be assigned %d rows; got %d", index, s.expRows2, blockAssignment[1])
		}
	}
}

func TestPerfectScheduler(t *testing.T) {
	type spec struct {
		frameH   uint32
		rTime1   time.Duration
		rTime2   time.Duration
		expRows1 uint32
		expRows2 uint32
	}
	specs := []spec{
		// First call always behaves like the naive scheduler
		spec{10, time.Duration(1), time.Duration(5), 5, 5},
		// Second call should use the render times to assign rows
		spec{10, time.Duration(1), time.Duration(5), 9, 1},
		// This time tracer 2 performed much better
		spec{10, time.Duration(5), time.Duration(1), 7, 3},
	}

	// Tracers have same speed
	tr1 := makeMockTracer("mock-1", 1)
	tr2 := makeMockTracer("mock-2", 1)
	tracers := []Tracer{tr1, tr2}

	sch := PerfectScheduler()
	for index, s := range specs {
		tr1.stats.RenderTime = s.rTime1
		tr2.stats.RenderTime = s.rTime2

		blockAssignment := sch.Schedule(tracers, s.frameH)

		if blockAssignment[0] != s.expRows1 {
			t.Fatalf("[spec %d] expected tracer 0 to be assigned %d rows; got %d", index, s.expRows1, blockAssignment[0])
		}

		if blockAssignment[1] != s.expRows2 {
			t.Fatalf("[spec %d] expected tracer 1 to be assigned %d rows; got %d", index, s.expRows2, blockAssignment[1])
		}

		tr1.stats.BlockH = blockAssignment[0]
		tr2.stats.BlockH = blockAssignment[1]
	}
}

type mockTracer struct {
	id    string
	speed uint32
	stats *Stats
}

func makeMockTracer(id string, speed uint32) *mockTracer {
	return &mockTracer{
		id:    id,
		speed: speed,
		stats: &Stats{},
	}
}

func (mt *mockTracer) Id() string {
	return mt.id
}

func (mt *mockTracer) Flags() Flag {
	return Local
}

func (mt *mockTracer) Speed() uint32 {
	return mt.speed
}

func (mt *mockTracer) Init() error {
	return nil
}

func (mt *mockTracer) Close() {
}

func (mt *mockTracer) Enqueue(_ BlockRequest) {
}

func (mt *mockTracer) Update(_ UpdateType, _ interface{}) {
}

func (mt *mockTracer) Stats() *Stats {
	return mt.stats
}
