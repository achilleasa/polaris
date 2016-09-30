package material

import (
	"errors"
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

const (
	ParamReflectance   = "reflectance"
	ParamSpecularity   = "specularity"
	ParamTransmittance = "transmittance"
	ParamRadiance      = "radiance"
	ParamIntIOR        = "intIOR"
	ParamExtIOR        = "extIOR"
	ParamScale         = "scale"
	ParamRoughness     = "roughness"
)

var (
	bxdfAllowedParameters = map[BxdfType]map[string]struct{}{
		BxdfEmissive: {
			ParamRadiance: struct{}{},
			ParamScale:    struct{}{},
		},
		BxdfDiffuse: {
			ParamReflectance: struct{}{},
		},
		BxdfConductor: {
			ParamSpecularity: struct{}{},
			ParamIntIOR:      struct{}{},
			ParamExtIOR:      struct{}{},
		},
		BxdfRoughtConductor: {
			ParamSpecularity: struct{}{},
			ParamIntIOR:      struct{}{},
			ParamExtIOR:      struct{}{},
			ParamRoughness:   struct{}{},
		},
		BxdfDielectric: {
			ParamSpecularity:   struct{}{},
			ParamTransmittance: struct{}{},
			ParamIntIOR:        struct{}{},
			ParamExtIOR:        struct{}{},
		},
		BxdfRoughDielectric: {
			ParamSpecularity:   struct{}{},
			ParamTransmittance: struct{}{},
			ParamIntIOR:        struct{}{},
			ParamExtIOR:        struct{}{},
			ParamRoughness:     struct{}{},
		},
	}
)

type ExprNode interface {
	Validate() error
}

type Vec3Node types.Vec3

type FloatNode float32

type MaterialNameNode string

type MaterialRefNode string

type TextureNode string

type BxdfParamNode struct {
	Name  string
	Value ExprNode
}

type BxdfParameterList []BxdfParamNode

type MixNode struct {
	Expressions [2]ExprNode
	Weights     [2]float32
}

type BumpMapNode struct {
	Expression ExprNode
	Texture    TextureNode
}

type MixMapNode struct {
	Expressions [2]ExprNode
	Texture     TextureNode
}

type NormalMapNode struct {
	Expression ExprNode
	Texture    TextureNode
}

type DisperseNode struct {
	Expression ExprNode
	IntIOR     Vec3Node
	ExtIOR     Vec3Node
}

type BxdfNode struct {
	Type       BxdfType
	Parameters BxdfParameterList
}

func (n Vec3Node) Validate() error {
	return nil
}

func (n FloatNode) Validate() error {
	return nil
}

func (n MaterialNameNode) Validate() error {
	if n == "" {
		return errors.New("material name cannot be empty")
	}
	return nil
}

func (n MaterialRefNode) Validate() error {
	if n == "" {
		return errors.New("material name cannot be empty")
	}
	return nil
}

func (n TextureNode) Validate() error {
	if n == "" {
		return errors.New("no texture path specified")
	}
	return nil
}

func (n BxdfParamNode) Validate() error {
	// Ensure energy conservation
	switch n.Name {
	case ParamReflectance:
		if v, isVec := n.Value.(Vec3Node); isVec && (v[0] >= 1.0 || v[1] >= 1.0 || v[2] >= 1.0) {
			return fmt.Errorf("energy conservation violation for Parameter %q; ensure that all vector components are < 1.0", n.Name)
		}
	case ParamSpecularity, ParamTransmittance:
		if v, isVec := n.Value.(Vec3Node); isVec && (v[0] > 1.0 || v[1] > 1.0 || v[2] > 1.0) {
			return fmt.Errorf("energy conservation violation for Parameter %q; ensure that all vector components are <= 1.0", n.Name)
		}
	case ParamRoughness:
		if v, isFloat := n.Value.(FloatNode); isFloat && v > 1.0 {
			return fmt.Errorf("values for Parameter %q must be in the [0, 1] range", n.Name)
		}
	case ParamIntIOR, ParamExtIOR:
		if v, isMat := n.Value.(MaterialNameNode); isMat {
			_, err := IOR(v)
			if err != nil {
				return err
			}
		}
	}

	return n.Value.Validate()
}

func (n BxdfParameterList) Validate() error {
	return nil
}

func (n BumpMapNode) Validate() error {
	if n.Expression == nil {
		return fmt.Errorf("missing expression argument for %q", "BumpMap")
	}
	err := n.Texture.Validate()
	if err != nil {
		return fmt.Errorf("BumpMap: %v", err)
	}
	return nil
}

func (n NormalMapNode) Validate() error {
	if n.Expression == nil {
		return fmt.Errorf("missing expression argument for %q", "NormalMap")
	}
	err := n.Texture.Validate()
	if err != nil {
		return fmt.Errorf("NormalMap: %v", err)
	}
	return nil
}

func (n DisperseNode) Validate() error {
	if n.Expression == nil {
		return fmt.Errorf("missing expression argument for %q", "Disperse")
	}
	if types.Vec3(n.IntIOR).MaxComponent() == 0.0 && types.Vec3(n.ExtIOR).MaxComponent() == 0.0 {
		return fmt.Errorf("Disperse: at least one of the intIOR and extIOR parameters must contain a non-zero value")
	}
	return nil
}

func (n MixMapNode) Validate() error {
	var err error
	for argIndex, arg := range n.Expressions {
		if arg == nil {
			return fmt.Errorf("missing expression argument %d for %q", argIndex, "mixMap")
		}
		err = arg.Validate()
		if err != nil {
			return fmt.Errorf("mixMap argument %d: %v", argIndex, err)
		}
	}

	err = n.Texture.Validate()
	if err != nil {
		return fmt.Errorf("MixMap: %v", err)
	}
	return nil
}

func (n MixNode) Validate() error {
	var err error
	for argIndex, arg := range n.Expressions {
		if arg == nil {
			return fmt.Errorf("missing expression argument %d for %q", argIndex, "mix")
		}
		err = arg.Validate()
		if err != nil {
			return fmt.Errorf("mix argument %d: %v", argIndex, err)
		}
		if n.Weights[argIndex] < 0 || n.Weights[argIndex] > 1.0 {
			return fmt.Errorf("mix weight %d: value must be in the [0, 1] range", argIndex)
		}
	}

	if n.Weights[0]+n.Weights[1] != 1.0 {
		return fmt.Errorf("mix weight sum must be equal to 1.0")
	}

	return nil
}

func (n BxdfNode) Validate() error {
	if n.Type == bxdfInvalid {
		return fmt.Errorf("invalid BXDF type")
	}

	// Validate list of allowed Parameter names
	var err error
	for _, Param := range n.Parameters {
		if _, isAllowed := bxdfAllowedParameters[n.Type][Param.Name]; !isAllowed {
			return fmt.Errorf("bxdf type %q does not support Parameter %q", n.Type, Param.Name)
		}

		// Validate Parameter
		if err = Param.Validate(); err != nil {
			return err
		}
	}

	return nil
}
