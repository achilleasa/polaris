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
	if len(r.sceneGraph.meshInstances) != expMeshInstances {
		t.Fatalf("expected %d mesh instances to be generated; got %d", expMeshInstances, len(r.sceneGraph.meshInstances))
	}
	inst0 := r.sceneGraph.meshInstances[0]
	if inst0.mesh != 0 {
		t.Fatalf("expected mesh instance to point to mesh at index 0; got %d", inst0.mesh)
	}
	ident := types.Ident4()
	if !reflect.DeepEqual(inst0.transform, ident) {
		t.Fatalf("expected mesh instance transform matric to be equal to a 4x4 identity matrix; got %s", inst0.transform)
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
	if len(r.sceneGraph.meshInstances) != expMeshInstances {
		t.Fatalf("expected %d mesh instances to be generated; got %d", expMeshInstances, len(r.sceneGraph.meshInstances))
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
		inst := r.sceneGraph.meshInstances[s.instance]
		out := inst.transform.Mul4x1(s.in.Vec4(1.0)).Vec3()
		if !types.ApproxEqual(out, s.expOut, 1e-3) {
			t.Fatalf("[spec %d] expected transformed point with instance %d matrix to be %v; got %v", idx, s.instance, s.expOut, out)
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
	if len(r.sceneGraph.meshes) != expMeshes {
		t.Fatalf("expected %d meshes to be parsed; got %d", expMeshes, len(r.sceneGraph.meshes))
	}

	mesh0 := r.sceneGraph.meshes[0]
	expName := "testObj"
	if mesh0.name != expName {
		t.Fatalf("expected mesh[0] name to be '%s'; got %s", expName, mesh0.name)
	}

	expPrimitives := 1
	if len(mesh0.primitives) != expPrimitives {
		t.Fatalf("expected mesh[0] to contain %d primitives; got %d", expPrimitives, len(mesh0.primitives))
	}

	expMaterials := 1
	if len(r.sceneGraph.materials) != expMaterials {
		t.Fatalf("expected scene to contain %d material(s); got %d", expMaterials, len(r.sceneGraph.materials))
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
	prim0 := mesh0.primitives[0]
	for idx, exp := range expPoints {
		if !reflect.DeepEqual(prim0.vertices[idx], exp) {
			t.Fatalf("expected vertex %d to be %v; got %v", idx, exp, prim0.vertices[idx])
		}
	}
	for idx, exp := range expNormals {
		if !reflect.DeepEqual(prim0.normals[idx], exp) {
			t.Fatalf("expected normal %d to be %v; got %v", idx, exp, prim0.normals[idx])
		}
	}
	for idx, exp := range expUVs {
		if !reflect.DeepEqual(prim0.uvs[idx], exp) {
			t.Fatalf("expected uv %d to be %v; got %v", idx, exp, prim0.uvs[idx])
		}
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

	matLen := len(r.sceneGraph.materials)
	if matLen != 1 {
		t.Fatalf("expected to parse 1 material; got %d", matLen)
	}

	mat := r.sceneGraph.materials[0]
	if mat.name != "foo" {
		t.Fatalf("expected material name to be 'foo'; got %s", mat.name)
	}

	expVec3 := types.Vec3{1, 1, 1}
	if !reflect.DeepEqual(mat.kd, expVec3) {
		t.Fatalf("expected Kd to be %v; got %v", expVec3, mat.kd)
	}
	expVec3 = types.Vec3{0.1, 0.2, 0.3}
	if !reflect.DeepEqual(mat.ks, expVec3) {
		t.Fatalf("expected Ks to be %v; got %v", expVec3, mat.ks)
	}
	expVec3 = types.Vec3{0.4, 0.5, 0.6}
	if !reflect.DeepEqual(mat.ke, expVec3) {
		t.Fatalf("expected Ke to be %v; got %v", expVec3, mat.ke)
	}
	var expScalar float32 = 2.5
	if mat.ni != expScalar {
		t.Fatalf("expected Ni to be %f; got %f", expScalar, mat.ni)
	}
	expScalar = 0
	if mat.nr != expScalar {
		t.Fatalf("expected Nr to be %f; got %f", expScalar, mat.nr)
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

	if len(r.sceneGraph.materials) != 1 {
		t.Fatalf("expected to parse 1 material; got %d", len(r.sceneGraph.materials))
	}

	expTexCount := 6
	if len(r.sceneGraph.textures) != expTexCount {
		t.Fatalf("expected to load %d textures; got %d", expTexCount, len(r.sceneGraph.textures))
	}

	texIndices := 0
	mat := r.sceneGraph.materials[0]
	if mat.kdTex != -1 {
		texIndices++
	}
	if mat.ksTex != -1 {
		texIndices++
	}
	if mat.keTex != -1 {
		texIndices++
	}
	if mat.niTex != -1 {
		texIndices++
	}
	if mat.nrTex != -1 {
		texIndices++
	}
	if mat.normalTex != -1 {
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
