package scene

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/achilleasa/go-pathtrace/types"
)

type asciiScene struct {
	cameraExposure float32
	cameraFov      float32
	cameraEye      types.Vec3
	cameraLook     types.Vec3
	cameraUp       types.Vec3

	// List of scene primitives.
	primitives []*Primitive

	// A map of material names to material instances. This map contains
	// all materials available in a material library.
	availableMaterials map[string]*Material

	// A map of material names to used material indices for materials referenced
	// by scene primitives. This is a subset of available materials.
	referencedMaterials map[string]int

	// A list of referenced materials. Primitives material index property
	// points to a material in this list.
	usedMaterials []*Material

	// Currently selected material index
	curMaterial int

	// List of vertices and uv coords.
	vertexList []types.Vec3
	uvList     []types.Vec2

	// An error stack that provides additional error information when
	// scene files include other files (models, mat libs e.t.c)
	errStack []string
}

// Parse a scene stored in a file.
func Parse(filename string) (*Scene, error) {
	if strings.HasSuffix(filename, ".obj") {
		ps := &asciiScene{
			// Init camera defaults
			cameraExposure: 1.0,
			cameraFov:      45.0,
			cameraLook:     types.Vec3{0.0, 0.0, -1.0},
			cameraUp:       types.Vec3{0.0, 1.0, 0.0},
			// Init other containers
			primitives:          make([]*Primitive, 0),
			availableMaterials:  make(map[string]*Material, 0),
			referencedMaterials: make(map[string]int, 0),
			usedMaterials:       make([]*Material, 0),
			curMaterial:         -1,
			vertexList:          make([]types.Vec3, 0),
			uvList:              make([]types.Vec2, 0),
			errStack:            make([]string, 0),
		}
		err := ps.Parse(filename)
		if err != nil {
			return nil, err
		}

		// Package scene
		return ps.Scene(), nil
	}

	return nil, fmt.Errorf("scene: unsupported file format")
}

// Generate an error message that also includes any data in the error stack.
func (as *asciiScene) emitError(file string, line int, msgFormat string, args ...interface{}) error {
	msg := fmt.Sprintf(msgFormat, args...)

	var errMsg string
	if file != "" {
		errMsg = strings.Trim(
			fmt.Sprintf("[%s: %d] error: %s\n%s", file, line, msg, strings.Join(as.errStack, "\n")),
			"\n",
		)
	} else {
		errMsg = strings.Trim(
			fmt.Sprintf("error: %s\n%s", msg, strings.Join(as.errStack, "\n")),
			"\n",
		)
	}

	return fmt.Errorf(errMsg)
}

// Push a frame to the error stack.
func (as *asciiScene) pushFrame(msg string) {
	as.errStack = append([]string{msg}, as.errStack...)
}

// Pop a frame from the error stack.
func (as *asciiScene) popFrame() {
	as.errStack = as.errStack[1:]
}

// Generate package scene from the parsed scene data
func (as *asciiScene) Scene() *Scene {
	sc := &Scene{
		Camera:                   NewCamera(as.cameraFov, as.cameraExposure),
		Materials:                make([]Material, len(as.usedMaterials)),
		Primitives:               make([]Primitive, len(as.primitives)),
		EmissivePrimitiveIndices: make([]int, 0),
		matNameToIndex:           as.referencedMaterials,
	}

	// Setup camera
	sc.Camera.LookAt(as.cameraEye, as.cameraLook, as.cameraUp)

	// Package materials
	for idx, mat := range as.usedMaterials {
		sc.Materials[idx] = *mat
	}

	// Package primitives and build emissive primitive list
	for idx, prim := range as.primitives {
		sc.Primitives[idx] = *prim
		matIndex := int(prim.properties[1])
		if sc.Materials[matIndex].properties[0] == emissiveMaterial {
			sc.EmissivePrimitiveIndices = append(sc.EmissivePrimitiveIndices, idx)
		}
	}

	return sc
}

// Create and select a default material for surfaces not using one.
func (as *asciiScene) defaultMaterialIndex() int {
	matName := ""

	// Search for material in referenced list
	matIndex, exists := as.referencedMaterials[matName]
	if !exists {
		// Add it to used materials
		as.usedMaterials = append(as.usedMaterials, &Material{diffuse: types.Vec4{0.75, 0.75, 0.75}})
		matIndex = len(as.usedMaterials) - 1
		as.referencedMaterials[matName] = matIndex
	}
	as.curMaterial = matIndex
	return as.curMaterial
}

