package opencl

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/types"
)

func appendPipeline(stage pipelineStage) tracer.Stage {
	return func(t tracer.Tracer) error {
		tr, ok := t.(*clTracer)
		if !ok {
			return ErrInvalidOption
		}

		tr.pipeline = append(tr.pipeline, stage)
		return nil
	}
}

// Clear the accumulator buffer.
func ClearAccumulator() tracer.Stage {
	return appendPipeline(
		func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error) {
			return tr.resources.ClearAccumulator(blockReq)
		},
	)
}

// Use a perspective camera for the primary ray generation stage.
func PerspectiveCamera() tracer.Stage {
	return appendPipeline(
		func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error) {
			return tr.resources.GeneratePrimaryRays(blockReq, tr.sceneData.Camera.Position, tr.sceneData.Camera.Frustrum)
		},
	)
}

// Apply simple Reinhard tone-mapping.
func TonemapSimpleReinhard(exposure float32) tracer.Stage {
	return appendPipeline(
		func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error) {
			return tr.resources.TonemapSimpleReinhard(blockReq, exposure)
		},
	)
}

// Use a montecarlo pathtracer implementation.
func MonteCarloIntegrator(numBounces uint32) tracer.Stage {
	return appendPipeline(
		func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error) {
			var err error

			start := time.Now()
			numPixels := int(blockReq.FrameW * blockReq.BlockH)
			numEmissives := uint32(len(tr.sceneData.EmissivePrimitives))

			var activeRayBuf uint32 = 0

			// Intersect primary rays outside of the loop
			// TODO: Use packet query
			_, err = tr.resources.RayIntersectionQuery(activeRayBuf, numPixels)
			if err != nil {
				return time.Since(start), err
			}

			_, err = debugPrimaryRayIntersections(tr.resources, blockReq, "debug-00-primary-intersections.png")
			if err != nil {
				return time.Since(start), err
			}

			var bounce uint32
			for bounce = 0; bounce < numBounces; bounce++ {
				tr.logger.Noticef("[bounce %02d] intersected rays: %d", bounce, readCounter(tr.resources, activeRayBuf))

				// Shade hits
				_, err = tr.resources.ShadeHits(bounce, 0, numEmissives, activeRayBuf, numPixels)
				if err != nil {
					return time.Since(start), err
				}

				// Shade misses on first bounce
				if bounce == 0 {
					_, err = tr.resources.ShadeMisses(tr.sceneData.SceneDiffuseMatIndex, activeRayBuf, numPixels)
					if err != nil {
						return time.Since(start), err
					}
				}

				_, err = debugThroughput(tr.resources, blockReq, fmt.Sprintf("debug-01-throughput-bounce-%d.png", bounce))
				if err != nil {
					return time.Since(start), err
				}

				_, err = debugEmissiveSamples(tr.resources, blockReq, fmt.Sprintf("debug-01-emissive-samples-bounce-%d.png", bounce))
				if err != nil {
					return time.Since(start), err
				}

				tr.logger.Noticef("[bounce %02d] generated indirect rays: %d", bounce, readCounter(tr.resources, 1-activeRayBuf))
				tr.logger.Noticef("[bounce %02d] generated occlusion rays: %d", bounce, readCounter(tr.resources, 2))

				// Process intersections for occlusion rays and accumulate emissive samples for non occluded paths
				_, err = tr.resources.RayIntersectionTest(2, numPixels)
				if err != nil {
					return time.Since(start), err
				}
				_, err = tr.resources.AccumulateEmissiveSamples(2, numPixels)
				if err != nil {
					return time.Since(start), err
				}

				// Process intersections for indirect rays
				activeRayBuf = 1 - activeRayBuf
				_, err = tr.resources.RayIntersectionQuery(activeRayBuf, numPixels)
				if err != nil {
					return time.Since(start), err
				}
			}

			return time.Since(start), nil
		},
	)
}

type ray struct {
	origin types.Vec4
	dir    types.Vec4
}

type intersection struct {
	wuvt         types.Vec4
	meshInstance uint32
	triIndex     uint32
	_padding1    uint32
	_padding2    uint32
}

type rayPath struct {
	throughput types.Vec4
	status     uint32
	pixelIndex uint32
	reserved   [2]uint32
}

