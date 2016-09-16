package renderer

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/achilleasa/go-pathtrace/asset/scene"
	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
)

type defaultRenderer struct {
	logger log.Logger

	options Options

	// Worker sync primitives
	workerInitGroup  sync.WaitGroup
	workerCloseGroup sync.WaitGroup

	// The list of registered tracers.
	tracers         []tracer.Tracer
	jobChans        []chan tracer.BlockRequest
	jobCompleteChan chan error

	// The selected primary tracer.
	primary int

	// The scheduler for distributing blocks to the list of tracers.
	scheduler tracer.BlockScheduler

	// The block assignments generated by the scheduler
	blockAssignments []uint32

	// Renderer statistics.
	stats FrameStats
}

// Create a new default renderer using the specified block scheduler and tracing pipeline.
func NewDefault(sc *scene.Scene, scheduler tracer.BlockScheduler, pipeline *opencl.Pipeline, opts Options) (Renderer, error) {
	if sc == nil {
		return nil, ErrSceneNotDefined
	} else if sc.Camera == nil {
		return nil, ErrCameraNotDefined
	}

	r := &defaultRenderer{
		logger:    log.New("renderer"),
		scheduler: scheduler,
		options:   opts,
	}

	err := r.initTracers(pipeline)
	if err != nil {
		return nil, err
	}
	r.jobChans = make([]chan tracer.BlockRequest, len(r.tracers))
	r.jobCompleteChan = make(chan error, 0)

	// Start workers
	r.workerInitGroup.Add(len(r.tracers))
	r.workerCloseGroup.Add(len(r.tracers))
	for trIndex := 0; trIndex < len(r.tracers); trIndex++ {
		// Queue state changes
		r.tracers[trIndex].UpdateState(tracer.Synchronous, tracer.FrameDimensions, [2]uint32{opts.FrameW, opts.FrameH})
		r.tracers[trIndex].UpdateState(tracer.Synchronous, tracer.SceneData, sc)
		r.tracers[trIndex].UpdateState(tracer.Synchronous, tracer.CameraData, sc.Camera)

		// Start worker
		r.jobChans[trIndex] = make(chan tracer.BlockRequest, 0)
		go r.jobWorker(trIndex)
	}

	// wait for all workers to start
	r.workerInitGroup.Wait()

	return r, nil
}

// Get last frame stats.
func (r *defaultRenderer) Stats() FrameStats {
	return r.stats
}

// Shutdown renderer and any attached tracers.
func (r *defaultRenderer) Close() {
	for _, ch := range r.jobChans {
		close(ch)
	}

	r.workerCloseGroup.Wait()
}

// Render next frame.
func (r *defaultRenderer) Render() error {
	return r.renderFrame(0)
}

// The actual frame implementation. This is intentionally split so it can be
// used by the opengl renderer.
func (r *defaultRenderer) renderFrame(accumulatedSamples uint32) error {
	var blockReq = tracer.BlockRequest{
		FrameW:             r.options.FrameW,
		FrameH:             r.options.FrameH,
		BlockW:             r.options.FrameW,
		SamplesPerPixel:    r.options.SamplesPerPixel,
		Exposure:           r.options.Exposure,
		NumBounces:         r.options.NumBounces,
		MinBouncesForRR:    r.options.MinBouncesForRR,
		AccumulatedSamples: accumulatedSamples,
		Seed:               rand.Uint32(),
	}

	// If running in progressive mode we need to capture a single sample
	if blockReq.SamplesPerPixel == 0 {
		blockReq.SamplesPerPixel = 1
	}

	start := time.Now()

	// Schedule blocks and process them in parallel
	r.blockAssignments = r.scheduler.Schedule(r.tracers, blockReq.FrameH)
	for trIndex, blockH := range r.blockAssignments {
		blockReq.BlockH = blockH
		r.jobChans[trIndex] <- blockReq

		r.stats.Tracers[trIndex].BlockH = blockH
		r.stats.Tracers[trIndex].FramePercent = 100.0 * float32(blockH) / float32(blockReq.FrameH)

		blockReq.BlockY += blockH
	}

	var tot uint32 = 0
	for _, bh := range r.blockAssignments {
		tot += bh
	}
	if tot != r.options.FrameH {
		fmt.Printf("S(assigned blocks) = %d != %d\n", tot, r.options.FrameH)
	}

	// Wait for all tracers to finish
	pending := len(r.tracers)
	for pending != 0 {
		err, ok := <-r.jobCompleteChan
		if !ok {
			err = ErrInterrupted
		}

		if err != nil {
			return err
		}

		pending--
	}

	// Run post-process filters on the primary tracer
	blockReq.BlockY = 0
	blockReq.BlockH = blockReq.FrameH
	r.tracers[r.primary].SyncFramebuffer(&blockReq)

	r.stats.RenderTime = time.Since(start)

	// Collect stats
	for trIndex, tr := range r.tracers {
		r.stats.Tracers[trIndex].RenderTime = tr.Stats().RenderTime
	}

	return nil
}

