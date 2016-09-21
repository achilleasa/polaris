package compiler

import (
	"fmt"
	"time"

	"github.com/achilleasa/go-pathtrace/asset"
	"github.com/achilleasa/go-pathtrace/asset/compiler/bvh"
	"github.com/achilleasa/go-pathtrace/asset/compiler/input"
	"github.com/achilleasa/go-pathtrace/asset/material"
	"github.com/achilleasa/go-pathtrace/asset/scene"
	"github.com/achilleasa/go-pathtrace/asset/texure"
	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	minPrimitivesPerLeaf      = 10
	SceneDiffuseMaterialName  = "scene_diffuse_material"
	SceneEmissiveMaterialName = "scene_emissive_material"
)

type sceneCompiler struct {
	parsedScene    *input.Scene
	optimizedScene *scene.Scene
	logger         log.Logger

	// A map of material indices to their layered material tree roots.
	matIndexToMatRoot map[int]int32

	// A map of a texture path to its index. This cache allows us to
	// re-use already loaded textures when referenced by multiple materials.
	texIndexCache map[string]int32

	// A map of material indices to an emissive layered material tree node.
	emissiveIndexCache map[int]int32
}

// Compile a scene representation parsed by a scene reader into a GPU-friendly
// optimized scene format.
func Compile(parsedScene *input.Scene) (*scene.Scene, error) {
	compiler := &sceneCompiler{
		parsedScene: parsedScene,
		optimizedScene: &scene.Scene{
			SceneDiffuseMatIndex:  -1,
			SceneEmissiveMatIndex: -1,
		},
		logger: log.New("scene compiler"),
	}

	start := time.Now()
	compiler.logger.Noticef("compiling scene")

	var err error
	err = compiler.createLayeredMaterialTrees()
	if err != nil {
		return nil, err
	}

	err = compiler.partitionGeometry()
	if err != nil {
		return nil, err
	}

	err = compiler.setupCamera()
	if err != nil {
		return nil, err
	}

	compiler.logger.Noticef("compiled scene in %d ms", time.Since(start).Nanoseconds()/1e6)
	return compiler.optimizedScene, nil
}

