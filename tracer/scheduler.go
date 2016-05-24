package tracer

import "math"

// The BlockScheduler interface is implemented by all block scheduling algorithms.
type BlockScheduler interface {
	// Split frame into row blocks of variable height and assign to the pool
	// of tracers using feedback collected from previous frames.
	Schedule(tracers []Tracer, frameH uint32) []uint32
}

// The naive scheduler distributes blocks to available renderers based on their
// reported speed estimate.
type naiveScheduler struct {
	blockAssignment []uint32
}

// Create a new naive scheduler
func NaiveScheduler() BlockScheduler {
	return &naiveScheduler{}
}

// Split frame into blocks and assign blocks bases on reported tracer speeds.
func (sch *naiveScheduler) Schedule(tracers []Tracer, frameH uint32) []uint32 {
	if len(sch.blockAssignment) != len(tracers) {
		sch.blockAssignment = assignBlocksBasedOnSpeed(tracers, frameH)
	}

	return sch.blockAssignment
}

// The perfect scheduler assumes that the volume of tracing work between two
// subsequent frames is approximately the same.
type perfectScheduler struct {
	blockAssignment []uint32
}

// Create a new perfect scheduler instance
func PerfectScheduler() BlockScheduler {
	return &perfectScheduler{}
}

// Split frame into blocks of variable height and assign to the pool
// of tracers using feedback collected from previous frames.
//
// This function returns the block height assignment for each tracer in the
// input list. When previous frame information is available the scheduler
// uses the following formula for estimating the workload for tracer w and frame i+1:
// w_i, f_i+1 = (blockH,w_i / time,w_i) / Σ(blockH_i-1 / time,i-1)
func (sch *perfectScheduler) Schedule(tracers []Tracer, frameH uint32) []uint32 {
	// Use a naive distribution for the first assignment
	if len(sch.blockAssignment) != len(tracers) {
		sch.blockAssignment = assignBlocksBasedOnSpeed(tracers, frameH)
		return sch.blockAssignment
	}

	// Use last frame statistics to build our scaler
	var stats *Stats
	var total, scaler float64
	for _, tr := range tracers {
		stats = tr.Stats()
		total += float64(stats.BlockH) / float64(stats.RenderTime.Nanoseconds())
	}
	scaler = float64(frameH) / total

	var assignedRows, blockH uint32 = 0, 0
	for idx, tr := range tracers {
		stats = tr.Stats()
		blockH = uint32(math.Max(1.0, math.Floor(float64(stats.BlockH)/float64(stats.RenderTime.Nanoseconds())*scaler)))
		assignedRows += blockH
		sch.blockAssignment[idx] = blockH
	}

	// If the assigned rows don't match our frame height add them to the first tracer
	if assignedRows < frameH {
		sch.blockAssignment[0] += frameH - assignedRows
	}

	return sch.blockAssignment
}

// Assign blocks to tracers based on reported speed.
func assignBlocksBasedOnSpeed(tracers []Tracer, frameH uint32) []uint32 {
	blockAssignment := make([]uint32, len(tracers))

	var speedSum uint32 = 0
	for _, tr := range tracers {
		speedSum += tr.Speed()
	}

	scaler := float64(frameH) / float64(speedSum)

	var assignedRows, blockH uint32 = 0, 0
	for idx, tr := range tracers {
		blockH = uint32(math.Max(1.0, float64(tr.Speed())*scaler))
		assignedRows += blockH
		blockAssignment[idx] = blockH
	}

	// If the assigned rows don't match our frame height add them to the first tracer
	if assignedRows < frameH {
		blockAssignment[0] += frameH - assignedRows
	}

	return blockAssignment
}
