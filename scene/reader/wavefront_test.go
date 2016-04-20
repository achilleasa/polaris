package reader

import (
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/achilleasa/go-pathtrace/types"
)

func TestFloat32Parser(t *testing.T) {
	expError := "unsupported syntax for 'v'; expected 1 argument; got 0"
	_, err := parseFloat32([]string{"v"})
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get %s; got %v", expError, err)
	}

	_, err = parseFloat32([]string{"v", "not-a-float"})
	if err == nil {
		t.Fatal("expected to get a parse error")
	}

	v, err := parseFloat32([]string{"v", "3.14"})
	if err != nil {
		t.Fatal(err)
	}

	if v != 3.14 {
		t.Fatalf("expected parsed value to be 3.14; got %f", v)
	}
}

func TestVec2Parser(t *testing.T) {
	expError := "unsupported syntax for 'v'; expected 2 arguments; got 0"
	_, err := parseVec2([]string{"v"})
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get %s; got %v", expError, err)
	}

	_, err = parseVec2([]string{"v", "not-a-float", "2"})
	if err == nil {
		t.Fatal("expected to get a parse error")
	}

	v, err := parseVec2([]string{"v", "3.14", "0"})
	if err != nil {
		t.Fatal(err)
	}

	expVal := types.Vec2{3.14, 0}
	if !reflect.DeepEqual(v, expVal) {
		t.Fatalf("expected parsed value to be %v; got %v", expVal, v)
	}
}

func TestVec3Parser(t *testing.T) {
	expError := "unsupported syntax for 'v'; expected 3 arguments; got 0"
	_, err := parseVec3([]string{"v"})
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get %s; got %v", expError, err)
	}

	_, err = parseVec3([]string{"v", "not-a-float", "2", "3"})
	if err == nil {
		t.Fatal("expected to get a parse error")
	}

	v, err := parseVec3([]string{"v", "3.14", "0", "0.4"})
	if err != nil {
		t.Fatal(err)
	}

	expVal := types.Vec3{3.14, 0, 0.4}
	if !reflect.DeepEqual(v, expVal) {
		t.Fatalf("expected parsed value to be %v; got %v", expVal, v)
	}
}

func TestSelectFaceCoordinate(t *testing.T) {
	expError := "index out of bounds"
	type spec struct {
		in       string
		listLen  int
		out      int
		expError string
	}
	specs := []spec{
		{"2", 1, -1, expError},
		{"-2", 1, -1, expError},
		{"1", 10, 0, ""}, // indices are 1-based
		{"-1", 10, 9, ""},
	}

	for idx, s := range specs {
		v, err := selectFaceCoordIndex(s.in, s.listLen)
		if s.expError != "" && (err == nil || err.Error() != s.expError) {
			t.Fatalf("[spec %d] expected error %s; got %v", idx, s.expError, err)
		} else if v != s.out {
			t.Fatalf("[spec %d] expected index to be %d; got %d", idx, s.out, v)
		}
	}
}

func TestDefaultMeshInstanceGeneration(t *testing.T) {
	payload := `
o testObj
v 0 0 0
v 1 0 0
v 0 1 0
vn 1 0 0
vt 0 0
vn 0 1 0
vt 0 1
vn 0 1 0
vt 1 0
vn 0 0 1
# Comment
f 1/1/1 2/2/2 -1/-1/-1
`

	res := mockResource(payload)
	r := newWavefrontReader()
	r.Read(res)

	expMeshInstances := 1
	if len(r.sceneGraph.MeshInstances) != expMeshInstances {
		t.Fatalf("expected %d mesh instances to be generated; got %d", expMeshInstances, len(r.sceneGraph.MeshInstances))
	}
	inst0 := r.sceneGraph.MeshInstances[0]
	if inst0.MeshIndex != 0 {
		t.Fatalf("expected mesh instance to point to mesh at index 0; got %d", inst0.MeshIndex)
	}
	ident := types.Ident4()
	if !reflect.DeepEqual(inst0.Transform, ident) {
		t.Fatalf("expected mesh instance transform matric to be equal to a 4x4 identity matrix; got %s", inst0.Transform)
	}

	expCenter := types.Vec3{0.5, 0.5, 0}
	if !types.ApproxEqual(inst0.Center(), expCenter, 1e-3) {
		t.Fatalf("expected mesh center to be %v; got %v", expCenter, inst0.Center())
	}
	expBBox := [2]types.Vec3{
		{0, 0, 0},
		{1, 1, 0},
	}
	bbox := inst0.BBox()
	if !types.ApproxEqual(bbox[0], expBBox[0], 1e-3) {
		t.Fatalf("expected bbox min to be %v; got %v", expBBox[0], bbox[0])
	}
	if !types.ApproxEqual(bbox[1], expBBox[1], 1e-3) {
		t.Fatalf("expected bbox max to be %v; got %v", expBBox[1], bbox[1])
	}
}

