package io

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/scene/tools"
	"github.com/achilleasa/go-pathtrace/types"
)

type textSceneReader struct {
	// The name of the file where the scene is stored
	sceneFile string

	// Camera parameters.
	cameraExposure float32
	cameraFov      float32
	cameraEye      types.Vec3
	cameraLook     types.Vec3
	cameraUp       types.Vec3

	// List of wrapped scene primitives.
	primitives []*scene.BvhPrimitive

	// A map of material names to material instances. This map contains
	// all materials available in a material library.
	availableMaterials map[string]*scene.Material

	// A map of material names to used material indices for materials referenced
	// by scene primitives. This is a subset of available materials.
	referencedMaterials map[string]int

	// A list of referenced materials. Primitives material index property
	// points to a material in this list.
	usedMaterials []*scene.Material

	// Currently selected material index
	curMaterial int

	// List of vertices and uv coords.
	vertexList []types.Vec3
	uvList     []types.Vec2

	// An error stack that provides additional error information when
	// scene files include other files (models, mat libs e.t.c)
	errStack []string
}

// Create a new text scene reader.
func newTextSceneReader(sceneFile string) *textSceneReader {
	return &textSceneReader{
		sceneFile: sceneFile,
		// Init camera defaults
		cameraExposure: 1.0,
		cameraFov:      45.0,
		cameraLook:     types.Vec3{0.0, 0.0, -1.0},
		cameraUp:       types.Vec3{0.0, 1.0, 0.0},
		// Init other containers
		primitives:          make([]*scene.BvhPrimitive, 0),
		availableMaterials:  make(map[string]*scene.Material, 0),
		referencedMaterials: make(map[string]int, 0),
		usedMaterials:       make([]*scene.Material, 0),
		curMaterial:         -1,
		vertexList:          make([]types.Vec3, 0),
		uvList:              make([]types.Vec2, 0),
		errStack:            make([]string, 0),
	}
}

// Read scene definition.
func (p *textSceneReader) Read() (*scene.Scene, error) {
	// Parse scene
	err := p.parse(p.sceneFile)
	if err != nil {
		return nil, err
	}

	// Generate packed scene representation
	sc := &scene.Scene{
		Camera:                   scene.NewCamera(p.cameraFov, p.cameraExposure),
		Materials:                make([]scene.Material, len(p.usedMaterials)),
		EmissivePrimitiveIndices: make([]uint32, 0),
		MatNameToIndex:           p.referencedMaterials,
	}

	// Setup camera
	sc.Camera.LookAt(p.cameraEye, p.cameraLook, p.cameraUp)

	// Package materials
	for idx, mat := range p.usedMaterials {
		sc.Materials[idx] = *mat
	}

	// Create BVH tree
	sc.BvhNodes, sc.Primitives = tools.BuildBVH(p.primitives, 2)

	// Detect primitives linked to emissive materials
	for idx, prim := range sc.Primitives {
		matIndex := int(prim.Properties[1])
		if sc.Materials[matIndex].Properties[0] == scene.EmissiveMaterial {
			sc.EmissivePrimitiveIndices = append(sc.EmissivePrimitiveIndices, uint32(idx))
		}
	}

	return sc, nil
}

// Generate an error message that also includes any data in the error stack.
func (p *textSceneReader) emitError(file string, line int, msgFormat string, args ...interface{}) error {
	msg := fmt.Sprintf(msgFormat, args...)

	var errMsg string
	if file != "" {
		errMsg = strings.Trim(
			fmt.Sprintf("[%s: %d] error: %s\n%s", file, line, msg, strings.Join(p.errStack, "\n")),
			"\n",
		)
	} else {
		errMsg = strings.Trim(
			fmt.Sprintf("error: %s\n%s", msg, strings.Join(p.errStack, "\n")),
			"\n",
		)
	}

	return fmt.Errorf(errMsg)
}

// Push a frame to the error stack.
func (p *textSceneReader) pushFrame(msg string) {
	p.errStack = append([]string{msg}, p.errStack...)
}

// Pop a frame from the error stack.
func (p *textSceneReader) popFrame() {
	p.errStack = p.errStack[1:]
}

