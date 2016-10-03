# Polaris 

Polaris is a path-tracer. All tooling is written in go while the 
actual path-tracing is performing using an opencl-based backend.

Feature list:
- Read scene data from wavefront [object](docs/scene.md) and [material](docs/materials.md) files
	- Scenes can also be compiled into GPU-optimized format and stored as a compressed zip archive
- Two-level BVH for intersection tests
	- separate BVH for each scene object
	- global BVH for the scene
- Mesh instancing
- Ray packet traversal for primary rays (based on [this](https://graphics.cg.uni-saarland.de/fileadmin/cguds/papers/2007/guenther_07_BVHonGPU/Guenter_et_al._-_Realtime_Ray_Tracing_on_GPU_with_BVH-based_Packet_Traversal.pdf) paper)
- [Layered materials](docs/materials.md)
 	- BxDF models: diffuse, conductor, dielectric, roughConductor, roughDielectric
	- Light dispersion
- Manual texture management; texture size only limited by device memory
	- Bilinear filtering for texture samples
	- Support for most common image formats including openEXR and HDR/RGBE
- Multiple importance sampling (MIS)
- Russian roulette for path termination
- HDR rendering
	- Simple Reinhard tone-mapping post-processing filter
- Pluggable rendering backends
	- Opencl backend split into multiple kernels allowing quick implementation of new features (eg. better camera or MIS/RIS for light selection)
	- Allows implementation of network backend to support multi-node multi-gpu rendering
- Multi-device rendering 
	- Single frame rendering 
	- Interactive opengl-based renderer
	- Pluggable block scheduling algorithms (naive, perfect)

# Getting started

Download and build polaris
```
go get -v github.com/achilleasa/polaris
cd $GOPATH/github.com/achilleasa/polaris
go build
```

For a single frame render run:
```
./polaris render single -width 512 -height 512 -spp 128 -out frame.png https://raw.githubusercontent.com/achilleasa/polaris-example-scenes/master/sphere/sphere.obj
```

For an interactive render run:
```
./polaris render interactive -width 512 -height 512 -blacklist CPU https://raw.githubusercontent.com/achilleasa/polaris-example-scenes/master/sphere/sphere.obj
```

While in interactive mode you can:
- Click and drag mouse to pan camera 
- Use the arrow keys to move around (press shift to double your move speed)
- Press `TAB` to display information about the block allocations between the available opencl devices.

If polaris cannot find any opencl devices it can use it will fail with an error
message. In that case you can still run the above examples by removing the `-blacklist CPU` argument.
For the complete list of CLI commands type `polaris -h` or  check the [CLI docs](docs/cli.md).

Here is an example of polaris running in interactive mode:

![interactive rendering demo](https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBVEY2aHB4bUwxQU0)

# Example renders

You can find some example test scenes at the [polaris-example-scenes](https://github.com/achilleasa/polaris-example-scenes)
repo. You can either checkout the repo locally or just pass a `raw` GH URL to the 
scene `.obj` file when invoking polaris. In the latter case, polaris will detect 
that the scene is being served by a remote host and adjust all relative resource 
paths accordingly.

Here are some example materials rendered by polaris. The mitsuba and cornell box 
models were obtained from [McGuire2011](http://graphics.cs.williams.edu/data/meshes.xml).

![example renders](https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBMElvYWJRMnVHd00)

# License

polaris is distributed under the [MIT license](LICENCE).

# Resources

- The talks by Takahiro Harada:
	- [Introduction to monte carlo ray tracing (CEDEC '13)](http://www.slideshare.net/takahiroharada/introduction-to-monte-carlo-ray-tracing-cedec2013)
	- [Introduction to monte carlo ray tracing (CEDEC '14)](http://www.slideshare.net/takahiroharada/introduction-to-monte-carlo-ray-tracing-opencl-implementation-cedec-2014)
- RadeonRays [GH](https://github.com/GPUOpen-LibrariesAndSDKs/RadeonRays_SDK) repo
- Papers
	- [Brigade renderer: a path tracer for real-time games](https://www.hindawi.com/journals/ijcgt/2013/578269/)
	- [Realtime Ray Tracing on GPU with BVH-based Packet Traversal](https://graphics.cg.uni-saarland.de/fileadmin/cguds/papers/2007/guenther_07_BVHonGPU/Guenter_et_al._-_Realtime_Ray_Tracing_on_GPU_with_BVH-based_Packet_Traversal.pdf)
	- [Microfacet Models for Refraction through Rough Surfaces](https://www.cs.cornell.edu/~srm/publications/EGSR07-btdf.html)