// Generate a two-level BVH tree for the scene. The top level BVH tree partitions
// the mesh instances. An additional BVH tree is also generated for each
// defined scene mesh. Each mesh instance points to the root BVH node of a mesh.
func (sc *sceneCompiler) partitionGeometry() error {
	start := time.Now()
	sc.logger.Notice("partitioning geometry")

	// Partition mesh instances so that each instance ends up in its own BVH leaf.
	sc.logger.Infof("building scene BVH tree (%d meshes, %d mesh instances)", len(sc.parsedScene.Meshes), len(sc.parsedScene.MeshInstances))
	volList := make([]bvh.BoundedVolume, len(sc.parsedScene.MeshInstances))
	for index, mi := range sc.parsedScene.MeshInstances {
		volList[index] = mi
	}
	sc.optimizedScene.BvhNodeList = bvh.Build(volList, 1, func(node *scene.BvhNode, workList []bvh.BoundedVolume) {
		pmi := workList[0].(*input.MeshInstance)

		// Assign mesh instance index to node
		for index, mi := range sc.parsedScene.MeshInstances {
			if pmi == mi {
				node.SetMeshIndex(uint32(index))
				break
			}
		}
	}, bvh.SurfaceAreaHeuristic)

	// Scan all meshes and calculate the size of material, vertex, normal
	// and uv lists; then pre-allocate them.
	totalVertices := 0
	for _, pm := range sc.parsedScene.Meshes {
		totalVertices += 3 * len(pm.Primitives)
	}

	sc.optimizedScene.VertexList = make([]types.Vec4, totalVertices)
	sc.optimizedScene.NormalList = make([]types.Vec4, totalVertices)
	sc.optimizedScene.UvList = make([]types.Vec2, totalVertices)
	sc.optimizedScene.MaterialIndex = make([]uint32, totalVertices/3)

	// Partition each mesh into its own BVH. Update all instances to point to this mesh BVH.
	var vertexOffset uint32 = 0
	var primOffset uint32 = 0
	meshBvhRoots := make([]uint32, len(sc.parsedScene.Meshes))
	meshEmissivePrimitives := make([]*scene.EmissivePrimitive, 0)
	emissiveIndexToMeshIndexMap := make(map[int]uint32, 0)
	for mIndex, pm := range sc.parsedScene.Meshes {
		volList := make([]bvh.BoundedVolume, len(pm.Primitives))
		for index, prim := range pm.Primitives {
			volList[index] = prim
		}

		sc.logger.Infof(`building BVH tree for "%s" (%d primitives)`, pm.Name, len(pm.Primitives))
		bvhNodes := bvh.Build(volList, minPrimitivesPerLeaf, func(node *scene.BvhNode, workList []bvh.BoundedVolume) {
			node.SetPrimitives(primOffset, uint32(len(workList)))

			// Copy primitive data to flat arrays
			for _, workItem := range workList {
				prim := workItem.(*input.Primitive)

				// Convert Vec3 to Vec4 which is required for proper alignment inside opencl kernels
				sc.optimizedScene.VertexList[vertexOffset+0] = prim.Vertices[0].Vec4(0)
				sc.optimizedScene.VertexList[vertexOffset+1] = prim.Vertices[1].Vec4(0)
				sc.optimizedScene.VertexList[vertexOffset+2] = prim.Vertices[2].Vec4(0)

				sc.optimizedScene.NormalList[vertexOffset+0] = prim.Normals[0].Vec4(0)
				sc.optimizedScene.NormalList[vertexOffset+1] = prim.Normals[1].Vec4(0)
				sc.optimizedScene.NormalList[vertexOffset+2] = prim.Normals[2].Vec4(0)

				sc.optimizedScene.UvList[vertexOffset+0] = prim.UVs[0]
				sc.optimizedScene.UvList[vertexOffset+1] = prim.UVs[1]
				sc.optimizedScene.UvList[vertexOffset+2] = prim.UVs[2]

				// Lookup root material node for primitive material index
				matNodeIndex := sc.matIndexToMatRoot[prim.MaterialIndex]
				sc.optimizedScene.MaterialIndex[primOffset] = uint32(matNodeIndex)

				// Check if this an emissive primitive and keep track of it
				// Since we may use multiple instances of this mesh we need a
				// separate pass to generate a primitive for each mesh instance
				if emissiveNodeIndex := sc.emissiveIndexCache[prim.MaterialIndex]; emissiveNodeIndex != -1 {
					meshEmissivePrimitives = append(meshEmissivePrimitives, &scene.EmissivePrimitive{
						// area = 0.5 * len(cross(v2-v0, v2-v1))
						Area:              0.5 * prim.Vertices[2].Sub(prim.Vertices[0]).Cross(prim.Vertices[2].Sub(prim.Vertices[1])).Len(),
						PrimitiveIndex:    primOffset,
						MaterialNodeIndex: uint32(emissiveNodeIndex),
						Type:              scene.AreaLight,
					})

					emissiveIndexToMeshIndexMap[len(meshEmissivePrimitives)-1] = uint32(mIndex)
				}

				vertexOffset += 3
				primOffset++
			}
		}, bvh.SurfaceAreaHeuristic)

		// Apply offset to bvh nodes and append them to the scene bvh list
		offset := int32(len(sc.optimizedScene.BvhNodeList))
		meshBvhRoots[mIndex] = uint32(offset)
		for index, _ := range bvhNodes {
			bvhNodes[index].OffsetChildNodes(offset)
		}
		sc.optimizedScene.BvhNodeList = append(sc.optimizedScene.BvhNodeList, bvhNodes...)
	}

	sc.logger.Infof("processing %d mesh instances", len(sc.parsedScene.MeshInstances))

	// Process each mesh instance
	sc.optimizedScene.MeshInstanceList = make([]scene.MeshInstance, len(sc.parsedScene.MeshInstances))
	for index, pmi := range sc.parsedScene.MeshInstances {
		mi := &sc.optimizedScene.MeshInstanceList[index]
		mi.MeshIndex = pmi.MeshIndex
		mi.BvhRoot = meshBvhRoots[pmi.MeshIndex]

		// We need to invert the transformation matrix when performing ray traversal
		mi.Transform = pmi.Transform.Inv()
	}

	sc.logger.Info("creating emissive primitive copies for mesh instances")

	// For each unique emissive primitive for the scene's meshes we need to
	// create a clone for each one of the mesh instances and fill in the
	// appropriate transformation matrix.
	sc.optimizedScene.EmissivePrimitives = make([]scene.EmissivePrimitive, 0)
	for _, mi := range sc.optimizedScene.MeshInstanceList {
		for emissiveIndex, meshIndex := range emissiveIndexToMeshIndexMap {
			if mi.MeshIndex != meshIndex {
				continue
			}

			// Copy original primitive and setup transformation matrix
			emp := *meshEmissivePrimitives[emissiveIndex]
			emp.Transform = mi.Transform
			sc.optimizedScene.EmissivePrimitives = append(sc.optimizedScene.EmissivePrimitives, emp)
		}
	}

	// If a global emission map is defined for the scene create an emissive for it
	if sc.optimizedScene.SceneEmissiveMatIndex != -1 && sc.emissiveIndexCache[int(sc.optimizedScene.SceneEmissiveMatIndex)] != -1 {
		emp := scene.EmissivePrimitive{
			MaterialNodeIndex: uint32(sc.emissiveIndexCache[int(sc.optimizedScene.SceneEmissiveMatIndex)]),
			Type:              scene.EnvironmentLight,
		}
		sc.optimizedScene.EmissivePrimitives = append(sc.optimizedScene.EmissivePrimitives, emp)
	}

	if len(sc.optimizedScene.EmissivePrimitives) > 0 {
		sc.logger.Infof("emitted %d emissive primitives for all mesh instances (%d unique mesh emissives)", len(sc.optimizedScene.EmissivePrimitives), len(meshEmissivePrimitives))
	} else {
		sc.logger.Warning("the scene contains no emissive primitives or a global environment light; output will appear black!")
	}

	sc.logger.Noticef("partitioned geometry in %d ms", time.Since(start).Nanoseconds()/1e6)

	return nil
}