func TestMeshInstancing(t *testing.T) {
	payload := `
o testObj
v 0 0 0
v 1 0 0
v 0 1 0
vn 1 0 0
vt 0 0
vn 0 1 0
vt 0 1
vn 0 1 0
vt 1 0
vn 0 0 1
# Comment
f 1/1/1 2/2/2 -1/-1/-1
# Mesh instances
instance testObj 	1 0 1	0 0 0 	1 1 1
instance testObj 	0 0 0	0 90 0 	1 1 1
instance testObj 	0 1 0	90 0 0	10 10 10
`

	res := mockResource(payload)
	r := newWavefrontReader()
	r.Read(res)

	expMeshInstances := 3
	if len(r.sceneGraph.MeshInstances) != expMeshInstances {
		t.Fatalf("expected %d mesh instances to be generated; got %d", expMeshInstances, len(r.sceneGraph.MeshInstances))
	}

	type spec struct {
		instance   uint32
		in, expOut types.Vec3
	}
	specs := []spec{
		{0, types.Vec3{0, 0, 0}, types.Vec3{1, 0, 1}},
		{0, types.Vec3{-1, 0, -1}, types.Vec3{0, 0, 0}},
		{1, types.Vec3{1, 0, 0}, types.Vec3{0, 0, -1}},
		{1, types.Vec3{0, 0, -1}, types.Vec3{-1, 0, 0}},
		{2, types.Vec3{0, 1, 0}, types.Vec3{0, 0, 20}},
	}
	for idx, s := range specs {
		inst := r.sceneGraph.MeshInstances[s.instance]
		out := inst.Transform.Mul4x1(s.in.Vec4(1.0)).Vec3()
		if !types.ApproxEqual(out, s.expOut, 1e-3) {
			t.Fatalf("[spec %d] expected transformed point with instance %d matrix to be %v; got %v", idx, s.instance, s.expOut, out)
		}
	}

	expBBoxes := [][2]types.Vec3{
		[2]types.Vec3{types.Vec3{1, 0, 1}, types.Vec3{2, 1, 1}},
	}
	for meshIndex, expBBox := range expBBoxes {
		bbox := r.sceneGraph.MeshInstances[meshIndex].BBox()
		if !types.ApproxEqual(bbox[0], expBBox[0], 1e-3) {
			t.Fatalf("[mesh inst. %d] expected bbox min to be %v; got %v", meshIndex, expBBox[0], bbox[0])
		}
		if !types.ApproxEqual(bbox[1], expBBox[1], 1e-3) {
			t.Fatalf("[mesh inst. %d] expected bbox max to be %v; got %v", meshIndex, expBBox[1], bbox[1])
		}
	}
}

