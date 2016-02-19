package opencl

import "errors"

var (
	ErrContextCreationFailed  = errors.New("opencl tracer: could not create opencl context")
	ErrCmdQueueCreationFailed = errors.New("opencl tracer: could not create opencl command queue")
	ErrAlreadyAttached        = errors.New("opencl tracer: tracer already attached to a scene")
)
