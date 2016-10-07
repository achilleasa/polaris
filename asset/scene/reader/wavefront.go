package reader

import (
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/achilleasa/polaris/asset"
	"github.com/achilleasa/polaris/asset/compiler"
	"github.com/achilleasa/polaris/asset/compiler/input"
	"github.com/achilleasa/polaris/asset/material"
	"github.com/achilleasa/polaris/asset/scene"
	"github.com/achilleasa/polaris/log"
	"github.com/achilleasa/polaris/types"
)

type wavefrontMaterial struct {
	Name string

	// Diffuse/Albedo color.
	Kd types.Vec3

	// Specular color.
	Ks types.Vec3

	// Emissive color and scaler.
	Ke       types.Vec3
	KeScaler float32

	// Transmission filter
	Tf types.Vec3

	// Index of refraction.
	Ni float32

	// Textures for modulating above parameters.
	KdTex     string
	KsTex     string
	KeTex     string
	TfTex     string
	BumpTex   string
	NormalTex string

	// Layered material expression.
	MaterialExpression string

	// Relative path for textures.
	AssetRelPath *asset.Resource

	// True if this material is used by at least one primitive.
	Used bool
}

// Generate material expression based on wavefront material properties.
func (wf *wavefrontMaterial) GetExpression() string {
	if wf.MaterialExpression != "" {
		return wf.MaterialExpression
	}

	isSpecularReflection := wf.Ks.MaxComponent() > 0.0 || wf.KsTex != ""
	isEmissive := wf.Ke.MaxComponent() > 0.0 || wf.KeTex != ""

	var bxdf material.BxdfType
	var exprArgs = make([]string, 0)
	switch {
	case isSpecularReflection && wf.Ni == 0.0:
		bxdf = material.BxdfConductor

		if wf.KsTex != "" {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %q", material.ParamSpecularity, wf.KsTex))
		} else if wf.Ks.MaxComponent() > 0.0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamSpecularity, wf.Ks))
		}
	case isSpecularReflection && wf.Ni != 0.0:
		bxdf = material.BxdfDielectric

		if wf.KsTex != "" {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %q", material.ParamSpecularity, wf.KsTex))
		} else if wf.Ks.MaxComponent() > 0.0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamSpecularity, wf.Ks))
		}

		if wf.TfTex != "" {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %q", material.ParamTransmittance, wf.TfTex))
		} else if wf.Tf.MaxComponent() > 0.0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamTransmittance, wf.Tf))
		}

		exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamIntIOR, wf.Ni))
	case isEmissive:
		bxdf = material.BxdfEmissive

		if wf.KeTex != "" {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %q", material.ParamRadiance, wf.KeTex))
		} else if wf.Ke.MaxComponent() > 0.0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamRadiance, wf.Ke))
		}

		if wf.KeScaler != 0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamScale, wf.KeScaler))
		}
	default:
		bxdf = material.BxdfDiffuse

		if wf.KdTex != "" {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %q", material.ParamReflectance, wf.KdTex))
		} else if wf.Kd.MaxComponent() > 0.0 {
			exprArgs = append(exprArgs, fmt.Sprintf("%s: %v", material.ParamReflectance, wf.Kd))
		}
	}

	materialExpr := bxdf.String() + "(" + strings.Join(exprArgs, ", ") + ")"

	// Apply bump map modifier (prefer normal maps to bump maps)
	if wf.NormalTex != "" {
		materialExpr = fmt.Sprintf("normalMap(%s, %q)", materialExpr, wf.NormalTex)
	} else if wf.BumpTex != "" {
		materialExpr = fmt.Sprintf("bumpMap(%s, %q)", materialExpr, wf.BumpTex)
	}

	return materialExpr
}

type wavefrontSceneReader struct {
	logger log.Logger

	// The parsed scene.
	rawScene *input.Scene

	// A map of material names to parsed wavefront materials
	matNameToIndex map[string]int

	// Currently selected material.
	curMaterial *wavefrontMaterial

	// Parsed wavefront materials.
	materials []*wavefrontMaterial

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
		logger:         log.New("wavefront scene reader"),
		rawScene:       input.NewScene(),
		matNameToIndex: make(map[string]int, 0),
		vertexList:     make([]types.Vec3, 0),
		normalList:     make([]types.Vec3, 0),
		uvList:         make([]types.Vec2, 0),
		errStack:       make([]string, 0),
	}
}