// Initialize and position the camera for the scene.
func (sc *sceneCompiler) setupCamera() error {
	sc.optimizedScene.Camera = scene.NewCamera(sc.parsedScene.Camera.FOV)
	sc.optimizedScene.Camera.Position = sc.parsedScene.Camera.Eye
	sc.optimizedScene.Camera.LookAt = sc.parsedScene.Camera.Look
	sc.optimizedScene.Camera.Up = sc.parsedScene.Camera.Up

	return nil
}

// Perform a DFS in a layered material tree trying to locate anode with a particular BXDF.
func (sc *sceneCompiler) findMaterialNodeByBxdf(nodeIndex uint32, bxdf material.BxdfType) int32 {
	node := sc.optimizedScene.MaterialNodeList[nodeIndex]
	nodeType := uint32(node.Union1[0])

	// This is a bxdf node
	if material.IsBxdfType(nodeType) {
		if nodeType == uint32(bxdf) {
			return int32(nodeIndex)
		}
		return -1
	}

	// This is a op node. Scan left arg first
	out := sc.findMaterialNodeByBxdf(uint32(node.Union1[1]), bxdf)
	if out != -1 {
		return out
	}

	// If this is a mix node descend into the right child
	if nodeType == uint32(material.OpMix) {
		out = sc.findMaterialNodeByBxdf(uint32(node.Union1[2]), bxdf)
	}

	return out
}

