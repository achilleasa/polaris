package material

import "github.com/achilleasa/go-pathtrace/types"

var (
	DefaultRoughness      float32 = 0.1
	DefaultReflectance            = types.Vec4{0.2, 0.2, 0.2, 0.0}
	DefaultSpecularity            = types.Vec4{1.0, 1.0, 1.0, 0.0}
	DefaultTransmittance          = types.Vec4{1.0, 1.0, 1.0, 0.0}
	DefaultRadiance               = types.Vec4{1.0, 1.0, 1.0, 0.0}
	DefaultRadianceScaler float32 = 1.0
	DefaultIntIOR                 = KnownIORs["Glass"]
	DefaultExtIOR                 = KnownIORs["Air"]
)
