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
	rayPacketIntersectionQuery
	// pt kernels
	shadeHits
	shadePrimaryRayMisses
	shadeIndirectRayMisses
	accumulateEmissiveSamples
	// hdr kernels
	tonemapSimpleReinhard
	// utils
	clearAccumulator
	// Debugging
	debugClearBuffer
	debugRayIntersectionDepth
	debugRayIntersectionNormals
	debugEmissiveSamples
	debugThroughput
	debugAccumulator
	//
	debugMicrofacet
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
	case rayPacketIntersectionQuery:
		return "rayPacketIntersectionQuery"
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
	case debugClearBuffer:
		return "debugClearBuffer"
	case debugRayIntersectionDepth:
		return "debugRayIntersectionDepth"
	case debugRayIntersectionNormals:
		return "debugRayIntersectionNormals"
	case debugEmissiveSamples:
		return "debugEmissiveSamples"
	case debugThroughput:
		return "debugThroughput"
	case debugAccumulator:
		return "debugAccumulator"
	case debugMicrofacet:
		return "debugMicrofacet"
	default:
		panic(fmt.Sprintf("Unsupported kernel type: %d", kt))
	}
}
