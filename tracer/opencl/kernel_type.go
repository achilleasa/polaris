package opencl

import "fmt"

type kernelType uint8

// The list of kernels that implement the tracer.
const (
	// camera kernels
	generatePrimaryRays kernelType = iota
	// intersection kernels
	rayIntersectionTest
	rayIntersectionQuery
	packetIntersectionQuery
	// pt kernels
	shadeHits
	shadePrimaryRayMisses
	shadeIndirectRayMisses
	accumulateEmissiveSamples
	// hdr kernels
	tonemapSimpleReinhard
	// utils
	clearAccumulator
	//
	numKernels
)

// Implements Stringer; map kernel type to the kernel name as defined in the CL source files.
func (kt kernelType) String() string {
	switch kt {
	case generatePrimaryRays:
		return "generatePrimaryRays"
	case rayIntersectionTest:
		return "rayIntersectionTest"
	case rayIntersectionQuery:
		return "rayIntersectionQuery"
	case packetIntersectionQuery:
		return "packetIntersectionQuery"
	case shadeHits:
		return "shadeHits"
	case shadePrimaryRayMisses:
		return "shadePrimaryRayMisses"
	case shadeIndirectRayMisses:
		return "shadeIndirectRayMisses"
	case accumulateEmissiveSamples:
		return "accumulateEmissiveSamples"
	case tonemapSimpleReinhard:
		return "tonemapSimpleReinhard"
	case clearAccumulator:
		return "clearAccumulator"
	default:
		panic(fmt.Sprintf("Unsupported kernel type: %d", kt))
	}
}
