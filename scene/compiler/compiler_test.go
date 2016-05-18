package compiler

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

func TestAlign16(t *testing.T) {
	for i := 1; i <= 16; i++ {
		if align4(i)%4 != 0 {
			t.Fatalf("expected align4(%d) % 4 to be 0; got %d", i, align4(i)%4)
		}
	}
}

func TestBakeTexture(t *testing.T) {
	ps := &scene.ParsedScene{
		Textures: make([]*scene.ParsedTexture, 0),
	}

	ps.Textures = append(ps.Textures, &scene.ParsedTexture{
		Width:  2,
		Height: 1,
		Format: scene.Rgba8,
		Data: []byte{
			0xBA, 0xDF, 0x00, 0x0D,
			0xBA, 0xDC, 0x00, 0xEE,
		},
	})

	ps.Textures = append(ps.Textures, &scene.ParsedTexture{
		Width:  1,
		Height: 2,
		Format: scene.Rgba8,
		Data: []byte{
			0xDE, 0xAD, 0xBE, 0xEF,
			0xF0, 0x0B, 0xAF, 0x00,
		},
	})

	sc := &sceneCompiler{
		parsedScene:    ps,
		optimizedScene: &scene.Scene{},
	}
	err := sc.bakeTextures()
	if err != nil {
		t.Fatal(err)
	}
	os := sc.optimizedScene

	expCount := 2
	if len(os.TextureMetadata) != expCount {
		t.Fatalf("expected optimized scene to contain %d texture meta entries; got %d", expCount, len(os.TextureMetadata))
	}
	expCount = int(align4(len(ps.Textures[0].Data)) + align4(len(ps.Textures[1].Data)))
	if len(os.TextureData) != expCount {
		t.Fatalf("expected optimized texture data len to be %d; got %d", expCount, len(os.TextureData))
	}

	var expOffset uint32 = 0
	for index, psTex := range ps.Textures {
		osTex := os.TextureMetadata[index]

		if osTex.Width != psTex.Width {
			t.Fatalf("[tex %d] expected texture width to be %d; got %d", index, psTex.Width, osTex.Width)
		}
		if osTex.Height != psTex.Height {
			t.Fatalf("[tex %d] expected texture height to be %d; got %d", index, psTex.Height, osTex.Height)
		}
		if osTex.Format != psTex.Format {
			t.Fatalf("[tex %d] expected texture format to be %d; got %d", index, psTex.Format, osTex.Format)
		}
		if osTex.DataOffset != expOffset {
			t.Fatalf("[tex %d] expected data offset to be %d; got %d", index, expOffset, osTex.DataOffset)
		}
		osData := os.TextureData[expOffset : expOffset+uint32(len(psTex.Data))]
		if !bytes.Equal(osData, psTex.Data) {
			t.Fatalf("[tex %d] expected copied data to be equal to original texture data", index)
		}
		expOffset += align4(len(psTex.Data))
	}
}

