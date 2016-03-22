package tracer

import "math"

// The BlockScheduler interface is implemented by all block scheduling algorithms.
type BlockScheduler interface {
	// Split frame into blocks of variable height and assign to the pool
	// of tracers using feedback collected from previous frames.
	//
	// This function returns the block height assignment for each tracer
	// in the input list.
	Schedule(tracers []Tracer, frameH uint32, lastFrameTime int64) []uint32
}

// The perfect scheduler assumes that the volume of tracing work between two
// subsequent frames is approximately the same.
type perfectScheduler struct {
	blockAssignment []uint32
}

// Create a new perfect scheduler instance
func NewPerfectScheduler() BlockScheduler {
	return &perfectScheduler{}
}

// Split frame into blocks of variable height and assign to the pool
// of tracers using feedback collected from previous frames.
//
// This function returns the block height assignment for each tracer in the
// input list. When previous frame information is available the scheduler
// uses the following formula for estimating the workload for tracer w and frame i+1:
// w_i, f_i+1 = (blockH,w_i / time,w_i) / Σ(blockH_i-1 / time,i-1)
func (sch *perfectScheduler) Schedule(tracers []Tracer, frameH uint32, lastFrameTime int64) []uint32 {
	var total float64 = 0.0
	var scaler float64

	// If this is the first time we try to schedule or the number of tracers
	// has changed we need to reset the block assignments
	if len(sch.blockAssignment) != len(tracers) {
		sch.blockAssignment = make([]uint32, len(tracers))

		// Get speed estimate for each tracer and distribute rows accordingly
		for _, tr := range tracers {
			total += float64(tr.SpeedEstimate())
		}
		scaler := float64(frameH) / total

		for idx, tr := range tracers {
			sch.blockAssignment[idx] = uint32(math.Max(1.0, math.Floor(float64(tr.SpeedEstimate())*scaler)))
		}

		return sch.blockAssignment
	}

	// Use last frame statistics
	var stats *Stats
	for _, tr := range tracers {
		stats = tr.Stats()
		total += float64(stats.BlockH) / float64(stats.BlockTime)
	}

	scaler = float64(frameH) / total
	var scheduledRows uint32 = 0
	for idx, tr := range tracers {
		stats = tr.Stats()
		sch.blockAssignment[idx] = uint32(math.Max(1.0, math.Floor(float64(stats.BlockH)/float64(stats.BlockTime)*scaler)))
		scheduledRows += sch.blockAssignment[idx]
	}

	// In case rows don't add up to the frame height append the missing ones to the first tracer
	sch.blockAssignment[0] += frameH - scheduledRows

	return sch.blockAssignment
}
