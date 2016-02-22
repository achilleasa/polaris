package opencl

import "errors"

var (
	ErrContextCreationFailed  = errors.New("opencl tracer: could not create opencl context")
	ErrCmdQueueCreationFailed = errors.New("opencl tracer: could not create opencl command queue")
	ErrAlreadyAttached        = errors.New("opencl tracer: tracer already attached to a scene")
	ErrProgramCreationFailed  = errors.New("opencl tracer: program creation failed")
	ErrProgramBuildFailed     = errors.New("opencl tracer: program compilation failed")
	ErrKernelCreationFailed   = errors.New("opencl tracer: could not create compute kernel")
	ErrGettingWorkgroupInfo   = errors.New("opencl tracer: could not get kernel work group info")
)