func TestPartitionGeometry(t *testing.T) {
	ps := &scene.ParsedScene{
		Meshes: []*scene.ParsedMesh{
			&scene.ParsedMesh{
				Name: "triangle",
				Primitives: []*scene.ParsedPrimitive{
					&scene.ParsedPrimitive{
						Vertices: [3]types.Vec3{
							{0, 0, 0},
							{1.0, 0, 0},
							{0.5, 1.0, 0},
						},
						Normals: [3]types.Vec3{
							{0, 0, -1},
							{0, 0, -1},
							{0, 0, -1},
						},
						UVs: [3]types.Vec2{
							{0, 0},
							{1.0, 0},
							{0.5, 1.0},
						},
					},
				},
			},
		},
		MeshInstances: []*scene.ParsedMeshInstance{
			&scene.ParsedMeshInstance{
				MeshIndex: 0,
				Transform: types.Translate4(types.Vec3{0, 0, -2}),
			},
			&scene.ParsedMeshInstance{
				MeshIndex: 0,
				Transform: types.Ident4(),
			},
		},
	}

	bbox := [2]types.Vec3{
		{0, 0, 0},
		{1.0, 1.0, 0},
	}
	center := types.Vec3{0.5, 0.5, 0}
	ps.Meshes[0].Primitives[0].SetBBox(bbox)
	ps.Meshes[0].Primitives[0].SetCenter(center)
	ps.Meshes[0].SetBBox(bbox)
	ps.MeshInstances[0].SetBBox([2]types.Vec3{
		ps.MeshInstances[0].Transform.Mul4x1(bbox[0].Vec4(1)).Vec3(),
		ps.MeshInstances[0].Transform.Mul4x1(bbox[1].Vec4(1)).Vec3(),
	})
	ps.MeshInstances[0].SetCenter(ps.MeshInstances[0].Transform.Mul4x1(center.Vec4(1.0)).Vec3())
	ps.MeshInstances[1].SetBBox(bbox)
	ps.MeshInstances[1].SetCenter(center)

	sc := &sceneCompiler{
		parsedScene: ps,
		optimizedScene: &scene.Scene{
			MaterialNodeRoots: []uint32{0},
		},
	}
	err := sc.partitionGeometry()
	if err != nil {
		t.Fatal(err)
	}
	os := sc.optimizedScene

	expCount := 3
	if len(os.VertexList) != expCount {
		t.Fatalf("expected optimized vertex count to be %d; got %d", expCount, len(os.VertexList))
	}
	if len(os.NormalList) != expCount {
		t.Fatalf("expected optimized normal count to be %d; got %d", expCount, len(os.NormalList))
	}
	if len(os.UvList) != expCount {
		t.Fatalf("expected optimized uv count to be %d; got %d", expCount, len(os.UvList))
	}
	expCount = 1
	if len(os.MaterialIndex) != expCount {
		t.Fatalf("expected optimized material index count to be %d; got %d", expCount, len(os.MaterialIndex))
	}

	expCount = 4
	if len(os.BvhNodeList) != expCount {
		t.Fatalf("expected bvh node count to be %d; got %d", expCount, len(os.BvhNodeList))
	}

	expCount = 2
	if len(os.MeshInstanceList) != expCount {
		t.Fatalf("expected optimized mesh instance count to be %d; got %d", expCount, len(os.MeshInstanceList))
	}

	leaf0 := os.BvhNodeList[1]
	leaf1 := os.BvhNodeList[2]
	if meshIndex := leaf0.GetMeshIndex(); meshIndex != 0 {
		t.Fatalf("expected bvh top leaf 0 to point to mesh 0; got %d", meshIndex)
	}
	if meshIndex := leaf1.GetMeshIndex(); meshIndex != 1 {
		t.Fatalf("expected bvh top leaf 1 to point to mesh 1; got %d", meshIndex)
	}

	if os.MeshInstanceList[0].BvhRoot != 3 {
		t.Fatalf("expected bvh bottom root for mesh instance 0 to be 3; got %d", os.MeshInstanceList[0].BvhRoot)
	}
	if os.MeshInstanceList[1].BvhRoot != 3 {
		t.Fatalf("expected bvh bottom root for mesh instance 1 to be 3; got %d", os.MeshInstanceList[1].BvhRoot)
	}
}