// Create and select a default material for surfaces not using one.
func (p *textSceneReader) defaultMaterialIndex() int {
	matName := ""

	// Search for material in referenced list
	matIndex, exists := p.referencedMaterials[matName]
	if !exists {
		// Add it to used materials
		p.usedMaterials = append(p.usedMaterials, &scene.Material{Diffuse: types.Vec4{0.75, 0.75, 0.75}})
		matIndex = len(p.usedMaterials) - 1
		p.referencedMaterials[matName] = matIndex
	}
	p.curMaterial = matIndex
	return p.curMaterial
}

// Parse wavefront object scene format.
func (p *textSceneReader) parse(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return p.emitError("", 0, "could not open %s", filename)
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
				return p.emitError(filename, lineNum, "unsupported syntax for 'call'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			p.pushFrame(fmt.Sprintf("referenced from %s:%d [call]", filename, lineNum))

			// If this is not an absolute path prepend the dir of current file
			extFile := lineTokens[1]
			if !filepath.IsAbs(extFile) {
				extFile = dir + extFile
			}

			// Try parsing file
			err := p.parse(extFile)
			if err != nil {
				return err
			}
			p.popFrame()

		case "mtllib":
			if len(lineTokens) == 1 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'mtllib'; expected 1 or more arguments")
			}

			for tokIdx := 1; tokIdx < len(lineTokens); tokIdx++ {
				p.pushFrame(fmt.Sprintf("referenced from %s:%d [mtllib]", filename, lineNum))

				// If this is not an absolute path prepend the dir of current file
				extFile := lineTokens[tokIdx]
				if !filepath.IsAbs(extFile) {
					extFile = dir + extFile
				}

				// Try parsing file
				err := p.parseMaterials(extFile)
				if err != nil {
					return err
				}
				p.popFrame()
			}
		case "usemtl":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'usemtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			// Search for material in used list
			matName := lineTokens[1]
			matIndex, exists := p.referencedMaterials[matName]
			if !exists {
				// Search material library
				matInstance, exists := p.availableMaterials[matName]
				if !exists {
					return p.emitError(filename, lineNum, "undefined material with name '%s'", matName)
				}

				// Add it to used materials
				p.usedMaterials = append(p.usedMaterials, matInstance)
				matIndex = len(p.usedMaterials) - 1
				p.referencedMaterials[matName] = matIndex
			}

			// Activate material
			p.curMaterial = matIndex
		case "v":
			if len(lineTokens) < 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'v'; expected at least 3 arguments; got %d", len(lineTokens)-1)
			}

			v := types.Vec3{}
			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse vertex coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}
			p.vertexList = append(p.vertexList, v)
		case "vt":
			if len(lineTokens) < 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'vt'; expected at least 2 arguments; got %d", len(lineTokens)-1)
			}

			v := types.Vec2{}
			for tokIdx := 1; tokIdx <= 2; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse vertex tex coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}
			p.uvList = append(p.uvList, v)
		case "f":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'f'; expected 3 arguments for triangular face; got %d. Select the triangulation option in your exporter.", len(lineTokens)-1)
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
					return p.emitError(filename, lineNum, "expected each face argument to contain %d indices; arg %d contains %d indices", expIndices, arg, len(vTokens))
				}

				if vTokens[0] == "" {
					return p.emitError(filename, lineNum, "face argument %d does not include a vertex index", arg)
				}
				index, err := strconv.ParseInt(vTokens[0], 10, 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse vertex index for face argument %d: %s", arg, err.Error())
				}

				// If index is negative it refers to the end of the vertex list
				var vOffset int = 0
				if index < 0 {
					vOffset = len(p.vertexList) - int(index)
				} else {
					vOffset = int(index - 1)
				}
				if vOffset < 0 || vOffset >= len(p.vertexList) {
					return p.emitError(filename, lineNum, "vertex index out of bounds for face argument %d", arg)
				}
				vertices[arg] = p.vertexList[vOffset]

				// Parse UV coords if specified
				if expIndices == 1 || vTokens[1] == "" {
					continue
				}

				index, err = strconv.ParseInt(vTokens[1], 10, 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse vertex tex index for face argument %d: %s", arg, err.Error())
				}

				// If index is negative it refers to the end of the uv list
				if index < 0 {
					vOffset = len(p.uvList) - int(index)
				} else {
					vOffset = int(index - 1)
				}
				if vOffset < 0 || vOffset >= len(p.uvList) {
					return p.emitError(filename, lineNum, "vertex tex index out of bounds for face argument %d", arg)
				}
				uv[arg] = p.uvList[vOffset]
			}

			// If no material defined select the default
			if p.curMaterial < 0 {
				p.curMaterial = p.defaultMaterialIndex()
			}

			// Create primitive and assign material
			prim := scene.NewTriangle(vertices, uv)
			prim.Properties[1] = float32(p.curMaterial)

			// Calc bbox
			bboxMin := types.MinVec3(types.MinVec3(vertices[0], vertices[1]), vertices[2])
			bboxMax := types.MaxVec3(types.MaxVec3(vertices[0], vertices[1]), vertices[2])

			// Append wrapped primitive
			p.primitives = append(p.primitives, &scene.BvhPrimitive{
				Min:       bboxMin,
				Max:       bboxMax,
				Center:    prim.Center.Vec3(),
				Primitive: prim,
			})
		case "plane":
			if len(lineTokens) != 5 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'plane'; expected 4 arguments (x, y, z, D); got %d", len(lineTokens)-1)
			}

			v := types.Vec4{}
			for tokIdx := 1; tokIdx <= 4; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse plane coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}

			// If no material defined select the default
			if p.curMaterial < 0 {
				p.curMaterial = p.defaultMaterialIndex()
			}

			// Create primitive and assign material
			prim := scene.NewPlane(v.Vec3(), v[3])
			prim.Properties[1] = float32(p.curMaterial)

			// Calc bbox
			bboxMin := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
			bboxMax := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}

			// Append wrapped primitive
			p.primitives = append(p.primitives, &scene.BvhPrimitive{
				Min:       bboxMin,
				Max:       bboxMax,
				Center:    prim.Center.Vec3(),
				Primitive: prim,
			})
		case "sphere":
			if len(lineTokens) != 5 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'sphere'; expected 4 arguments (x, y, z, radius); got %d", len(lineTokens)-1)
			}

			v := types.Vec4{}
			for tokIdx := 1; tokIdx <= 4; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse sphere coordinate %d: %s", tokIdx-1, err.Error())
				}
				v[tokIdx-1] = float32(coord)
			}

			// If no material defined select the default
			if p.curMaterial < 0 {
				p.curMaterial = p.defaultMaterialIndex()
			}

			// Create primitive and assign material
			prim := scene.NewSphere(v.Vec3(), v[3])
			prim.Properties[1] = float32(p.curMaterial)

			// Calc bbox
			rvec := types.Vec3{v[3], v[3], v[3]}
			bboxMin := v.Vec3().Sub(rvec)
			bboxMax := v.Vec3().Add(rvec)

			// Append wrapped primitive
			p.primitives = append(p.primitives, &scene.BvhPrimitive{
				Min:       bboxMin,
				Max:       bboxMax,
				Center:    v.Vec3(),
				Primitive: prim,
			})
		case "box":
			if len(lineTokens) != 7 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'box'; expected 6 arguments (xmin, ymin, zmin, xmax, ymax, zmax); got %d", len(lineTokens)-1)
			}

			bmin := types.Vec3{}
			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse box min coordinate %d: %s", tokIdx-1, err.Error())
				}
				bmin[tokIdx-1] = float32(coord)
			}

			bmax := types.Vec3{}
			for tokIdx := 4; tokIdx <= 6; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse box max coordinate %d: %s", tokIdx-4, err.Error())
				}
				bmax[tokIdx-4] = float32(coord)
			}

			// If no material defined select the default
			if p.curMaterial < 0 {
				p.curMaterial = p.defaultMaterialIndex()
			}

			// Create primitive and assign material
			prim := scene.NewBox(bmin, bmax)
			prim.Properties[1] = float32(p.curMaterial)

			// Calc bbox
			bboxMin := types.MinVec3(bmin, bmax)
			bboxMax := types.MaxVec3(bmin, bmax)

			// Append wrapped primitive
			p.primitives = append(p.primitives, &scene.BvhPrimitive{
				Min:       bboxMin,
				Max:       bboxMax,
				Center:    bboxMin.Add(bboxMax).Mul(0.5),
				Primitive: prim,
			})
		case "camera_fov":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'camera_fov'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			v, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return p.emitError(filename, lineNum, "could not parse camera fov: %s", err.Error())
			}
			p.cameraFov = float32(v)
		case "camera_exposure":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'camera_exposure'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			v, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return p.emitError(filename, lineNum, "could not parse camera exposure: %s", err.Error())
			}
			p.cameraExposure = float32(v)
		case "camera_eye":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'camera_eye'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse camera eye coordinate %d: %s", tokIdx-1, err.Error())
				}
				p.cameraEye[tokIdx-1] = float32(coord)
			}
		case "camera_look":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'camera_look'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse camera look coordinate %d: %s", tokIdx-1, err.Error())
				}
				p.cameraLook[tokIdx-1] = float32(coord)
			}
		case "camera_up":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'camera_up'; expected 3 argument; got %d", len(lineTokens)-1)
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse camera up coordinate %d: %s", tokIdx-1, err.Error())
				}
				p.cameraUp[tokIdx-1] = float32(coord)
			}
		}
	}

	return nil
}