// Read scene definition.
func (r *wavefrontSceneReader) Read(sceneRes *asset.Resource) (*scene.Scene, error) {
	r.logger.Noticef(`parsing scene from "%s"`, sceneRes.Path())
	start := time.Now()

	// Parse scene
	err := r.parse(sceneRes)
	if err != nil {
		return nil, err
	}

	// If no mesh instances are defined, create instances for each defined mesh
	if len(r.rawScene.MeshInstances) == 0 {
		r.createDefaultMeshInstances()
	}

	// Prune unused materials
	r.processMaterials()

	r.logger.Noticef("parsed scene in %d ms", time.Since(start).Nanoseconds()/1e6)

	// Compile scene into an optimized, gpu-friendly format
	return compiler.Compile(r.rawScene)
}

// Generate scene materials for material entries that are in use and update the
// material indices for all parsed primitives.
func (r *wavefrontSceneReader) processMaterials() {
	wfMaterialToSceneMaterial := make(map[int]int, 0)
	prunedMaterials := make([]*input.Material, 0)
	pruned := 0
	for wfIndex, wfMat := range r.materials {
		// Whitelist scene materials
		if wfMat.Name == compiler.SceneDiffuseMaterialName || wfMat.Name == compiler.SceneEmissiveMaterialName {
			wfMat.Used = true
		}

		// Prune unused materials
		if !wfMat.Used {
			r.logger.Infof("skipping unused material %q", wfMat.Name)
			prunedMaterials = append(
				prunedMaterials,
				&input.Material{
					Name:         wfMat.Name,
					Expression:   wfMat.GetExpression(),
					AssetRelPath: wfMat.AssetRelPath,
				},
			)
			pruned++
			continue
		}

		r.rawScene.Materials = append(
			r.rawScene.Materials,
			&input.Material{
				Name:         wfMat.Name,
				Expression:   wfMat.GetExpression(),
				AssetRelPath: wfMat.AssetRelPath,
				Used:         true,
			},
		)

		wfMaterialToSceneMaterial[wfIndex] = len(r.rawScene.Materials) - 1
	}

	// For each primitive, map wavefront material indices to the generated materials
	for _, mesh := range r.rawScene.Meshes {
		for _, prim := range mesh.Primitives {
			prim.MaterialIndex = wfMaterialToSceneMaterial[prim.MaterialIndex]
		}
	}

	// Append pruned materials at the end of the list as they may be
	// referenced by material expressions.
	r.rawScene.Materials = append(r.rawScene.Materials, prunedMaterials...)

	if pruned > 0 {
		r.logger.Noticef("pruned %d unused materials", pruned)
	}
}