// Parse wavefront object scene format.
func (as *asciiScene) Parse(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return as.emitError("", 0, "could not open %s", filename)
	}
	defer f.Close()

	// Get path to file. We use this to lookup relative includes
	dir := filepath.Dir(filename) + "/"

	// line stats
	var lineNum int = 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 {
			continue
		}

		switch lineTokens[0] {
		case "call":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'call'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			as.pushFrame(fmt.Sprintf("referenced from %s:%d [call]", filename, lineNum))

			// If this is not an absolute path prepend the dir of current file
			extFile := lineTokens[1]
			if !filepath.IsAbs(extFile) {
				extFile = dir + extFile
			}

			// Try parsing file
			err := as.Parse(extFile)
			if err != nil {
				return err
			}
			as.popFrame()

		case "mtllib":
			if len(lineTokens) == 1 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'mtllib'; expected 1 or more arguments")
			}

			for tokIdx := 1; tokIdx < len(lineTokens); tokIdx++ {
				as.pushFrame(fmt.Sprintf("referenced from %s:%d [mtllib]", filename, lineNum))

				// If this is not an absolute path prepend the dir of current file
				extFile := lineTokens[tokIdx]
				if !filepath.IsAbs(extFile) {
					extFile = dir + extFile
				}

				// Try parsing file
				err := as.parseMaterials(extFile)
				if err != nil {
					return err
				}
				as.popFrame()
			}
		case "usemtl":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'usemtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			// Search for material in used list
			matName := lineTokens[1]
			matIndex, exists := as.referencedMaterials[matName]
			if !exists {
				// Search material library
				matInstance, exists := as.availableMaterials[matName]
				if !exists {
					return as.emitError(filename, lineNum, "undefined material with name '%s'", matName)
				}

				// Add it to used materials
				as.usedMaterials = append(as.usedMaterials, matInstance)
				matIndex = len(as.usedMaterials) - 1
				as.referencedMaterials[matName] = matIndex
			}

			// Activate material
			as.curMaterial = matIndex
		case "v":
			if len(lineTokens) < 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'v'; expected at least 3 arguments; got %d", len(lineTokens)-1)
			}

			v := types.Vec3{}
			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse vertex coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}
			as.vertexList = append(as.vertexList, v)
		case "vt":
			if len(lineTokens) < 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'vt'; expected at least 2 arguments; got %d", len(lineTokens)-1)
			}

			v := types.Vec2{}
			for tokIdx := 1; tokIdx <= 2; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse vertex tex coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}
			as.uvList = append(as.uvList, v)
		case "f":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'f'; expected 3 arguments for triangular face; got %d. Select the triangulation option in your exporter.", len(lineTokens)-1)
			}

			// Face format consists of 3 arguments. Each arg consists
			// of 1 to 3 args separated by a slash character:
			// - vertexIndex
			// - vertexIndex/uvIndex
			// - vertexIndex//normalIndex
			// - vertexIndex/uvIndex/normalIndex
			//
			// Indices start from 1 and may be negative to indicate
			// an offset off the end of the vertex/uv list
			var vertices [3]types.Vec3
			var uv [3]types.Vec2
			expIndices := 0
			for arg := 0; arg < 3; arg++ {
				vTokens := strings.Split(lineTokens[arg+1], "/")

				// The first arg defines the format for the following args
				if arg == 0 {
					expIndices = len(vTokens)
				} else if len(vTokens) != expIndices {
					return as.emitError(filename, lineNum, "expected each face argument to contain %d indices; arg %d contains %d indices", expIndices, arg, len(vTokens))
				}

				if vTokens[0] == "" {
					return as.emitError(filename, lineNum, "face argument %d does not include a vertex index", arg)
				}
				index, err := strconv.ParseInt(vTokens[0], 10, 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse vertex index for face argument %d: %s", arg, err.Error())
				}

				// If index is negative it refers to the end of the vertex list
				var vOffset int = 0
				if index < 0 {
					vOffset = len(as.vertexList) - int(index)
				} else {
					vOffset = int(index - 1)
				}
				if vOffset < 0 || vOffset >= len(as.vertexList) {
					return as.emitError(filename, lineNum, "vertex index out of bounds for face argument %d", arg)
				}
				vertices[arg] = as.vertexList[vOffset]

				// Parse UV coords if specified
				if expIndices == 1 || vTokens[1] == "" {
					continue
				}

				index, err = strconv.ParseInt(vTokens[1], 10, 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse vertex tex index for face argument %d: %s", arg, err.Error())
				}

				// If index is negative it refers to the end of the uv list
				if index < 0 {
					vOffset = len(as.uvList) - int(index)
				} else {
					vOffset = int(index - 1)
				}
				if vOffset < 0 || vOffset >= len(as.uvList) {
					return as.emitError(filename, lineNum, "vertex tex index out of bounds for face argument %d", arg)
				}
				uv[arg] = as.uvList[vOffset]
			}

			// If no material defined select the default
			if as.curMaterial < 0 {
				as.curMaterial = as.defaultMaterialIndex()
			}

			prim := NewTriangle(vertices, uv)
			prim.properties[1] = float32(as.curMaterial)
			as.primitives = append(as.primitives, prim)
		case "plane":
			if len(lineTokens) != 5 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'plane'; expected 4 arguments (x, y, z, D); got %d", len(lineTokens)-1)
			}

			v := types.Vec4{}
			for tokIdx := 1; tokIdx <= 4; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse plane coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}

			// If no material defined select the default
			if as.curMaterial < 0 {
				as.curMaterial = as.defaultMaterialIndex()
			}

			prim := NewPlane(v.Vec3(), v[3])
			prim.properties[1] = float32(as.curMaterial)
			as.primitives = append(as.primitives, prim)
		case "sphere":
			if len(lineTokens) != 5 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'sphere'; expected 4 arguments (x, y, z, radius); got %d", len(lineTokens)-1)
			}

			v := types.Vec4{}
			for tokIdx := 1; tokIdx <= 4; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse sphere coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}

			// If no material defined select the default
			if as.curMaterial < 0 {
				as.curMaterial = as.defaultMaterialIndex()
			}

			prim := NewSphere(v.Vec3(), v[3])
			prim.properties[1] = float32(as.curMaterial)
			as.primitives = append(as.primitives, prim)
		case "box":
			if len(lineTokens) != 7 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'box'; expected 6 arguments (xmin, ymin, zmin, xmax, ymax, zmax); got %d", len(lineTokens)-1)
			}

			bmin := types.Vec3{}
			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse box min coordinate %d: %s", tokIdx-1, err.Error())
				}
				bmin[tokIdx-1] = float32(coord)
			}

			bmax := types.Vec3{}
			for tokIdx := 4; tokIdx <= 6; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse box max coordinate %d: %s", tokIdx-4, err.Error())
				}
				bmax[tokIdx-4] = float32(coord)
			}

			// If no material defined select the default
			if as.curMaterial < 0 {
				as.curMaterial = as.defaultMaterialIndex()
			}
			prim := NewBox(bmin, bmax)
			prim.properties[1] = float32(as.curMaterial)
			as.primitives = append(as.primitives, prim)
		case "camera_fov":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'camera_fov'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			v, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return as.emitError(filename, lineNum, "could not parse camera fov: %s", err.Error())
			}
			as.cameraFov = float32(v)
		case "camera_exposure":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'camera_exposure'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			v, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return as.emitError(filename, lineNum, "could not parse camera exposure: %s", err.Error())
			}
			as.cameraExposure = float32(v)
		case "camera_eye":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'camera_eye'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse camera eye coordinate %d: %s", tokIdx-1, err.Error())
				}
				as.cameraEye[tokIdx-1] = float32(coord)
			}
		case "camera_look":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'camera_look'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse camera look coordinate %d: %s", tokIdx-1, err.Error())
				}
				as.cameraLook[tokIdx-1] = float32(coord)
			}
		case "camera_up":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'camera_up'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse camera up coordinate %d: %s", tokIdx-1, err.Error())
				}
				as.cameraUp[tokIdx-1] = float32(coord)
			}
		}
	}

	return nil
}