// Parse a wavefront material library.
func (p *textSceneReader) parseMaterials(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return p.emitError("", 0, "could not open %s", filename)
	}
	defer f.Close()

	// line stats
	var lineNum int = 0

	scanner := bufio.NewScanner(f)

	var curMaterial *scene.Material = nil
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
				return p.emitError(filename, lineNum, "unsupported syntax for 'newmtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			matName = lineTokens[1]
			if _, exists := p.availableMaterials[matName]; exists {
				return p.emitError(filename, lineNum, "material '%s' already defined", matName)
			}

			// Allocate new material and add it to library
			curMaterial = &scene.Material{}
			p.availableMaterials[matName] = curMaterial
		case "Kd":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'Kd'; expected 3 arguments; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return p.emitError(filename, lineNum, "got 'Kd' without a 'newmtl'")
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				val, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse diffuse component %d: %s", tokIdx-1, err.Error())
				}
				curMaterial.Diffuse[tokIdx-1] = float32(val)
			}
		case "Ke":
			if len(lineTokens) != 4 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'Ke'; expected 3 arguments; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return p.emitError(filename, lineNum, "got 'Ke' without a 'newmtl'")
			}

			for tokIdx := 1; tokIdx <= 3; tokIdx++ {
				val, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
				if err != nil {
					return p.emitError(filename, lineNum, "could not parse emissive component %d: %s", tokIdx-1, err.Error())
				}
				curMaterial.Emissive[tokIdx-1] = float32(val)
			}
		case "ior":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'ior'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return p.emitError(filename, lineNum, "got 'ior' without a 'newmtl'")
			}

			val, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return p.emitError(filename, lineNum, "could not parse IOR value: %s", err.Error())
			}
			curMaterial.Properties[1] = float32(val)
		case "roughness":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'roughness'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return p.emitError(filename, lineNum, "got 'roughness' without a 'newmtl'")
			}

			val, err := strconv.ParseFloat(lineTokens[1], 32)
			if err != nil {
				return p.emitError(filename, lineNum, "could not parse roughness value: %s", err.Error())
			}
			curMaterial.Properties[2] = float32(val)
		case "stype":
			if len(lineTokens) != 2 {
				return p.emitError(filename, lineNum, "unsupported syntax for 'stype'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			if curMaterial == nil {
				return p.emitError(filename, lineNum, "got 'stype' without a 'newmtl'")
			}

			switch lineTokens[1] {
			case "diffuse":
				curMaterial.Properties[0] = scene.DiffuseMaterial
			case "specular":
				curMaterial.Properties[0] = scene.SpecularMaterial
			case "refractive":
				curMaterial.Properties[0] = scene.RefractiveMaterial
			case "emissive":
				curMaterial.Properties[0] = scene.EmissiveMaterial
			default:
				return p.emitError(filename, lineNum, "unknown surface type '%s'", lineTokens[1])
			}
		}
	}

	return nil
}
