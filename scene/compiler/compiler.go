package compiler

import (
	"fmt"
	"time"

	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	minPrimitivesPerLeaf      = 10
	sceneDiffuseMaterialName  = "scene_diffuse_material"
	sceneEmissiveMaterialName = "scene_emissive_material"
)

type sceneCompiler struct {
	parsedScene    *scene.ParsedScene
	optimizedScene *scene.Scene
	logger         log.Logger
}

// Compile a scene representation parsed by a scene reader into a GPU-friendly
// optimized scene format.
func Compile(parsedScene *scene.ParsedScene) (*scene.Scene, error) {
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
	err = compiler.bakeTextures()
	if err != nil {
		return nil, err
	}

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

// Allocate a contiguous memory block for all texture data and initialize the
// scene's texture metadata so that they point to the proper index inside the block.
func (sc *sceneCompiler) bakeTextures() error {
	start := time.Now()
	sc.logger.Noticef("processing %d textures", len(sc.parsedScene.Textures))

	// Find how much memory we need. To ensure proper memory alignment we pad
	// each texture's data len so its a multiple of a qword
	var totalDataLen uint32 = 0
	for _, tex := range sc.parsedScene.Textures {
		totalDataLen += align4(len(tex.Data))
	}

	sc.optimizedScene.TextureData = make([]byte, totalDataLen)
	sc.optimizedScene.TextureMetadata = make([]scene.TextureMetadata, len(sc.parsedScene.Textures))
	var offset uint32 = 0
	for index, tex := range sc.parsedScene.Textures {
		sc.optimizedScene.TextureMetadata[index].Format = tex.Format
		sc.optimizedScene.TextureMetadata[index].Width = tex.Width
		sc.optimizedScene.TextureMetadata[index].Height = tex.Height
		sc.optimizedScene.TextureMetadata[index].DataOffset = offset

		// Copy data
		copy(sc.optimizedScene.TextureData[offset:], tex.Data)
		offset += uint32(align4(len(tex.Data)))
	}

	sc.logger.Noticef("processed textures in %d ms", time.Since(start).Nanoseconds()/1e6)
	return nil
}

// Generate a two-level BVH tree for the scene. The top level BVH tree partitions
// the mesh instances. An additional BVH tree is also generated for each
// defined scene mesh. Each mesh instance points to the root BVH node of a mesh.
func (sc *sceneCompiler) partitionGeometry() error {
	start := time.Now()
	sc.logger.Notice("partitioning geometry")

	// Partition mesh instances so that each instance ends up in its own BVH leaf.
	sc.logger.Infof("building scene BVH tree (%d meshes, %d mesh instances)", len(sc.parsedScene.Meshes), len(sc.parsedScene.MeshInstances))
	volList := make([]BoundedVolume, len(sc.parsedScene.MeshInstances))
	for index, mi := range sc.parsedScene.MeshInstances {
		volList[index] = mi
	}
	sc.optimizedScene.BvhNodeList = BuildBVH(volList, 1, func(node *scene.BvhNode, workList []BoundedVolume) {
		pmi := workList[0].(*scene.ParsedMeshInstance)

		// Assign mesh instance index to node
		for index, mi := range sc.parsedScene.MeshInstances {
			if pmi == mi {
				node.SetMeshIndex(uint32(index))
				break
			}
		}
	})

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
		volList := make([]BoundedVolume, len(pm.Primitives))
		for index, prim := range pm.Primitives {
			volList[index] = prim
		}

		sc.logger.Infof(`building BVH tree for "%s" (%d primitives)`, pm.Name, len(pm.Primitives))
		bvhNodes := BuildBVH(volList, minPrimitivesPerLeaf, func(node *scene.BvhNode, workList []BoundedVolume) {
			node.SetPrimitives(primOffset, uint32(len(workList)))

			// Copy primitive data to flat arrays
			for _, workItem := range workList {
				prim := workItem.(*scene.ParsedPrimitive)

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
				matNodeIndex := sc.optimizedScene.MaterialNodeRoots[prim.MaterialIndex]
				sc.optimizedScene.MaterialIndex[primOffset] = matNodeIndex

				// Check if this an emissive primitive and keep track of it
				// Since we may use multiple instances of this mesh we need a
				// separate pass to generate a primitive for each mesh instance
				emissiveNodeIndex := sc.findMaterialNodeByBxdf(matNodeIndex, scene.Emissive)
				if emissiveNodeIndex != -1 {
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
		})

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
	if sc.optimizedScene.SceneEmissiveMatIndex != -1 {
		sceneEmissiveMat := sc.optimizedScene.MaterialNodeList[sc.optimizedScene.SceneEmissiveMatIndex]
		if sceneEmissiveMat.Kval.MaxComponent() > 0.0 || sceneEmissiveMat.GetKvalTex() != -1 {
			emp := scene.EmissivePrimitive{
				MaterialNodeIndex: uint32(sc.optimizedScene.SceneEmissiveMatIndex),
				Type:              scene.EnvironmentLight,
			}
			sc.optimizedScene.EmissivePrimitives = append(sc.optimizedScene.EmissivePrimitives, emp)
		}
	}

	if len(sc.optimizedScene.EmissivePrimitives) > 0 {
		sc.logger.Infof("emitted %d emissive primitives for all mesh instances (%d unique mesh emissives)", len(sc.optimizedScene.EmissivePrimitives), len(meshEmissivePrimitives))
	} else {
		sc.logger.Warning("the scene contains no emissive primitives or a global environment light; output will appear black!")
	}

	sc.logger.Noticef("partitioned geometry in %d ms", time.Since(start).Nanoseconds()/1e6)

	return nil
}

// Perform a DFS in a layered material tree trying to locate anode with a particular BXDF.
func (sc *sceneCompiler) findMaterialNodeByBxdf(nodeIndex uint32, bxdf scene.MatBxdfType) int32 {
	node := sc.optimizedScene.MaterialNodeList[nodeIndex]
	if node.IsNode == 0 {
		if node.GetBxdfType() == bxdf {
			return int32(nodeIndex)
		}
		return -1
	}

	out := sc.findMaterialNodeByBxdf(node.GetLeftIndex(), bxdf)
	if out != -1 {
		return out
	}

	return sc.findMaterialNodeByBxdf(node.GetRightIndex(), bxdf)
}

// Convert material definitions into a node-based structure that models a
// layered material.
func (sc *sceneCompiler) createLayeredMaterialTrees() error {
	start := time.Now()
	sc.logger.Noticef("processing %d materials", len(sc.parsedScene.Materials))

	sc.optimizedScene.MaterialNodeList = make([]scene.MaterialNode, 0)
	sc.optimizedScene.MaterialNodeRoots = make([]uint32, len(sc.parsedScene.Materials))

	var rootNodeIndex uint32 = 0
	for matIndex, mat := range sc.parsedScene.Materials {
		sc.logger.Infof(`processing material "%s"`, mat.Name)

		// Till we get material expressions working use a naive approach to building material nodes
		isDiffuse := mat.IsDiffuse()
		isSpecularReflection := mat.IsSpecularReflection()
		isSpecularTransmission := mat.IsSpecularTransmission()
		isEmissive := mat.IsEmissive()

		var materialExpr string
		switch {
		case mat.MaterialExpression != "":
			materialExpr = mat.MaterialExpression
		case isSpecularReflection && !(isDiffuse || isSpecularTransmission || isEmissive):
			materialExpr = "S"
		case isSpecularTransmission && !(isDiffuse || isEmissive):
			materialExpr = "T"
		case isSpecularReflection && isSpecularTransmission && !(isDiffuse || isEmissive):
			materialExpr = "fresnel(S, T)"
		case isEmissive:
			materialExpr = "E"
		default:
			materialExpr = "D"
		}

		// Parse expression and generate material nodes
		lexer := &exprLex{
			line:     []byte(materialExpr),
			compiler: sc,
			material: mat,
		}
		parser := exprNewParser()
		parser.Parse(lexer)

		if lexer.lastError != nil {
			return fmt.Errorf(
				"could not parse expression %q for material %q: %s",
				mat.MaterialExpression,
				mat.Name,
				lexer.lastError,
			)
		}

		// Parser uses a stack so the last node to be appended will be the root
		rootNodeIndex = uint32(len(sc.optimizedScene.MaterialNodeList) - 1)
		sc.optimizedScene.MaterialNodeRoots[matIndex] = rootNodeIndex

		// Handle special scene material
		if mat.Name == sceneDiffuseMaterialName {
			sc.optimizedScene.SceneDiffuseMatIndex = sc.findMaterialNodeByBxdf(rootNodeIndex, scene.Diffuse)
		}
		if mat.Name == sceneEmissiveMaterialName {
			sc.optimizedScene.SceneEmissiveMatIndex = sc.findMaterialNodeByBxdf(rootNodeIndex, scene.Emissive)
		}
	}

	sc.logger.Noticef("processed materials in %d ms", time.Since(start).Nanoseconds()/1e6)
	return nil
}

// Append material node and return its insertion index.
func (sc *sceneCompiler) appendMaterialNode(node *scene.MaterialNode) uint32 {
	sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, *node)
	return uint32(len(sc.optimizedScene.MaterialNodeList) - 1)
}

// Initialize and position the camera for the scene.
func (sc *sceneCompiler) setupCamera() error {
	sc.optimizedScene.Camera = scene.NewCamera(sc.parsedScene.Camera.FOV)
	sc.optimizedScene.Camera.Position = sc.parsedScene.Camera.Eye
	sc.optimizedScene.Camera.LookAt = sc.parsedScene.Camera.Look
	sc.optimizedScene.Camera.Up = sc.parsedScene.Camera.Up

	return nil
}

// Adjust value so its divisible by 4.
func align4(value int) uint32 {
	for {
		if value%4 == 0 {
			return uint32(value)
		}
		value++
	}
}
