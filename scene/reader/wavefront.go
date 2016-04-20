package reader

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	scenePkg "github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

type wavefrontSceneReader struct {
	logger *log.Logger

	// The parsed scene.
	sceneGraph *Scene

	// A map of material names to material index.
	matNameToIndex map[string]uint32

	// Currently selected material index
	curMaterial int32

	// List of vertices, normals and uv coords.
	vertexList []types.Vec3
	normalList []types.Vec3
	uvList     []types.Vec2

	// An error stack that provides additional error information when
	// scene files include other files (models, mat libs e.t.c)
	errStack []string
}

// Create a new text scene reader.
func newWavefrontReader() *wavefrontSceneReader {
	return &wavefrontSceneReader{
		logger:         log.New(os.Stdout, "wavefrontSceneReader: ", log.LstdFlags),
		sceneGraph:     newScene(),
		matNameToIndex: make(map[string]uint32, 0),
		curMaterial:    -1,
		vertexList:     make([]types.Vec3, 0),
		normalList:     make([]types.Vec3, 0),
		uvList:         make([]types.Vec2, 0),
		errStack:       make([]string, 0),
	}
}

// Read scene definition.
func (r *wavefrontSceneReader) Read(sceneRes *resource) (*scenePkg.Scene, error) {
	r.logger.Printf("parsing scene from %s", sceneRes.Path())
	start := time.Now()

	// Parse scene
	err := r.parse(sceneRes)
	if err != nil {
		return nil, err
	}

	// If no mesh instances are defined, create instances for each defined mesh
	if len(r.sceneGraph.MeshInstances) == 0 {
		r.createDefaultMeshInstances()
	}

	r.logger.Printf("parsed scene in %d ms", time.Since(start).Nanoseconds()/1000000)

	return nil, fmt.Errorf("scenegraph conversion not yet implemented")
}

// Generate a mesh instance with an identity transformation for each defined mesh.
func (r *wavefrontSceneReader) createDefaultMeshInstances() {
	for meshIndex, _ := range r.sceneGraph.Meshes {
		r.sceneGraph.MeshInstances = append(r.sceneGraph.MeshInstances, &MeshInstance{
			MeshIndex: uint32(meshIndex),
			Transform: types.Ident4(),
		})
	}
}

// Generate an error message that also includes any data in the error stack.
func (r *wavefrontSceneReader) emitError(file string, line int, msgFormat string, args ...interface{}) error {
	msg := fmt.Sprintf(msgFormat, args...)

	var errMsg string
	if file != "" {
		errMsg = strings.Trim(
			fmt.Sprintf("[%s: %d] error: %s\n%s", file, line, msg, strings.Join(r.errStack, "\n")),
			"\n",
		)
	} else {
		errMsg = strings.Trim(
			fmt.Sprintf("error: %s\n%s", msg, strings.Join(r.errStack, "\n")),
			"\n",
		)
	}

	return fmt.Errorf(errMsg)
}

// Push a frame to the error stack.
func (r *wavefrontSceneReader) pushFrame(msg string) {
	r.errStack = append([]string{msg}, r.errStack...)
}

// Pop a frame from the error stack.
func (r *wavefrontSceneReader) popFrame() {
	r.errStack = r.errStack[1:]
}

// Create and select a default material for surfaces not using one.
func (r *wavefrontSceneReader) defaultMaterial() int32 {
	matName := ""

	// Search for material in referenced list
	matIndex, exists := r.matNameToIndex[matName]
	if !exists {
		// Add it now
		r.sceneGraph.Materials = append(r.sceneGraph.Materials, &Material{Kd: types.Vec3{0.7, 0.7, 0.7}})
		matIndex = uint32(len(r.sceneGraph.Materials) - 1)
	}
	r.curMaterial = int32(matIndex)
	return r.curMaterial
}