func TestParseSingleFacedObject(t *testing.T) {
	payload := `
o testObj
v 0 0 0
v 1 0 0
v 0 1 0
vn 1 0 0
vt 0 0
vn 0 1 0
vt 0 1
vn 0 1 0
vt 1 0
vn 0 0 1
# Comment
f 1/1/1 2/2/2 -1/-1/-1
`

	res := mockResource(payload)
	r := newWavefrontReader()
	err := r.parse(res)
	if err != nil {
		t.Fatal(err)
	}

	expMeshes := 1
	if len(r.sceneGraph.Meshes) != expMeshes {
		t.Fatalf("expected %d meshes to be parsed; got %d", expMeshes, len(r.sceneGraph.Meshes))
	}

	mesh0 := r.sceneGraph.Meshes[0]
	expName := "testObj"
	if mesh0.Name != expName {
		t.Fatalf("expected mesh[0] name to be '%s'; got %s", expName, mesh0.Name)
	}

	expPrimitives := 1
	if len(mesh0.Primitives) != expPrimitives {
		t.Fatalf("expected mesh[0] to contain %d primitives; got %d", expPrimitives, len(mesh0.Primitives))
	}

	expMaterials := 1
	if len(r.sceneGraph.Materials) != expMaterials {
		t.Fatalf("expected scene to contain %d material(s); got %d", expMaterials, len(r.sceneGraph.Materials))
	}

	expPoints := []types.Vec3{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
	}
	expNormals := []types.Vec3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	expUVs := []types.Vec2{
		{0, 0},
		{0, 1},
		{1, 0},
	}
	prim0 := mesh0.Primitives[0]
	for idx, exp := range expPoints {
		if !reflect.DeepEqual(prim0.Vertices[idx], exp) {
			t.Fatalf("expected vertex %d to be %v; got %v", idx, exp, prim0.Vertices[idx])
		}
	}
	for idx, exp := range expNormals {
		if !reflect.DeepEqual(prim0.Normals[idx], exp) {
			t.Fatalf("expected normal %d to be %v; got %v", idx, exp, prim0.Normals[idx])
		}
	}
	for idx, exp := range expUVs {
		if !reflect.DeepEqual(prim0.UVs[idx], exp) {
			t.Fatalf("expected uv %d to be %v; got %v", idx, exp, prim0.UVs[idx])
		}
	}

	expCenter := types.Vec3{0.333, 0.333, 0}
	if !types.ApproxEqual(prim0.Center(), expCenter, 1e-3) {
		t.Fatalf("expected face center to be %v; got %v", expCenter, prim0.Center())
	}
	expBBox := [2]types.Vec3{
		{0, 0, 0},
		{1, 1, 0},
	}
	bbox := prim0.BBox()
	if !types.ApproxEqual(bbox[0], expBBox[0], 1e-3) {
		t.Fatalf("expected bbox min to be %v; got %v", expBBox[0], bbox[0])
	}
	if !types.ApproxEqual(bbox[1], expBBox[1], 1e-3) {
		t.Fatalf("expected bbox max to be %v; got %v", expBBox[1], bbox[1])
	}
}

func TestMaterialLoaderMissingNewMaterialCommand(t *testing.T) {
	payload := `Kd 1.0 1.0 1.0`
	res := mockResource(payload)
	err := newWavefrontReader().parseMaterials(res)

	expError := "[embedded: 1] error: got 'Kd' without a 'newmtl'"
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get error: %s; got %v", expError, err)
	}
}

func TestMaterialLoaderInvalidVec3Param(t *testing.T) {
	payload := `
	newmtl foo
	Kd 1.0`
	res := mockResource(payload)
	err := newWavefrontReader().parseMaterials(res)

	expError := "[embedded: 3] error: unsupported syntax for 'Kd'; expected 3 arguments; got 1"
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get error: %s; got %v", expError, err)
	}
}

func TestMaterialLoaderInvalidScalarParam(t *testing.T) {
	payload := `
	newmtl foo
	Ni`
	res := mockResource(payload)
	err := newWavefrontReader().parseMaterials(res)

	expError := "[embedded: 3] error: unsupported syntax for 'Ni'; expected 1 argument; got 0"
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get error: %s; got %v", expError, err)
	}
}