// Generate a mesh instance with an identity transformation for each defined mesh.
func (r *wavefrontSceneReader) createDefaultMeshInstances() {
	for meshIndex, mesh := range r.rawScene.Meshes {
		bbox := mesh.BBox()
		inst := &input.MeshInstance{
			MeshIndex: uint32(meshIndex),
			Transform: types.Ident4(),
		}
		inst.SetBBox(bbox)
		inst.SetCenter(bbox[0].Add(bbox[1]).Mul(0.5))
		r.rawScene.MeshInstances = append(r.rawScene.MeshInstances, inst)
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
func (r *wavefrontSceneReader) defaultMaterial() *wavefrontMaterial {
	matName := ""

	// Search for material in referenced list
	matIndex, exists := r.matNameToIndex[matName]
	if !exists {
		// Add it now
		r.materials = append(r.materials, &wavefrontMaterial{Kd: types.Vec3{0.7, 0.7, 0.7}})
		matIndex = len(r.materials) - 1
		r.matNameToIndex[matName] = matIndex
	}
	r.curMaterial = r.materials[matIndex]
	return r.curMaterial
}

// Parse wavefront object scene format.
func (r *wavefrontSceneReader) parse(res *asset.Resource) error {
	var lineNum int = 0
	var err error

	// The main obj file may include (call) several other object files. Each
	// object file contains 1-based indices (when they are positive). By
	// tracking the current vertex/uv/normal offsets we can apply them
	// while parsing faces to select the correct coordinates.
	relVertexOffset := len(r.vertexList)
	relUvOffset := len(r.uvList)
	relNormalOffset := len(r.normalList)

	scanner := bufio.NewScanner(res)
	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 || strings.HasPrefix(lineTokens[0], "#") {
			continue
		}

		switch lineTokens[0] {
		case "call", "mtllib":
			if len(lineTokens) != 2 {
				return r.emitError(res.Path(), lineNum, `unsupported syntax for "%s"; expected 1 argument; got %d`, lineTokens[0], len(lineTokens)-1)
			}

			r.pushFrame(fmt.Sprintf("referenced from %s:%d [%s]", res.Path(), lineNum, lineTokens[0]))

			incRes, err := asset.NewResource(lineTokens[1], res)
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
				return r.emitError(res.Path(), lineNum, `unsupported syntax for 'usemtl'; expected 1 argument; got %d`, len(lineTokens)-1)
			}

			// Lookup material
			matName := lineTokens[1]
			matIndex, exists := r.matNameToIndex[matName]
			if !exists {
				return r.emitError(res.Path(), lineNum, `undefined material with name "%s"`, matName)
			}

			// Activate material
			r.curMaterial = r.materials[matIndex]
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
				return r.emitError(res.Path(), lineNum, `unsupported syntax for "%s"; expected 1 argument for object name; got %d`, lineTokens[0], len(lineTokens)-1)
			}

			r.verifyLastParsedMesh()
			r.rawScene.Meshes = append(r.rawScene.Meshes, input.NewMesh(lineTokens[1]))
		case "f":
			primList, err := r.parseFace(lineTokens, relVertexOffset, relUvOffset, relNormalOffset)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}

			// If no object has been defined create a default one
			if len(r.rawScene.Meshes) == 0 {
				r.rawScene.Meshes = append(r.rawScene.Meshes, input.NewMesh("default"))
			}

			// Append primitive
			meshIndex := len(r.rawScene.Meshes) - 1
			r.rawScene.Meshes[meshIndex].MarkBBoxDirty()
			r.rawScene.Meshes[meshIndex].Primitives = append(r.rawScene.Meshes[meshIndex].Primitives, primList...)
		case "camera_fov":
			r.rawScene.Camera.FOV, err = parseFloat32(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_eye":
			r.rawScene.Camera.Eye, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_look":
			r.rawScene.Camera.Look, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "camera_up":
			r.rawScene.Camera.Up, err = parseVec3(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		case "instance":
			instance, err := r.parseMeshInstance(lineTokens)
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
			r.rawScene.MeshInstances = append(r.rawScene.MeshInstances, instance)
		}
	}

	r.verifyLastParsedMesh()
	return nil
}

// Drop the last parsed mesh if it contains no primitives.
func (r *wavefrontSceneReader) verifyLastParsedMesh() {
	lastMeshIndex := len(r.rawScene.Meshes) - 1
	if lastMeshIndex >= 0 && len(r.rawScene.Meshes[lastMeshIndex].Primitives) == 0 {
		r.logger.Warningf(`dropping mesh "%s" as it contains no polygons`, r.rawScene.Meshes[lastMeshIndex].Name)
		r.rawScene.Meshes = r.rawScene.Meshes[:lastMeshIndex]
	}
}

