# Scene format

The scene format is built around the [alias/wavefront object](http://paulbourke.net/dataformats/obj/)
with a few extensions for accessing the advanced rendering features offered by polaris 

When specifying a path to a material file or other external resource:
- A relative path (to the current file) can be used
- An absolute path can be used 
- An http/https URL can be specified to pull the resource from a remote host

# Supported commands of the obj format 

| Command          | Description 
|------------------|----------------
| mtllib           | Import [material](materials.md) library
| usemtl           | use material by name
| v                | specify geometry vertex
| vn               | specify normal vertex
| vt               | specify uv coordinate
| g                | specify object group name
| o                | specify object name
| f                | specify triangular or quad face


# Specifying the scene camera

The following command extensions can be used to specify the scene camera properties:

| Command          | Description         | Type          |Default value | Example
|------------------|---------------------|---------------|--------------|---------------------------
| camera\_fov      | Field of view       | Scalar        | 45           | `camera_fov 60`
| camera\_eye      | Eye position        | Vector        | 0 0 0        | `camera_eye 10 0 0`
| camera\_look     | Camera target       | Vector        | 0 0 -1       | `camera_look 10 -1 0`
| camera\_up       | World up vector     | Vector        | 0 1 0        | `camera_up 0 1 0`

# Including objects from external files

Scene files can include other wavefront object files using the `call` directive.
This allows users to break down complex scenes into multiple files or to compose
scenes by including pre-defined objects.

The `call` directive expects one argument for the file to include. For example:
```obj
call sphere.obj
call http://github.com/achilleasa/polaris/test.obj
```

# Polaris-specific extensions: mesh instancing

Polaris supports `mesh instancing`. This allows users to instantiate multiple
copies of the same object and associate each copy with a 4x4 transformation matrix.
All instances share the same geometry, materials and BVH tree.

A mesh instance can be created using the `instance` directive:
```
instance mesh_name tX tY tZ yaw pitch roll sX sY sZ
```

where:
- mesh\_name is the name of a previously defined group or object.
- tX, tY, tZ specify the object translation vector in world coordinates.
- yaw, pitch, roll specify the rotation angles.
- sX sY sZ specify the object scaling vector. To disable scaling all values must be set to `1`.

If no mesh instances are defined, polaris will automatically generate an instance
for each defined object using an identity transformation matrix.
