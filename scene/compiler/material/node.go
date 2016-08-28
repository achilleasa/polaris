package material

import (
	"errors"
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

const (
	paramReflectance   = "reflectance"
	paramSpecularity   = "specularity"
	paramTransmittance = "transmittance"
	paramRadiance      = "radiance"
	paramIntIOR        = "intIOR"
	paramExtIOR        = "extIOR"
	paramScale         = "scale"
	paramRoughness     = "roughness"
)

var (
	bxdfAllowedParameters = map[BxdfType]map[string]struct{}{
		BxdfEmissive: {
			paramRadiance: struct{}{},
			paramScale:    struct{}{},
		},
		BxdfDiffuse: {
			paramReflectance: struct{}{},
		},
		BxdfConductor: {
			paramSpecularity: struct{}{},
			paramIntIOR:      struct{}{},
			paramExtIOR:      struct{}{},
		},
		BxdfRoughtConductor: {
			paramSpecularity: struct{}{},
			paramIntIOR:      struct{}{},
			paramExtIOR:      struct{}{},
			paramRoughness:   struct{}{},
		},
		BxdfDielectric: {
			paramSpecularity:   struct{}{},
			paramTransmittance: struct{}{},
			paramIntIOR:        struct{}{},
			paramExtIOR:        struct{}{},
		},
		BxdfRoughDielectric: {
			paramSpecularity:   struct{}{},
			paramTransmittance: struct{}{},
			paramIntIOR:        struct{}{},
			paramExtIOR:        struct{}{},
			paramRoughness:     struct{}{},
		},
	}
)

type exprNode interface {
	Validate() error
}

type vec3Node types.Vec3

type floatNode float32

type materialNameNode string

type textureNode string

type bxdfParamNode struct {
	Name  string
	Value exprNode
}

type bxdfParameterList []bxdfParamNode

type mixNode struct {
	Expressions [2]exprNode
	Weights     [2]float32
}

type bumpMapNode struct {
	Expression exprNode
	Texture    textureNode
}

type normalMapNode struct {
	Expression exprNode
	Texture    textureNode
}

type bxdfNode struct {
	Type       BxdfType
	Parameters bxdfParameterList
}

func (n vec3Node) Validate() error {
	return nil
}

func (n floatNode) Validate() error {
	return nil
}

func (n materialNameNode) Validate() error {
	if n == "" {
		return errors.New("material name cannot be empty")
	}
	return nil
}

func (n textureNode) Validate() error {
	if n == "" {
		return errors.New("no texture path specified")
	}
	return nil
}

func (n bxdfParamNode) Validate() error {
	// Ensure energy conservation
	switch n.Name {
	case paramReflectance:
		if v, isVec := n.Value.(vec3Node); isVec && (v[0] >= 1.0 || v[1] >= 1.0 || v[2] >= 1.0) {
			return fmt.Errorf("energy conservation violation for parameter %q; ensure that all vector components are < 1.0", n.Name)
		}
	case paramSpecularity, paramTransmittance:
		if v, isVec := n.Value.(vec3Node); isVec && (v[0] > 1.0 || v[1] > 1.0 || v[2] > 1.0) {
			return fmt.Errorf("energy conservation violation for parameter %q; ensure that all vector components are <= 1.0", n.Name)
		}
	case paramRoughness:
		if v, isFloat := n.Value.(floatNode); isFloat && v > 1.0 {
			return fmt.Errorf("values for parameter %q must be in the [0, 1] range", n.Name)
		}
	case paramIntIOR, paramExtIOR:
		if v, isMat := n.Value.(materialNameNode); isMat {
			_, err := iorForMaterial(v)
			if err != nil {
				return err
			}
		}
	}

	return n.Value.Validate()
}

func (n bxdfParameterList) Validate() error {
	return nil
}

func (n bumpMapNode) Validate() error {
	if n.Expression == nil {
		return fmt.Errorf("missing expression argument for %q", "bumpMap")
	}
	err := n.Texture.Validate()
	if err != nil {
		return fmt.Errorf("bumpMap: %v", err)
	}
	return nil
}

func (n normalMapNode) Validate() error {
	if n.Expression == nil {
		return fmt.Errorf("missing expression argument for %q", "normalMap")
	}
	err := n.Texture.Validate()
	if err != nil {
		return fmt.Errorf("normalMap: %v", err)
	}
	return nil
}

func (n mixNode) Validate() error {
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

	if n.Weights[0]+n.Weights[1] > 1.0 {
		return fmt.Errorf("mix weight sum must be <= 1.0")
	}

	return nil
}

func (n bxdfNode) Validate() error {
	if n.Type == bxdfInvalid {
		return fmt.Errorf("invalid BXDF type")
	}

	// Validate list of allowed parameter names
	var err error
	for _, param := range n.Parameters {
		if _, isAllowed := bxdfAllowedParameters[n.Type][param.Name]; !isAllowed {
			return fmt.Errorf("bxdf type %q does not support parameter %q", n.Type, param.Name)
		}

		// Validate parameter
		if err = param.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// The ParsedExpression struct holds the AST tree for a material expression.
type ParsedExpression struct {
	expressionAST exprNode
}

// Perform semantic validation on the parsed expression.
func (pe *ParsedExpression) Validate() error {
	return pe.expressionAST.Validate()
}