// Debug primary ray intersections.
func debugPrimaryRayIntersections(dr *deviceResources, blockReq *tracer.BlockRequest, imgFile string) (time.Duration, error) {
	start := time.Now()

	f, err := os.Create(imgFile)
	if err != nil {
		return time.Since(start), err
	}
	defer f.Close()

	im := image.NewRGBA(image.Rect(0, 0, int(blockReq.FrameW), int(blockReq.FrameH)))

	data, err := dr.buffers.Intersections.ReadDataIntoSlice(make([]intersection, 0))
	if err != nil {
		return time.Since(start), err
	}
	intersections := data.([]intersection)

	data, err = dr.buffers.Paths.ReadDataIntoSlice(make([]rayPath, 0))
	if err != nil {
		return time.Since(start), err
	}
	paths := data.([]rayPath)

	// Find max depth
	var maxDepth float32 = 0
	for _, hit := range intersections {
		if hit.wuvt[3] < math.MaxFloat32 && hit.wuvt[3] > maxDepth {
			maxDepth = hit.wuvt[3]
		}
	}
	maxDepth++

	// convert depth to 0-255 range
	for hitIndex, hit := range intersections {
		if hit.wuvt[3] == math.MaxFloat32 {
			continue
		}

		normDepth := uint8(255.0 * (1.0 - hit.wuvt[3]/maxDepth))
		offset := paths[hitIndex].pixelIndex << 2
		im.Pix[offset] = normDepth
		im.Pix[offset+1] = normDepth
		im.Pix[offset+2] = normDepth
		im.Pix[offset+3] = 255 // alpha
	}

	return time.Since(start), png.Encode(f, im)
}

// Debug path throughput.
func debugThroughput(dr *deviceResources, blockReq *tracer.BlockRequest, imgFile string) (time.Duration, error) {
	start := time.Now()

	f, err := os.Create(imgFile)
	if err != nil {
		return time.Since(start), err
	}
	defer f.Close()

	im := image.NewRGBA(image.Rect(0, 0, int(blockReq.FrameW), int(blockReq.FrameH)))

	data, err := dr.buffers.Paths.ReadDataIntoSlice(make([]rayPath, 0))
	if err != nil {
		return time.Since(start), err
	}
	paths := data.([]rayPath)

	for _, path := range paths {
		offset := path.pixelIndex << 2
		scaledThroughput := path.throughput.Mul(255.0)
		im.Pix[offset] = uint8(scaledThroughput[0])
		im.Pix[offset+1] = uint8(scaledThroughput[1])
		im.Pix[offset+2] = uint8(scaledThroughput[2])
		im.Pix[offset+3] = 255 // alpha
	}

	return time.Since(start), png.Encode(f, im)
}

// Debug emissive samples.
func debugEmissiveSamples(dr *deviceResources, blockReq *tracer.BlockRequest, imgFile string) (time.Duration, error) {
	start := time.Now()

	f, err := os.Create(imgFile)
	if err != nil {
		return time.Since(start), err
	}
	defer f.Close()

	im := image.NewRGBA(image.Rect(0, 0, int(blockReq.FrameW), int(blockReq.FrameH)))

	data, err := dr.buffers.Rays[2].ReadDataIntoSlice(make([]ray, 0))
	if err != nil {
		return time.Since(start), err
	}
	rays := data.([]ray)

	data, err = dr.buffers.EmissiveSamples.ReadDataIntoSlice(make([]types.Vec4, 0))
	if err != nil {
		return time.Since(start), err
	}
	emissiveSamples := data.([]types.Vec4)

	data, err = dr.buffers.Paths.ReadDataIntoSlice(make([]rayPath, 0))
	if err != nil {
		return time.Since(start), err
	}
	paths := data.([]rayPath)

	numRays := int(readCounter(dr, 2))
	for rayIndex := 0; rayIndex < numRays; rayIndex++ {
		pathIndex := int(rays[rayIndex].dir[3])
		offset := paths[pathIndex].pixelIndex << 2
		scaledThroughput := types.MinVec3(emissiveSamples[rayIndex].Mul(255.0).Vec3(), types.Vec3{255.0, 255.0, 255.0})
		im.Pix[offset] = uint8(scaledThroughput[0])
		im.Pix[offset+1] = uint8(scaledThroughput[1])
		im.Pix[offset+2] = uint8(scaledThroughput[2])
		im.Pix[offset+3] = 255 // alpha
	}

	return time.Since(start), png.Encode(f, im)
}

// Dump a copy of the RGBA framebuffer.
func DebugFrameBuffer(imgFile string) tracer.Stage {
	return appendPipeline(
		func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error) {
			start := time.Now()
			f, err := os.Create(imgFile)
			if err != nil {
				return time.Since(start), err
			}
			defer f.Close()

			im := image.NewRGBA(image.Rect(0, 0, int(blockReq.FrameW), int(blockReq.FrameH)))

			pix, err := tr.resources.buffers.FrameBuffer.ReadDataIntoSlice(make([]uint8, 0))
			if err != nil {
				return time.Since(start), err
			}
			im.Pix = pix.([]uint8)

			return time.Since(start), png.Encode(f, im)
		},
	)
}

func readCounter(dr *deviceResources, counterIndex uint32) uint32 {
	out := make([]uint32, 1)
	dr.buffers.RayCounters[counterIndex].ReadData(0, 0, 4, out)
	return out[0]
}
