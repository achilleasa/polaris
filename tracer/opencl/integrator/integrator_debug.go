package integrator

import (
	"image"
	"image/png"
	"math"
	"os"

	"github.com/achilleasa/go-pathtrace/types"
)

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

func (in *MonteCarlo) DebugIntersections(frameW, frameH uint32, imgFile string) error {
	f, err := os.Create(imgFile)
	if err != nil {
		return err
	}
	defer f.Close()

	im := image.NewRGBA(image.Rect(0, 0, int(frameW), int(frameH)))

	data, err := in.buffers.Intersections.ReadDataIntoSlice(make([]intersection, 0))
	if err != nil {
		return err
	}
	intersections := data.([]intersection)

	data, err = in.buffers.Paths.ReadDataIntoSlice(make([]rayPath, 0))
	if err != nil {
		return err
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
		offset := paths[hitIndex].pixelIndex
		im.Pix[offset] = normDepth
		im.Pix[offset+1] = normDepth
		im.Pix[offset+2] = normDepth
		im.Pix[offset+3] = 255 // alpha
	}

	return png.Encode(f, im)
}
