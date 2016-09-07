package material

// BxdfType represents the surface types supported by the renderer.
type BxdfType uint32

const (
	bxdfInvalid BxdfType = 1 << iota
	//
	BxdfEmissive
	BxdfDiffuse
	BxdfConductor
	BxdfRoughtConductor
	BxdfDielectric
	BxdfRoughDielectric
	//
	bxdfLastEntry
)

// Lookup bxdf type by its name.
func bxdfTypeFromName(name string) BxdfType {
	switch name {
	case "emissive":
		return BxdfEmissive
	case "diffuse":
		return BxdfDiffuse
	case "conductor":
		return BxdfConductor
	case "roughConductor":
		return BxdfRoughtConductor
	case "dielectric":
		return BxdfDielectric
	case "roughDielectric":
		return BxdfRoughDielectric
	}

	return bxdfInvalid
}

func (t BxdfType) String() string {
	switch t {
	case BxdfEmissive:
		return "emissive"
	case BxdfDiffuse:
		return "diffuse"
	case BxdfConductor:
		return "conductor"
	case BxdfRoughtConductor:
		return "roughConductor"
	case BxdfDielectric:
		return "dielectric"
	case BxdfRoughDielectric:
		return "roughDielectric"
	}

	return "invalid"
}

// Helper function to check if a value represents a bxdf type.
func IsBxdfType(t uint32) bool {
	return t > uint32(bxdfInvalid) && t < uint32(bxdfLastEntry)
}