// Parse a wavefront material library.
func (as *asciiScene) parseMaterials(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return as.emitError("", 0, "could not open %s", filename)
	}
	defer f.Close()

	// line stats
	var lineNum int = 0

	scanner := bufio.NewScanner(f)

	var curMaterial *Material = nil
	var matName string = ""

	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 {
			continue
		}

		switch lineTokens[0] {
		case "newmtl":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'newmtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			matName = lineTokens[1]
			if _, exists := as.availableMaterials[matName]; exists {
				return as.emitError(filename, lineNum, "material '%s' already defined", matName)
			}

			// Allocate new material and add it to library
			curMaterial = &Material{}
			as.availableMaterials[matName] = curMaterial
		case "Kd":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'Kd'; expected 3 arguments; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return as.emitError(filename, lineNum, "got 'Kd' without a 'newmtl'")
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				val, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse diffuse component %d: %s", tokIdx-1, err.Error())
				}
				curMaterial.diffuse[tokIdx-1] = float32(val)
			}
		case "Ke":
			if len(lineTokens) != 4 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'Ke'; expected 3 arguments; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return as.emitError(filename, lineNum, "got 'Ke' without a 'newmtl'")
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				val, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return as.emitError(filename, lineNum, "could not parse emissive component %d: %s", tokIdx-1, err.Error())
				}
				curMaterial.emissive[tokIdx-1] = float32(val)
			}
		case "ior":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'ior'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return as.emitError(filename, lineNum, "got 'ior' without a 'newmtl'")
			}

			val, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return as.emitError(filename, lineNum, "could not parse IOR value: %s", err.Error())
			}
			curMaterial.properties[1] = float32(val)
		case "roughness":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'roughness'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return as.emitError(filename, lineNum, "got 'roughness' without a 'newmtl'")
			}

			val, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return as.emitError(filename, lineNum, "could not parse roughness value: %s", err.Error())
			}
			curMaterial.properties[2] = float32(val)
		case "stype":
			if len(lineTokens) != 2 {
				return as.emitError(filename, lineNum, "unsupported syntax for 'stype'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return as.emitError(filename, lineNum, "got 'stype' without a 'newmtl'")
			}

			switch lineTokens[1] {
			case "diffuse":
				curMaterial.properties[0] = diffuseMaterial
			case "specular":
				curMaterial.properties[0] = specularMaterial
			case "refractive":
				curMaterial.properties[0] = refractiveMaterial
			case "emissive":
				curMaterial.properties[0] = emissiveMaterial
			default:
				return as.emitError(filename, lineNum, "unknown surface type '%s'", lineTokens[1])
			}
		}
	}

	return nil
}
