package integrator

import "fmt"

type kernelType uint8

// The list of kernels that implement the tracer.
const (
	generatePrimaryRays kernelType = iota
	rayIntersectionTest
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
	default:
		panic(fmt.Sprintf("Unsupported kernel type: %d", kt))
	}
}
