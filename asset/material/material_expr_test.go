package material

import "testing"

func TestParser(t *testing.T) {
	validExpr := []string{
		`diffuse()`,
		`diffuse(reflectance: {0.9, 0.9, 0.9})`,
		`diffuse(reflectance: "texture.jpg")`,
		`dielectric(specularity: "texture.jpg", intIOR: "gold", extIOR: "air")`,
		`dielectric(specularity: "texture.jpEg", transmittance: {.9,.9,.9}, intIOR: 1.33, extIOR: "air")`,
		`roughDielectric(specularity: "texture.jpEg", transmittance: {1,1,1}, intIOR: 1.33, extIOR: "air", roughness: 0.2)`,
		`conductor(specularity: "texture.jpg")`,
		`roughConductor(specularity: {.3,.3,.3}, intIOR: "gold", roughness: 1)`,
		`emissive(radiance: {1,1,1}, scale: 10)`,
		`bumpMap(conductor(specularity: "texture.jpg"), "foo.jpg")`,
		`normalMap(conductor(specularity: "texture.jpg"), "foo.jpg")`,
		`mix(diffuse(reflectance:{0.2, 0.2, 0.2}), conductor(specularity: "texture.jpg"), 0.2, 0.8)`,
	}

	for index, expr := range validExpr {
		parsedExpression, err := ParseExpression(expr)
		if err != nil {
			t.Errorf("[expr %d] parse error for %q: %v", index, expr, err)
			continue
		}

		err = parsedExpression.Validate()
		if err != nil {
			t.Errorf("[expr %d] semantic validatione error for %q: %v", index, expr, err)
			continue
		}
	}
}

func TestSemanticParseErrors(t *testing.T) {
	invalidExpr := []string{
		`diffuse(specularity: {0.9, 0.9, 0.9})`,
		`diffuse(reflectance: {1.0, 0.9, 0.9})`,
		`conductor(roughness: "texture.jpg")`,
		`roughConductor(specularity: {.3,.3,.3}, intIOR: "gold!!!", roughness: 1)`,
		`roughConductor(specularity: {.3,.3,.3}, intIOR: 1.2, extIOR: "foo", roughness: 1)`,
		`dielectric(transmittance: {1.3,.3,.3})`,
		`mix(diffuse(), conductor(), 0.2, 1.0)`,
	}

	for index, expr := range invalidExpr {
		pe, err := ParseExpression(expr)
		if err != nil {
			t.Errorf("[expr %d] parse error for %q: %v", index, expr, err)
			continue
		}

		err = pe.Validate()
		if err == nil {
			t.Errorf("[expr %d] expected a semantic parse error for %q", index, expr)
		}
	}
}