// Parse mesh instance definition. Definitions use the following format:
// instance mesh_name tX tY tZ yaw pitch roll sX sY sZ
// where:
// - tX, tY, tZ       : translation vector
// - yaw, pitch, roll : rotation angles in degrees
// - sX, sY, sZ	      : scale
func (r *wavefrontSceneReader) parseMeshInstance(lineTokens []string) (*input.MeshInstance, error) {
	if len(lineTokens) != 11 {
		return nil, fmt.Errorf(`unsupported syntax for "instance"; expected 10 arguments: mesh_name tX tY tZ yaw pitch roll sX sY sZ; got %d`, len(lineTokens)-1)
	}

	// Find object by name
	meshName := lineTokens[1]
	meshIndex := -1
	for index, mesh := range r.rawScene.Meshes {
		if mesh.Name == meshName {
			meshIndex = index
			break
		}
	}

	if meshIndex == -1 {
		return nil, fmt.Errorf(`unknown mesh with name "%s"`, meshName)
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
	meshBBox := r.rawScene.Meshes[meshIndex].BBox()
	min, max := transMat.Mul4x1(meshBBox[0].Vec4(1)).Vec3(), transMat.Mul4x1(meshBBox[1].Vec4(1)).Vec3()
	instBBox := [2]types.Vec3{
		types.MinVec3(min, max),
		types.MaxVec3(min, max),
	}
	inst := &input.MeshInstance{
		MeshIndex: uint32(meshIndex),
		Transform: scaleMat.Mul4(rotMat.Mul4(transMat)),
	}
	inst.SetBBox(instBBox)
	inst.SetCenter(instBBox[0].Add(instBBox[1]).Mul(0.5))

	return inst, nil
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
// This method only works with triangular/quad faces and will return an error if a
// face with more than 4 vertices is encountered.
func (r *wavefrontSceneReader) parseFace(lineTokens []string, relVertexOffset, relUvOffset, relNormalOffset int) ([]*input.Primitive, error) {
	if len(lineTokens) < 4 || len(lineTokens) > 5 {
		return nil, fmt.Errorf(`unsupported syntax for "f"; expected 3 arguments for triangular face or 4 arguments for a quad face; got %d. Select the triangulation option in your exporter`, len(lineTokens)-1)
	}

	var vertices [4]types.Vec3
	var normals [4]types.Vec3
	var uv [4]types.Vec2
	var vOffset int
	var err error
	expIndices := 0
	hasNormals := false
	for arg := 0; arg < len(lineTokens)-1; arg++ {
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

		vOffset, err = selectFaceCoordIndex(vTokens[0], len(r.vertexList), relVertexOffset)
		if err != nil {
			return nil, fmt.Errorf("could not parse vertex coord for face argument %d: %s", arg, err.Error())
		}
		vertices[arg] = r.vertexList[vOffset]

		// Parse UV coords if specified
		if expIndices > 1 && vTokens[1] != "" {
			vOffset, err = selectFaceCoordIndex(vTokens[1], len(r.uvList), relUvOffset)
			if err != nil {
				return nil, fmt.Errorf("could not parse tex coord for face argument %d: %s", arg, err.Error())
			}
			uv[arg] = r.uvList[vOffset]
		}

		// Parse normal coords if specified
		if expIndices > 2 && vTokens[2] != "" {
			vOffset, err = selectFaceCoordIndex(vTokens[2], len(r.normalList), relNormalOffset)
			if err != nil {
				return nil, fmt.Errorf("could not parse normal coord for face argument %d: %s", arg, err.Error())
			}
			normals[arg] = r.normalList[vOffset]
			hasNormals = true
		}
	}

	// If no material defined select the default. Also flag the current material
	// as being in use so we don't prune it later.
	if r.curMaterial == nil {
		r.curMaterial = r.defaultMaterial()
	}
	r.curMaterial.Used = true

	// If no normals are available generate them from the vertices
	if !hasNormals {
		e01 := vertices[1].Sub(vertices[0])
		e02 := vertices[2].Sub(vertices[0])
		faceNormal := e01.Cross(e02).Normalize()
		normals[0] = faceNormal
		normals[1] = faceNormal
		normals[2] = faceNormal
		normals[3] = faceNormal
	}

	// Assemble vertices into one or two primitives depending on whether we are parsing a triangular or a quad face
	primitives := make([]*input.Primitive, 0)
	indiceList := [][3]int{{0, 1, 2}}
	if len(lineTokens) == 5 {
		indiceList = append(indiceList, [3]int{0, 2, 3})
	}

	var triVerts [3]types.Vec3
	var triNormals [3]types.Vec3
	var triUVs [3]types.Vec2
	for _, indices := range indiceList {
		// copy vertices for this triangle
		for triIndex, selectIndex := range indices {
			triVerts[triIndex] = vertices[selectIndex]
			triNormals[triIndex] = normals[selectIndex]
			triUVs[triIndex] = uv[selectIndex]
		}

		prim := &input.Primitive{
			Vertices:      triVerts,
			Normals:       triNormals,
			UVs:           triUVs,
			MaterialIndex: r.matNameToIndex[r.curMaterial.Name],
		}
		prim.SetBBox(
			[2]types.Vec3{
				types.MinVec3(triVerts[0], types.MinVec3(triVerts[1], triVerts[2])),
				types.MaxVec3(triVerts[0], types.MaxVec3(triVerts[1], triVerts[2])),
			},
		)
		prim.SetCenter(triVerts[0].Add(triVerts[1]).Add(triVerts[2]).Mul(1.0 / 3.0))
		primitives = append(primitives, prim)
	}

	return primitives, nil
}

// Parse a wavefront material library.
func (r *wavefrontSceneReader) parseMaterials(res *asset.Resource) error {
	var lineNum int = 0
	var err error

	r.logger.Infof(`parsing material library "%s"`, res.Path())

	scanner := bufio.NewScanner(res)

	var curMaterial *wavefrontMaterial = nil
	var matName string = ""

	for scanner.Scan() {
		lineNum++
		lineTokens := strings.Fields(scanner.Text())
		if len(lineTokens) == 0 || strings.HasPrefix(lineTokens[0], "#") {
			continue
		}

		switch lineTokens[0] {
		case "newmtl":
			if len(lineTokens) != 2 {
				return r.emitError(res.Path(), lineNum, `unsupported syntax for "newmtl"; expected 1 argument; got %d`, len(lineTokens)-1)
			}

			matName = lineTokens[1]
			if _, exists := r.matNameToIndex[matName]; exists {
				return r.emitError(res.Path(), lineNum, `material "%s" already defined`, matName)
			}

			// Allocate new material and add it to library
			curMaterial = &wavefrontMaterial{
				Name:         matName,
				AssetRelPath: res,
			}
			r.materials = append(r.materials, curMaterial)
			r.matNameToIndex[matName] = len(r.materials) - 1
		default:
			if curMaterial == nil {
				return r.emitError(res.Path(), lineNum, `got "%s" without a "newmtl"`, lineTokens[0])
			}

			switch lineTokens[0] {
			case "include":
				if len(lineTokens) < 2 {
					return r.emitError(res.Path(), lineNum, `unsupported syntax for "%s"; expected 1 argument; got %d`, lineTokens[0], len(lineTokens)-1)
				}

				baseMaterialIndex, exists := r.matNameToIndex[lineTokens[1]]
				if !exists {
					return r.emitError(res.Path(), lineNum, `could not include unknown material "%s"`, lineTokens[1])
				}

				// Overwrite material but keep the original name
				*curMaterial = *r.materials[baseMaterialIndex]
				curMaterial.Name = matName
			case "Kd", "Ks", "Ke", "Tf":

				var target *types.Vec3
				switch lineTokens[0] {
				case "Kd":
					target = &curMaterial.Kd
				case "Ks":
					target = &curMaterial.Ks
				case "Ke":
					target = &curMaterial.Ke
				case "Tf":
					target = &curMaterial.Tf
				}

				*target, err = parseVec3(lineTokens)
			case "Ni":
				curMaterial.Ni, err = parseFloat32(lineTokens)
			case "map_Kd", "map_Ks", "map_Ke", "map_Tf", "map_bump", "map_normal":
				var target *string
				switch lineTokens[0] {
				case "map_Kd":
					target = &curMaterial.KdTex
				case "map_Ks":
					target = &curMaterial.KsTex
				case "map_Ke":
					target = &curMaterial.KeTex
				case "map_Tf":
					target = &curMaterial.TfTex
				case "map_bump":
					target = &curMaterial.BumpTex
				case "map_normal":
					target = &curMaterial.NormalTex
				}

				*target = lineTokens[1]
			case "mat_expr":
				if len(lineTokens) < 2 {
					return r.emitError(res.Path(), lineNum, `unsupported syntax for "%s"; expected 1 argument; got %d`, lineTokens[0], len(lineTokens)-1)
				}
				curMaterial.MaterialExpression = strings.Join(lineTokens[1:], " ")
			case "KeScaler":
				if len(lineTokens) < 2 {
					return r.emitError(res.Path(), lineNum, `unsupported syntax for "%s"; expected 1 argument; got %d`, lineTokens[0], len(lineTokens)-1)
				}
				curMaterial.KeScaler, err = parseFloat32(lineTokens)
			}

			// Report any errors
			if err != nil {
				return r.emitError(res.Path(), lineNum, err.Error())
			}
		}
	}

	return nil
}

// Given an index for a face coord type (vertex, normal, tex) calculate the
// proper offset into the coord list. Wavefront format can also use negative
// indices to reference elements from the end of the coord list.
func selectFaceCoordIndex(indexToken string, coordListLen int, relOffset int) (int, error) {
	index, err := strconv.ParseInt(indexToken, 10, 32)
	if err != nil {
		return -1, err
	}

	var vOffset int = 0
	if index < 0 {
		vOffset = coordListLen + int(index)
	} else {
		vOffset = relOffset + int(index-1)
	}
	if vOffset < 0 || vOffset >= coordListLen {
		return -1, fmt.Errorf("index out of bounds")
	}
	return vOffset, nil
}

// Parse a float scalar value.
func parseFloat32(lineTokens []string) (float32, error) {
	if len(lineTokens) < 2 {
		return 0, fmt.Errorf(`unsupported syntax for "%s"; expected 1 argument; got %d`, lineTokens[0], len(lineTokens)-1)
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
		return types.Vec3{}, fmt.Errorf(`unsupported syntax for "%s"; expected 3 arguments; got %d`, lineTokens[0], len(lineTokens)-1)
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
		return types.Vec2{}, fmt.Errorf(`unsupported syntax for "%s"; expected 2 arguments; got %d`, lineTokens[0], len(lineTokens)-1)
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