func TestMaterialLoaderSuccess(t *testing.T) {
	payload := `
	# comment
	newmtl foo
	Kd 1.0 1.0 1.0
	Ks 0.1 0.2 0.3
	Ke 0.4    0.5 0.6
	Ni 2.5
	Nr 0`
	res := mockResource(payload)
	r := newWavefrontReader()
	err := r.parseMaterials(res)
	if err != nil {
		t.Fatal(err)
	}

	matLen := len(r.sceneGraph.Materials)
	if matLen != 1 {
		t.Fatalf("expected to parse 1 material; got %d", matLen)
	}

	mat := r.sceneGraph.Materials[0]
	if mat.Name != "foo" {
		t.Fatalf("expected material name to be 'foo'; got %s", mat.Name)
	}

	expVec3 := types.Vec3{1, 1, 1}
	if !reflect.DeepEqual(mat.Kd, expVec3) {
		t.Fatalf("expected Kd to be %v; got %v", expVec3, mat.Kd)
	}
	expVec3 = types.Vec3{0.1, 0.2, 0.3}
	if !reflect.DeepEqual(mat.Ks, expVec3) {
		t.Fatalf("expected Ks to be %v; got %v", expVec3, mat.Ks)
	}
	expVec3 = types.Vec3{0.4, 0.5, 0.6}
	if !reflect.DeepEqual(mat.Ke, expVec3) {
		t.Fatalf("expected Ke to be %v; got %v", expVec3, mat.Ke)
	}
	var expScalar float32 = 2.5
	if mat.Ni != expScalar {
		t.Fatalf("expected Ni to be %f; got %f", expScalar, mat.Ni)
	}
	expScalar = 0
	if mat.Nr != expScalar {
		t.Fatalf("expected Nr to be %f; got %f", expScalar, mat.Nr)
	}

}

func TestMaterialLoaderWithTextures(t *testing.T) {
	serverFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		png.Encode(w, image.NewRGBA64(image.Rect(0, 0, 1, 1)))
	})
	server := httptest.NewServer(serverFn)
	defer server.Close()

	payload := `
newmtl foo
map_Kd SERVER/kd.png
map_Ks SERVER/ks.png
map_Ke SERVER/ke.png
map_bump SERVER/bump.png
map_Ni SERVER/ni.png
map_Nr SERVER/nr.png
`
	res := mockResource(strings.Replace(payload, "SERVER", server.URL, -1))
	r := newWavefrontReader()
	err := r.parseMaterials(res)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.sceneGraph.Materials) != 1 {
		t.Fatalf("expected to parse 1 material; got %d", len(r.sceneGraph.Materials))
	}

	expTexCount := 6
	if len(r.sceneGraph.Textures) != expTexCount {
		t.Fatalf("expected to load %d textures; got %d", expTexCount, len(r.sceneGraph.Textures))
	}

	texIndices := 0
	mat := r.sceneGraph.Materials[0]
	if mat.KdTex != -1 {
		texIndices++
	}
	if mat.KsTex != -1 {
		texIndices++
	}
	if mat.KeTex != -1 {
		texIndices++
	}
	if mat.NiTex != -1 {
		texIndices++
	}
	if mat.NrTex != -1 {
		texIndices++
	}
	if mat.NormalTex != -1 {
		texIndices++
	}
	if texIndices != expTexCount {
		t.Fatalf("expected %d texture indices to be assigned to mat0; got %d", expTexCount, texIndices)
	}
}

func TestMaterialLoaderWithMissingTextures(t *testing.T) {
	payload := `
newmtl foo
map_Kd invalid.png
`
	res := mockResource(payload)
	r := newWavefrontReader(res.Path())
	err := r.parseMaterials(res)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.sceneGraph.textures) != 0 {
		t.Fatalf("expected texture list to be empty; got %d items", len(r.sceneGraph.textures))
	}

	mat := r.sceneGraph.materials[0]
	if mat.kdTex != -1 {
		t.Fatalf("expected kdTex to be -1 for missing texture; got %d", mat.kdTex)
	}
}
