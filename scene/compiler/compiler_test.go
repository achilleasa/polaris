package compiler

import (
	"bytes"
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
		parsedScene:    ps,
		optimizedScene: &scene.Scene{},
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