// Parse wavefront object scene format.
func (r *wavefrontSceneReader) parse(res *resource) error {
	var lineNum int = 0
	var err error

	scanner := bufio.NewScanner(res)
	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 {
			continue
		}

		switch lineTokens[0] {
		case "#":
			continue
		case "call", "mtllib":
			if len(lineTokens) != 2 {
				return r.emitError(res.Path(), lineNum, "unsupported syntax for '%s'; expected 1 argument; got %d", lineTokens[0], len(lineTokens)-1)
			}

			r.pushFrame(fmt.Sprintf("referenced from %s:%d [%s]", res.Path(), lineNum, lineTokens[0]))

			incRes, err := newResource(lineTokens[1], res)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			defer incRes.Close()

			switch lineTokens[0] {
			case "call":
				err = r.parse(incRes)
			case "mtllib":
				err = r.parseMaterials(incRes)
			}

			if err != nil {
				return err
			}
			r.popFrame()
		case "usemtl":
			if len(lineTokens) != 2 {
				return r.emitError(res.Path(), lineNum, "unsupported syntax for 'usemtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			// Lookup material
			matName := lineTokens[1]
			matIndex, exists := r.matNameToIndex[matName]
			if !exists {
				return r.emitError(res.Path(), lineNum, "undefined material with name '%s'", matName)
			}

			// Activate material
			r.curMaterial = int32(matIndex)
		case "v":
			v, err := parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			r.vertexList = append(r.vertexList, v)
		case "vn":
			v, err := parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			r.normalList = append(r.normalList, v)
		case "vt":
			v, err := parseVec2(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			r.uvList = append(r.uvList, v)
		case "g", "o":
			if len(lineTokens) < 2 {
				return r.emitError(res.Path(), lineNum, "unsupported syntax for '%s'; expected 1 argument for object name; got %d", lineTokens[0], len(lineTokens)-1)
			}

			r.sceneGraph.Meshes = append(r.sceneGraph.Meshes, newMesh(lineTokens[1]))
		case "f":
			prim, err := r.parseFace(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}

			// If no object has been defined create a default one
			if len(r.sceneGraph.Meshes) == 0 {
				r.sceneGraph.Meshes = append(r.sceneGraph.Meshes, newMesh("default"))
			}

			// Append primitive
			meshIndex := len(r.sceneGraph.Meshes) - 1
			r.sceneGraph.Meshes[meshIndex].Primitives = append(r.sceneGraph.Meshes[meshIndex].Primitives, prim)
		case "camera_fov":
			r.sceneGraph.Camera.FOV, err = parseFloat32(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_eye":
			r.sceneGraph.Camera.Eye, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_look":
			r.sceneGraph.Camera.Look, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_up":
			r.sceneGraph.Camera.Up, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "instance":
			instance, err := r.parseMeshInstance(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			r.sceneGraph.MeshInstances = append(r.sceneGraph.MeshInstances, instance)
		}
	}

	return nil
}

// Parse mesh instance definition. Definitions use the following format:
// instance mesh_name tX tY tZ yaw pitch roll sX sY sZ
// where:
// - tX, tY, tZ       : translation vector
// - yaw, pitch, roll : rotation angles in degrees
// - sX, sY, sZ	      : scale
func (r *wavefrontSceneReader) parseMeshInstance(lineTokens []string) (*MeshInstance, error) {
	if len(lineTokens) != 11 {
		return nil, fmt.Errorf("unsupported syntax for 'instance'; expected 10 arguments: mesh_name tX tY tZ yaw pitch roll sX sY sZ; got %d", len(lineTokens)-1)
	}

	// Find object by name
	meshName := lineTokens[1]
	meshIndex := -1
	for index, mesh := range r.sceneGraph.Meshes {
		if mesh.Name == meshName {
			meshIndex = index
			break
		}
	}

	if meshIndex == -1 {
		return nil, fmt.Errorf("unknown mesh with name '%s'", meshName)
	}

	var translation, rotation, scale types.Vec3

	// Parse translation
	for index := 2; index < 5; index++ {
		v, err := strconv.ParseFloat(lineTokens[index], 32)
		if err != nil {
			return nil, err
		}
		translation[index-2] = float32(v)
	}

	// Parse rotation angles and convert to radians
	for index := 5; index < 8; index++ {
		v, err := strconv.ParseFloat(lineTokens[index], 32)
		if err != nil {
			return nil, err
		}
		v *= math.Pi / 180.0
		rotation[index-5] = float32(v)
	}

	// Parse scale
	for index := 8; index < 11; index++ {
		v, err := strconv.ParseFloat(lineTokens[index], 32)
		if err != nil {
			return nil, err
		}
		scale[index-8] = float32(v)
	}

	// Generate final matrix: M = T * R * S
	yawQuat := types.QuatFromAxisAngle(types.Vec3{1, 0, 0}, rotation[0])
	pitchQuat := types.QuatFromAxisAngle(types.Vec3{0, 1, 0}, rotation[1])
	rollQuat := types.QuatFromAxisAngle(types.Vec3{0, 0, 1}, rotation[2])
	rotMat := rollQuat.Mul(pitchQuat.Mul(yawQuat)).Normalize().Mat4()
	scaleMat := types.Scale4(scale)
	transMat := types.Translate4(translation)

	// Transform mesh bbox and recalculate a new AABB for the mesh instance
	meshBBox := r.sceneGraph.Meshes[meshIndex].BBox
	min, max := transMat.Mul4x1(meshBBox[0].Vec4(1)).Vec3(), transMat.Mul4x1(meshBBox[1].Vec4(1)).Vec3()

	return &MeshInstance{
		MeshIndex: uint32(meshIndex),
		Transform: scaleMat.Mul4(rotMat.Mul4(transMat)),
		BBox: [2]types.Vec3{
			types.MinVec3(min, max),
			types.MaxVec3(min, max),
		},
	}, nil
}

// Parse face definition. Each face definitions consists of 3 arguments,
// one for each vertex. Each one of the vertex arguments is comprised of
// 1, 2 or 3 args separated by a slash character. The following formats are
// supported:
// - vertexIndex
// - vertexIndex/uvIndex
// - vertexIndex//normalIndex
// - vertexIndex/uvIndex/normalIndex
//
// Indices start from 1 and may be negative to indicate
// an offset off the end of the vertex/uv list.
//
// This method only works with triangular faces and will return an error if a
// face with more than 3 vertices is encountered.
func (r *wavefrontSceneReader) parseFace(lineTokens []string) (*Primitive, error) {
	if len(lineTokens) != 4 {
		return nil, fmt.Errorf("unsupported syntax for 'f'; expected 3 arguments for triangular face; got %d. Select the triangulation option in your exporter.", len(lineTokens)-1)
	}

	var vertices [3]types.Vec3
	var normals [3]types.Vec3
	var uv [3]types.Vec2
	var vOffset int
	var err error
	expIndices := 0
	for arg := 0; arg < 3; arg++ {
		vTokens := strings.Split(lineTokens[arg+1], "/")

		// The first arg defines the format for the following args
		if arg == 0 {
			expIndices = len(vTokens)
		} else if len(vTokens) != expIndices {
			return nil, fmt.Errorf("expected each face argument to contain %d indices; arg %d contains %d indices", expIndices, arg, len(vTokens))
		}

		// Faces must at least define a vertex coord
		if vTokens[0] == "" {
			return nil, fmt.Errorf("face argument %d does not include a vertex index", arg)
		}

		vOffset, err = selectFaceCoordIndex(vTokens[0], len(r.vertexList))
		if err != nil {
			return nil, fmt.Errorf("could not parse vertex coord for face argument %d: %s", arg, err.Error())
		}
		vertices[arg] = r.vertexList[vOffset]

		// Parse UV coords if specified
		if vTokens[1] != "" {
			vOffset, err = selectFaceCoordIndex(vTokens[1], len(r.uvList))
			if err != nil {
				return nil, fmt.Errorf("could not parse tex coord for face argument %d: %s", arg, err.Error())
			}
			uv[arg] = r.uvList[vOffset]
		}

		// Parse normal coords if specified
		if vTokens[2] != "" {
			vOffset, err = selectFaceCoordIndex(vTokens[2], len(r.normalList))
			if err != nil {
				return nil, fmt.Errorf("could not parse normal coord for face argument %d: %s", arg, err.Error())
			}
			normals[arg] = r.normalList[vOffset]
		}
	}

	// If no material defined select the default
	if r.curMaterial < 0 {
		r.curMaterial = r.defaultMaterial()
	}

	return &Primitive{
		Vertices: vertices,
		Normals:  normals,
		UVs:      uv,
		BBox: [2]types.Vec3{
			types.MinVec3(vertices[0], types.MinVec3(vertices[1], vertices[2])),
			types.MaxVec3(vertices[0], types.MaxVec3(vertices[1], vertices[2])),
		},
		MaterialIndex: uint32(r.curMaterial),
	}, nil
}

// Parse a wavefront material library.
func (r *wavefrontSceneReader) parseMaterials(res *resource) error {
	var lineNum int = 0
	var err error

	scanner := bufio.NewScanner(res)

	var curMaterial *Material = nil
	var matName string = ""

	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 {
			continue
		}

		switch lineTokens[0] {
		case "#":
			continue
		case "newmtl":
			if len(lineTokens) != 2 {
				return r.emitError(res.Path(), lineNum, "unsupported syntax for 'newmtl'; expected 1 argument; got %d", len(lineTokens)-1)
			}

			matName = lineTokens[1]
			if _, exists := r.matNameToIndex[matName]; exists {
				return r.emitError(res.Path(), lineNum, "material '%s' already defined", matName)
			}

			// Allocate new material and add it to library
			curMaterial = newMaterial(matName)
			r.sceneGraph.Materials = append(r.sceneGraph.Materials, curMaterial)
			r.matNameToIndex[matName] = uint32(len(r.sceneGraph.Materials) - 1)
		default:
			if curMaterial == nil {
				return r.emitError(res.Path(), lineNum, "got '%s' without a 'newmtl'", lineTokens[0])
			}

			switch lineTokens[0] {
			case "Kd", "Ks", "Ke":

				var target *types.Vec3
				switch lineTokens[0] {
				case "Kd":
					target = &curMaterial.Kd
				case "Ks":
					target = &curMaterial.Ks
				case "Ke":
					target = &curMaterial.Ke
				}

				*target, err = parseVec3(lineTokens)
			case "Ni", "Nr":

				var target *float32
				switch lineTokens[0] {
				case "Ni":
					target = &curMaterial.Ni
				case "Nr":
					target = &curMaterial.Nr
				}

				*target, err = parseFloat32(lineTokens)
			case "map_Kd", "map_Ks", "map_Ke", "map_bump", "map_Ni", "map_Nr":
				var target *int32
				switch lineTokens[0] {
				case "map_Kd":
					target = &curMaterial.KdTex
				case "map_Ks":
					target = &curMaterial.KsTex
				case "map_Ke":
					target = &curMaterial.KeTex
				case "map_bump":
					target = &curMaterial.NormalTex
				case "map_Ni":
					target = &curMaterial.NiTex
				case "map_Nr":
					target = &curMaterial.NrTex
				}

				imgRes, err := newResource(lineTokens[1], res)
				if err != nil {
					// Ignore missing textures
					if strings.Contains(err.Error(), "no such file or directory") {
						r.logger.Printf("warning: ignoring missing texture %s", lineTokens[1])
						continue
					}

					return r.emitError(res.Path(), lineNum, err.Error())
				}

				*target, err = r.loadTexture(imgRes)
			}

			// Report any errors
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		}
	}

	return nil
}

// Load texture and return its index in the texture list.
func (r *wavefrontSceneReader) loadTexture(res *resource) (int32, error) {
	tex, err := newTexture(res)
	if err != nil {
		return -1, err
	}

	r.sceneGraph.Textures = append(r.sceneGraph.Textures, tex)
	return int32(len(r.sceneGraph.Textures) - 1), nil
}

// Given an index for a face coord type (vertex, normal, tex) calculate the
// proper offset into the coord list. Wavefront format can also use negative
// indices to reference elements from the end of the coord list.
func selectFaceCoordIndex(indexToken string, coordListLen int) (int, error) {
	index, err := strconv.ParseInt(indexToken, 10, 32)
	if err != nil {
		return -1, err
	}

	var vOffset int = 0
	if index < 0 {
		vOffset = coordListLen + int(index)
	} else {
		vOffset = int(index - 1)
	}
	if vOffset < 0 || vOffset >= coordListLen {
		return -1, fmt.Errorf("index out of bounds")
	}
	return vOffset, nil
}

// Parse a float scalar value.
func parseFloat32(lineTokens []string) (float32, error) {
	if len(lineTokens) < 2 {
		return 0, fmt.Errorf("unsupported syntax for '%s'; expected 1 argument; got %d", lineTokens[0], len(lineTokens)-1)
	}

	val, err := strconv.ParseFloat(lineTokens[1], 32)
	if err != nil {
		return 0, err
	}

	return float32(val), nil
}

// Parse a Vec3 row.
func parseVec3(lineTokens []string) (types.Vec3, error) {
	if len(lineTokens) < 4 {
		return types.Vec3{}, fmt.Errorf("unsupported syntax for '%s'; expected 3 arguments; got %d", lineTokens[0], len(lineTokens)-1)
	}

	v := types.Vec3{}
	for tokIdx := 1; tokIdx <= 3; tokIdx++ {
		coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
		if err != nil {
			return v, err
		}
		v[tokIdx-1] = float32(coord)
	}
	return v, nil
}

// Parse a Vec2 row.
func parseVec2(lineTokens []string) (types.Vec2, error) {
	if len(lineTokens) < 3 {
		return types.Vec2{}, fmt.Errorf("unsupported syntax for '%s'; expected 2 arguments; got %d", lineTokens[0], len(lineTokens)-1)
	}

	v := types.Vec2{}
	for tokIdx := 1; tokIdx <= 2; tokIdx++ {
		coord, err := strconv.ParseFloat(lineTokens[tokIdx], 32)
		if err != nil {
			return v, err
		}
		v[tokIdx-1] = float32(coord)
	}
	return v, nil
}
