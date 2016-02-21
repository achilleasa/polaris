package renderer

import "errors"

var (
	ErrNoTracers        = errors.New("renderer: no tracers attached")
	ErrSceneNotDefined  = errors.New("renderer: no scene defined")
	ErrCameraNotDefined = errors.New("renderer: no camera defined")
	ErrInterrupted      = errors.New("renderer: interrupted while rendering")
)