// Parse material definitions into a node-based structure that models a layered material.
func (sc *sceneCompiler) createLayeredMaterialTrees() error {
	start := time.Now()
	sc.logger.Noticef("processing %d materials", len(sc.parsedScene.Materials))

	sc.matIndexToMatRoot = make(map[int]int32, 0)
	sc.texIndexCache = make(map[string]int32, 0)
	sc.emissiveIndexCache = make(map[int]int32, 0)
	sc.optimizedScene.MaterialNodeList = make([]scene.MaterialNode, 0)
	sc.optimizedScene.TextureData = make([]byte, 0)
	sc.optimizedScene.TextureMetadata = make([]scene.TextureMetadata, 0)

	for matIndex, mat := range sc.parsedScene.Materials {
		sc.logger.Infof(`processing material "%s"`, mat.Name)

		// Parse expression and perform semantic validation
		exprNode, err := material.ParseExpression(mat.Expression)
		if err != nil {
			return fmt.Errorf("material %q: %v", mat.Name, err)
		}
		err = exprNode.Validate()
		if err != nil {
			return fmt.Errorf("material %q: %v", mat.Name, err)
		}

		// Create material node tree and store its root index
		sc.matIndexToMatRoot[matIndex], err = sc.generateMaterialTree(mat, exprNode)
		if err != nil {
			return err
		}

		sc.emissiveIndexCache[matIndex] = sc.findMaterialNodeByBxdf(uint32(sc.matIndexToMatRoot[matIndex]), material.BxdfEmissive)

		if mat.Name == SceneDiffuseMaterialName {
			sc.optimizedScene.SceneDiffuseMatIndex = sc.matIndexToMatRoot[matIndex]
		} else if mat.Name == SceneEmissiveMaterialName {
			sc.optimizedScene.SceneEmissiveMatIndex = sc.matIndexToMatRoot[matIndex]
		}
	}

	sc.logger.Noticef("processed %d materials in %d ms", len(sc.parsedScene.Materials), time.Since(start).Nanoseconds()/1e6)
	return nil
}

// Recursively construct an optimized material node tree from the given expression.
// Returns the index of the tree root in the scene's material node list slice.
func (sc *sceneCompiler) generateMaterialTree(mat *input.Material, exprNode material.ExprNode) (int32, error) {
	var err error

	// Initialize material node; set all texture entries to -1
	node := scene.MaterialNode{
		Union1: [4]int32{0, -1, -1, -1},
		Union5: [1]int32{-1},
		// Default IORs
		Union4: types.Vec3{material.DefaultIntIOR, material.DefaultExtIOR, 0.0},
	}

	switch t := exprNode.(type) {
	case material.BxdfNode:
		node.Union1[0] = int32(t.Type)

		// Set bxdf defaults
		switch t.Type {
		case material.BxdfDiffuse:
			// Default reflectance
			node.Union2 = material.DefaultReflectance
		case material.BxdfConductor:
			// Default specularity
			node.Union2 = material.DefaultSpecularity
		case material.BxdfDielectric:
			// Default specularity and transmittance
			node.Union2 = material.DefaultSpecularity
			node.Union3 = material.DefaultTransmittance
		case material.BxdfRoughtConductor:
			// Default specularity and roughness
			node.Union2 = material.DefaultSpecularity
			node.Union4[2] = material.DefaultRoughness
		case material.BxdfRoughDielectric:
			// Default specularity, transmittance and roughness
			node.Union2 = material.DefaultSpecularity
			node.Union3 = material.DefaultTransmittance
			node.Union4[2] = material.DefaultRoughness
		case material.BxdfEmissive:
			// Default radiance and scaler
			node.Union2 = material.DefaultRadiance
			node.Union4[2] = material.DefaultRadianceScaler
		}

		// Apply parameters
		for _, paramNode := range t.Parameters {
			err = sc.setMaterialNodeParameter(mat, &node, paramNode)
			if err != nil {
				return -1, err
			}
		}
	case material.MixNode:
		node.Union1[0] = int32(material.OpMix)
		node.Union1[1], err = sc.generateMaterialTree(mat, t.Expressions[0])
		if err != nil {
			return -1, err
		}
		node.Union1[2], err = sc.generateMaterialTree(mat, t.Expressions[1])
		if err != nil {
			return -1, err
		}

		node.Union2[0] = t.Weights[0]
		node.Union2[1] = t.Weights[1]
	case material.MixMapNode:
		node.Union1[0] = int32(material.OpMixMap)
		node.Union1[1], err = sc.generateMaterialTree(mat, t.Expressions[0])
		if err != nil {
			return -1, err
		}
		node.Union1[2], err = sc.generateMaterialTree(mat, t.Expressions[1])
		if err != nil {
			return -1, err
		}

		node.Union1[3], err = sc.bakeTexture(mat, t.Texture)
		if err != nil {
			return -1, err
		}
	case material.BumpMapNode:
		node.Union1[0] = int32(material.OpBumpMap)
		node.Union1[1], err = sc.generateMaterialTree(mat, t.Expression)
		if err != nil {
			return -1, err
		}

		node.Union1[3], err = sc.bakeTexture(mat, t.Texture)
		if err != nil {
			return -1, err
		}
	case material.NormalMapNode:
		node.Union1[0] = int32(material.OpNormalMap)
		node.Union1[1], err = sc.generateMaterialTree(mat, t.Expression)
		if err != nil {
			return -1, err
		}

		node.Union1[3], err = sc.bakeTexture(mat, t.Texture)
		if err != nil {
			return -1, err
		}
	default:
		return -1, fmt.Errorf("%q: unsupported node %#+v\n", mat.Name, exprNode)
	}

	sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
	return int32(len(sc.optimizedScene.MaterialNodeList) - 1), nil
}

