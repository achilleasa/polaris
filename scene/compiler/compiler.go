package compiler

import (
	"time"

	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	minPrimitivesPerLeaf = 10
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
		parsedScene:    parsedScene,
		optimizedScene: &scene.Scene{},
		logger:         log.New("scene compiler"),
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
	for mIndex, pm := range sc.parsedScene.Meshes {
		volList := make([]BoundedVolume, len(pm.Primitives))
		for index, prim := range pm.Primitives {
			volList[index] = prim
		}

		sc.logger.Infof(`building BVH tree for "%s" (%d primitices)`, pm.Name, len(pm.Primitives))
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

				// Lookup root material node for primitive material index
				sc.optimizedScene.MaterialIndex[primOffset] = sc.optimizedScene.MaterialNodeRoots[prim.MaterialIndex]

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

	sc.logger.Noticef("partitioned geometry in %d ms", time.Since(start).Nanoseconds()/1e6)

	return nil
}

// Convert material definitions into a node-based structure that models a
// layered material.
func (sc *sceneCompiler) createLayeredMaterialTrees() error {
	start := time.Now()
	sc.logger.Noticef("processing %d materials", len(sc.parsedScene.Materials))

	sc.optimizedScene.MaterialNodeList = make([]scene.MaterialNode, 0)
	sc.optimizedScene.MaterialNodeRoots = make([]uint32, len(sc.parsedScene.Materials))

	var nextNodeIndex uint32 = 0
	var node scene.MaterialNode

	for matIndex, mat := range sc.parsedScene.Materials {
		sc.logger.Infof(`processing material "%s"`, mat.Name)
		nextNodeIndex = uint32(len(sc.optimizedScene.MaterialNodeList))
		sc.optimizedScene.MaterialNodeRoots[matIndex] = nextNodeIndex

		// Till we get material expressions working use a naive approach to building material nodes
		isDiffuse := mat.IsDiffuse()
		isSpecular := mat.IsSpecular()
		isRefractive := mat.IsRefractive()
		isEmissive := mat.IsEmissive()

		if isSpecular && !(isDiffuse || isRefractive || isEmissive) {
			// Ideal specular surface
			node = scene.MaterialNode{}
			node.Init()

			node.Kval = mat.Ks.Vec4(0)
			node.SetKvalTex(mat.KsTex)
			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Specular)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
		} else if isRefractive && !(isDiffuse || isSpecular || isEmissive) {
			// Ideal refractive surface
			node = scene.MaterialNode{}
			node.Init()

			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Refractive)
			node.Nval = mat.Ni
			node.SetNvalTex(mat.NiTex)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
		} else if isSpecular && isRefractive && !(isDiffuse || isEmissive) {
			// Ideal dielectric; define a 2-node material using a frensel blend func
			node = scene.MaterialNode{}
			node.Init()

			node.IsNode = 1
			node.SetLeftIndex(nextNodeIndex + 1)
			node.SetRightIndex(nextNodeIndex + 2)
			node.SetBlendFunc(scene.Fresnel)
			node.Nval = mat.Ni
			node.SetNvalTex(mat.NiTex)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)

			// Left child: specular
			node = scene.MaterialNode{}
			node.Init()

			node.Kval = mat.Ks.Vec4(0)
			node.SetKvalTex(mat.KsTex)
			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Specular)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)

			// Right child: refractive
			node = scene.MaterialNode{}
			node.Init()

			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Refractive)
			node.Nval = mat.Ni
			node.SetNvalTex(mat.NiTex)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
		} else if isEmissive && !(isDiffuse || isSpecular || isRefractive) {
			// Light emitter
			node = scene.MaterialNode{}
			node.Init()

			node.Kval = mat.Ke.Vec4(0)
			node.SetKvalTex(mat.KeTex)
			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Emissive)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
		} else {
			// Treat everything else as diffuse
			node = scene.MaterialNode{}
			node.Init()

			node.Kval = mat.Kd.Vec4(0)
			node.SetKvalTex(mat.KdTex)
			node.SetNormalTex(mat.NormalTex)
			node.SetBrdfType(scene.Diffuse)
			sc.optimizedScene.MaterialNodeList = append(sc.optimizedScene.MaterialNodeList, node)
		}
	}

	sc.logger.Noticef("processed materials in %d ms", time.Since(start).Nanoseconds()/1e6)
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

// Adjust value so its divisible by 4.
func align4(value int) uint32 {
	for {
		if value%4 == 0 {
			return uint32(value)
		}
		value++
	}
}
