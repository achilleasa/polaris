package opencl

import "errors"

var (
	ErrContextCreationFailed  = errors.New("opencl tracer: could not create opencl context")
	ErrCmdQueueCreationFailed = errors.New("opencl tracer: could not create opencl command queue")
	ErrAlreadySetup           = errors.New("opencl tracer: tracer already set up")
	ErrProgramCreationFailed  = errors.New("opencl tracer: program creation failed")
	ErrProgramBuildFailed     = errors.New("opencl tracer: program compilation failed")
	ErrKernelCreationFailed   = errors.New("opencl tracer: could not create compute kernel")
	ErrGettingWorkgroupInfo   = errors.New("opencl tracer: could not get kernel work group info")
	ErrAllocatingBuffer       = errors.New("opencl tracer: could not allocate device buffer")
	ErrCopyingDataToDevice    = errors.New("opencl tracer: could not copy local data to device buffer")
	ErrSettingKernelArgument  = errors.New("opencl tracer: error setting kernel argument")
	ErrKernelExecutionFailed  = errors.New("opencl tracer: kernel execution failed")
	ErrCopyingDataToHost      = errors.New("opencl tracer: could not copy device data to local buffer")
	ErrUnsupportedChangeType  = errors.New("opencl tracer: unsupported change type")
	ErrInvalidChangeData      = errors.New("opencl tracer: invalid data type for change")
)