func (sc *sceneCompiler) setMaterialNodeParameter(mat *input.Material, node *scene.MaterialNode, param material.BxdfParamNode) error {
	var err error
	switch param.Name {
	case material.ParamReflectance, material.ParamSpecularity, material.ParamRadiance:
		switch t := param.Value.(type) {
		case material.Vec3Node:
			node.Union2 = types.Vec3(t).Vec4(0.0)
		case material.TextureNode:
			node.Union1[3], err = sc.bakeTexture(mat, t)
		}
	case material.ParamTransmittance:
		switch t := param.Value.(type) {
		case material.Vec3Node:
			node.Union3 = types.Vec3(t).Vec4(0.0)
		case material.TextureNode:
			node.Union1[2], err = sc.bakeTexture(mat, t)
		}
	case material.ParamIntIOR, material.ParamExtIOR:
		index := 0
		if param.Name == material.ParamExtIOR {
			index = 1
		}

		switch t := param.Value.(type) {
		case material.FloatNode:
			node.Union4[index] = float32(t)
		case material.MaterialNameNode:
			node.Union4[index], err = material.IOR(t)
		}
	case material.ParamScale:
		node.Union4[2] = float32(param.Value.(material.FloatNode))
	case material.ParamRoughness:
		switch t := param.Value.(type) {
		case material.FloatNode:
			node.Union4[2] = float32(t)
		case material.TextureNode:
			node.Union5[0], err = sc.bakeTexture(mat, t)
		}
	}

	return err
}

// Load a texture resource and store its metadata/data into the optimized scene.
// Texture data is always aligned on a dword boundary.
func (sc *sceneCompiler) bakeTexture(mat *input.Material, texNode material.TextureNode) (int32, error) {
	texPath := string(texNode)
	res, err := asset.NewResource(texPath, mat.AssetRelPath)
	if err != nil {
		sc.logger.Warningf("%q: skipping missing texture %q", mat.Name, texPath)
		return -1, nil
	}

	// Check if texture is already loaded
	if texIndex, exists := sc.texIndexCache[res.Path()]; exists {
		sc.logger.Infof("%q: re-using already loaded texture %q", mat.Name, texPath)
		return texIndex, nil
	}

	sc.logger.Infof("%q: processing texture %q", mat.Name, texPath)

	tex, err := texture.New(res)
	if err != nil {
		return -1, fmt.Errorf("%q: %v", mat.Name, err)
	}

	dataOffset := len(sc.optimizedScene.TextureData)
	realLen := len(tex.Data)
	alignedLen := align4(realLen)

	// Copy data and add alignment padding
	sc.optimizedScene.TextureData = append(sc.optimizedScene.TextureData, tex.Data...)
	if alignedLen > realLen {
		pad := make([]byte, alignedLen-realLen)
		sc.optimizedScene.TextureData = append(sc.optimizedScene.TextureData, pad...)
	}

	// Setup metadata
	sc.optimizedScene.TextureMetadata = append(
		sc.optimizedScene.TextureMetadata,
		scene.TextureMetadata{
			Format:     tex.Format,
			Width:      tex.Width,
			Height:     tex.Height,
			DataOffset: uint32(dataOffset),
		},
	)

	texIndex := int32(len(sc.optimizedScene.TextureMetadata) - 1)
	sc.texIndexCache[res.Path()] = texIndex
	return texIndex, nil
}

// Adjust value so its divisible by 4.
func align4(value int) int {
	for {
		if value%4 == 0 {
			return value
		}
		value++
	}
}