// A tracing job processor.
func (r *defaultRenderer) jobWorker(trIndex int) {
	r.workerInitGroup.Done()
	defer func() {
		r.tracers[trIndex].Close()
		r.workerCloseGroup.Done()
	}()

	for {
		select {
		case blockReq, ok := <-r.jobChans[trIndex]:
			if !ok {
				return
			}

			_, err := r.tracers[trIndex].Trace(&blockReq)
			if err == nil {
				// Merge trace accumulator output for this pass with primary tracer's frame accumulator
				_, err = r.tracers[r.primary].MergeOutput(r.tracers[trIndex], &blockReq)
			}
			r.jobCompleteChan <- err
		}
	}
}

// Select and initialize opencl devices excluding the ones which match the blacklist entries.
func (r *defaultRenderer) initTracers(pipeline *opencl.Pipeline) error {
	if len(r.options.BlackListedDevices) != 0 {
		r.logger.Infof("blacklisted devices: %s", strings.Join(r.options.BlackListedDevices, ", "))
	}

	platforms, err := device.GetPlatformInfo()
	if err != nil {
		return err
	}

	selectedDevices := make([]*device.Device, 0)
	for _, platformInfo := range platforms {
		for _, device := range platformInfo.Devices {
			keep := true
			for _, text := range r.options.BlackListedDevices {
				if text != "" && strings.Contains(device.Name, text) {
					keep = false
					break
				}
			}

			if keep {
				selectedDevices = append(selectedDevices, device)
			}
		}
	}

	// Create shared context for seleected devices
	sharedCtx, err := device.NewSharedContext(selectedDevices)
	if err != nil {
		return err
	}

	// Initialize all tracers using the shared context
	r.tracers = make([]tracer.Tracer, 0)
	r.stats.Tracers = make([]TracerStat, 0)
	r.primary = -1

	for _, device := range selectedDevices {
		// Create and initialize tracer
		tr, err := opencl.NewTracer(
			fmt.Sprintf("%s (%d)", device.Name, len(r.tracers)),
			device,
			sharedCtx,
			pipeline,
		)
		if err == nil {
			err = tr.Init()
		}

		if err != nil {
			r.logger.Warningf("could not init device %q: %v", device.Name, err)
			continue
		}

		// If no error occured add to list
		r.logger.Noticef("using device %q", tr.Id())
		r.tracers = append(r.tracers, tr)

		if r.options.ForcePrimaryDevice != "" && strings.Contains(device.Name, r.options.ForcePrimaryDevice) {
			r.primary = len(r.tracers) - 1
		}

		// Init statistics
		r.stats.Tracers = append(r.stats.Tracers, TracerStat{
			Id: tr.Id(),
		})
	}

	if len(r.tracers) == 0 {
		return ErrNoTracers
	}

	// If no primary tracer selected, pick the GPU with max estimated speed
	if r.primary == -1 {
		var bestSpeed uint32 = 0
		for trIndex, tr := range r.tracers {
			if ((tr.Flags() & tracer.CpuDevice) == 0) && tr.Speed() > bestSpeed {
				bestSpeed = tr.Speed()
				r.primary = trIndex
			}
		}
	}

	// If we still haven't found a primary device just select the first available
	if r.primary == -1 {
		r.primary = 0
	}

	r.stats.Tracers[r.primary].IsPrimary = true
	r.logger.Noticef("selected %q as primary device", r.tracers[r.primary].Id())

	return nil
}