func TestCreateLayeredMaterialTrees(t *testing.T) {
	ps := &scene.ParsedScene{
		Materials: []*scene.ParsedMaterial{
			&scene.ParsedMaterial{
				Kd:        types.Vec3{1, 1, 1},
				KdTex:     1,
				NormalTex: 2,
			},
			&scene.ParsedMaterial{
				Ks:        types.Vec3{1, 1, 1},
				KsTex:     1,
				NormalTex: 2,
				//
				KdTex: -1,
				KeTex: -1,
				NiTex: -1,
			},
			&scene.ParsedMaterial{
				NormalTex: 2,
				Ni:        1.333,
				NiTex:     1,
				//
				KdTex: -1,
				KsTex: -1,
				KeTex: -1,
			},
			&scene.ParsedMaterial{
				Ks:        types.Vec3{1, 1, 1},
				KsTex:     1,
				NormalTex: 2,
				Ni:        1.333,
				NiTex:     3,
				//
				KdTex: -1,
				KeTex: -1,
			},
			&scene.ParsedMaterial{
				Ke:    types.Vec3{10, 10, 10},
				KeTex: 1,
				//
				KdTex: -1,
				KsTex: -1,
				NiTex: -1,
			},
		},
	}

	sc := &sceneCompiler{
		parsedScene:    ps,
		optimizedScene: &scene.Scene{},
	}

	err := sc.createLayeredMaterialTrees()
	if err != nil {
		t.Fatal(err)
	}

	if len(sc.optimizedScene.MaterialNodeRoots) != len(ps.Materials) {
		t.Fatalf("expected len(MaterialNodeRoots) to be %d; got %d", len(ps.Materials), len(sc.optimizedScene.MaterialNodeRoots))
	}

	var node scene.MaterialNode
	var expValue int32
	var matIndex int

	// First material should be diffuse
	matIndex = 0
	node = sc.optimizedScene.MaterialNodeList[sc.optimizedScene.MaterialNodeRoots[matIndex]]

	if !reflect.DeepEqual(node.Kval, ps.Materials[matIndex].Kd.Vec4(0)) {
		t.Fatalf("[mat %d] expected Kval to be %#+v; got %#+v", matIndex, ps.Materials[matIndex].Kd, node.Kval)
	}

	expValue = ps.Materials[matIndex].KdTex
	if node.UnionData[0] != expValue {
		t.Fatalf("[mat %d] expected Kval tex index to be %d; got %d", matIndex, expValue, node.UnionData[matIndex])
	}

	expValue = ps.Materials[matIndex].NormalTex
	if node.UnionData[1] != expValue {
		t.Fatalf("[mat %d] expected Normal tex index to be %d; got %d", matIndex, expValue, node.UnionData[1])
	}

	if node.UnionData[3] != int32(scene.Diffuse) {
		t.Fatalf("[mat %d] expected BRDF type to be Diffuse; got %d", matIndex, node.UnionData[3])
	}

	// Second material should be specular
	matIndex = 1
	node = sc.optimizedScene.MaterialNodeList[sc.optimizedScene.MaterialNodeRoots[matIndex]]

	if !reflect.DeepEqual(node.Kval, ps.Materials[matIndex].Ks.Vec4(0)) {
		t.Fatalf("[mat %d] expected Kval to be %#+v; got %#+v", matIndex, ps.Materials[matIndex].Ks, node.Kval)
	}

	expValue = ps.Materials[matIndex].KsTex
	if node.UnionData[0] != expValue {
		t.Fatalf("[mat %d] expected Kval tex index to be %d; got %d", matIndex, expValue, node.UnionData[matIndex])
	}

	expValue = ps.Materials[matIndex].NormalTex
	if node.UnionData[1] != expValue {
		t.Fatalf("[mat %d] expected Normal tex index to be %d; got %d", matIndex, expValue, node.UnionData[1])
	}

	if node.UnionData[3] != int32(scene.Specular) {
		t.Fatalf("[mat %d] expected BRDF type to be Specular; got %d", matIndex, node.UnionData[3])
	}

	// Third material should be refractive
	matIndex = 2
	node = sc.optimizedScene.MaterialNodeList[sc.optimizedScene.MaterialNodeRoots[matIndex]]

	if node.Nval != ps.Materials[matIndex].Ni {
		t.Fatalf("[mat %d] expected Nval to be %f; got %f", matIndex, ps.Materials[matIndex].Ni, node.Nval)
	}

	expValue = ps.Materials[matIndex].NiTex
	if node.UnionData[2] != expValue {
		t.Fatalf("[mat %d] expected Ni tex index to be %d; got %d", matIndex, expValue, node.UnionData[2])
	}

	expValue = ps.Materials[matIndex].NormalTex
	if node.UnionData[1] != expValue {
		t.Fatalf("[mat %d] expected Normal tex index to be %d; got %d", matIndex, expValue, node.UnionData[1])
	}

	if node.UnionData[3] != int32(scene.Refractive) {
		t.Fatalf("[mat %d] expected BRDF type to be Refractive; got %d", matIndex, node.UnionData[3])
	}

	// Fourth material should be a 2-level specular/refractive material
	matIndex = 3
	node = sc.optimizedScene.MaterialNodeList[sc.optimizedScene.MaterialNodeRoots[matIndex]]

	if node.IsNode != 1 {
		t.Fatalf("[mat %d] expected to find an intermediate node; got leaf", matIndex)
	}

	if node.UnionData[3] != int32(scene.Fresnel) {
		t.Fatalf("[mat %d] expected material node to specify a Fresnel blend func; got %d", matIndex, node.UnionData[2])
	}

	// Expect left node to be specular
	{
		leftNode := sc.optimizedScene.MaterialNodeList[node.UnionData[0]]

		if !reflect.DeepEqual(leftNode.Kval, ps.Materials[matIndex].Ks.Vec4(0)) {
			t.Fatalf("[mat %d - left child] expected Kval to be %#+v; got %#+v", matIndex, ps.Materials[matIndex].Ks, leftNode.Kval)
		}

		expValue = ps.Materials[matIndex].KsTex
		if leftNode.UnionData[0] != expValue {
			t.Fatalf("[mat %d - left child] expected Kval tex index to be %d; got %d", matIndex, expValue, leftNode.UnionData[matIndex])
		}

		expValue = ps.Materials[matIndex].NormalTex
		if leftNode.UnionData[1] != expValue {
			t.Fatalf("[mat %d - left child] expected Normal tex index to be %d; got %d", matIndex, expValue, leftNode.UnionData[1])
		}

		if leftNode.UnionData[3] != int32(scene.Specular) {
			t.Fatalf("[mat %d - left child] expected BRDF type to be Specular; got %d", matIndex, leftNode.UnionData[3])
		}
	}

	// Expect right node to be refractive
	{
		rightNode := sc.optimizedScene.MaterialNodeList[node.UnionData[1]]

		if rightNode.Nval != ps.Materials[matIndex].Ni {
			t.Fatalf("[mat %d - right child] expected Nval to be %f; got %f", matIndex, ps.Materials[matIndex].Ni, rightNode.Nval)
		}

		expValue = ps.Materials[matIndex].NiTex
		if rightNode.UnionData[2] != expValue {
			t.Fatalf("[mat %d - right child] expected Ni tex index to be %d; got %d", matIndex, expValue, rightNode.UnionData[2])
		}

		expValue = ps.Materials[matIndex].NormalTex
		if rightNode.UnionData[1] != expValue {
			t.Fatalf("[mat %d - right child] expected Normal tex index to be %d; got %d", matIndex, expValue, rightNode.UnionData[1])
		}

		if rightNode.UnionData[3] != int32(scene.Refractive) {
			t.Fatalf("[mat %d - right child] expected BRDF type to be Refractive; got %d", matIndex, rightNode.UnionData[3])
		}
	}

	// Fifth material should be emissive
	matIndex = 4
	node = sc.optimizedScene.MaterialNodeList[sc.optimizedScene.MaterialNodeRoots[matIndex]]

	if !reflect.DeepEqual(node.Kval, ps.Materials[matIndex].Ke.Vec4(0)) {
		t.Fatalf("[mat %d] expected Kval to be %#+v; got %#+v", matIndex, ps.Materials[matIndex].Ke, node.Kval)
	}

	expValue = ps.Materials[matIndex].NormalTex
	if node.UnionData[1] != expValue {
		t.Fatalf("[mat %d] expected Normal tex index to be %d; got %d", matIndex, expValue, node.UnionData[1])
	}

	if node.UnionData[3] != int32(scene.Emissive) {
		t.Fatalf("[mat %d] expected BRDF type to be Emissive; got %d", matIndex, node.UnionData[3])
	}
}
