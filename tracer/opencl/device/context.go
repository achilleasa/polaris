package device

import (
	"errors"
	"fmt"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

// Create a shared opencl context for the given device list.
func NewSharedContext(devices []*Device) (*cl.Context, error) {
	if len(devices) == 0 {
		return nil, errors.New("empty device list passed to NewSharedContext")
	}

	idList := make([]cl.DeviceId, len(devices))
	for i := 0; i < len(devices); i++ {
		idList[i] = devices[i].Id
	}

	var errCode cl.ErrorCode
	ctx := cl.CreateContext(nil, uint32(len(idList)), &idList[0], nil, nil, (*int32)(&errCode))
	if errCode != cl.SUCCESS {
		return nil, fmt.Errorf("could not create shared opencl context (error: %s; code %d)", ErrorName(errCode), errCode)
	}

	return ctx, nil
}
