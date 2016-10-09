# Polaris CLI overview

# Logging verbosity

Polaris uses leveled logging for its various subsystems. By default, it will 
only print out log entries of level `NOTICE` and `ERROR`.

You can control the  log message verbosity by specifying the `-v` (be more verbose) 
or `-vv` (be even more verbose) options. See `polaris -h` for more details.

For example:

```
polaris -v list-devices
```

# Listing devices

To list the available opencl devices on your system you can use the `list-devices`
command. For example:

```
polaris list-devices

[13:34:04.049] [polaris] [NOTICE] system provides 1 opencl platform(s)
+-------------------------------------------+------+-----------------+--------+-----------------------------------+
|                  Device                   | Type | Estimated speed | Vendor |              Version              |
+-------------------------------------------+------+-----------------+--------+-----------------------------------+
| Intel(R) Core(TM) i7-4870HQ CPU @ 2.50GHz | CPU  | 20 GFlops       | Apple  | OpenCL 1.2 (Apr 26 2016 00:05:53) |
| Iris Pro                                  | GPU  | 48 GFlops       | Apple  | OpenCL 1.2 (Apr 26 2016 00:05:53) |
| AMD Radeon R9 M370X Compute Engine        | GPU  | 8 GFlops        | Apple  | OpenCL 1.2 (Apr 26 2016 00:05:53) |
+-------------------------------------------+------+-----------------+--------+-----------------------------------+
```

The device names (or parts of their name) can be used to blacklist specific
devices when rendering scenes via the `-blacklist command`. For example:
`./polaris render frame -blacklist CPU scene.obj`


# Scene management

## Comple scene

If you need to render the same scene multiple times you can pre-compile it into a 
gpu-friendly compressed format which allows polaris to skip the compilation step 
and begin rendering the scene immediately.

To do this you can use the `scene compile` command. Here is an example:

```
polaris scene compile ../polaris-example-scenes/sphere/sphere.obj

[14:40:09.749] [polaris] [NOTICE] parsing and compiling scene: ../polaris-example-scenes/sphere/sphere.obj
[14:40:09.749] [wavefront scene reader] [NOTICE] parsing scene from "../polaris-example-scenes/sphere/sphere.obj"
[14:40:09.751] [wavefront scene reader] [WARNING] dropping mesh "default" as it contains no polygons
[14:40:09.752] [wavefront scene reader] [NOTICE] pruned 3 unused materials
[14:40:09.752] [wavefront scene reader] [NOTICE] parsed scene in 2 ms
[14:40:09.752] [scene compiler] [NOTICE] compiling scene
[14:40:09.752] [scene compiler] [NOTICE] processing 6 materials
[14:40:09.770] [scene compiler] [NOTICE] processed 6 materials in 17 ms
[14:40:09.770] [scene compiler] [NOTICE] partitioning geometry
[14:40:09.832] [scene compiler] [NOTICE] partitioned geometry in 62 ms
[14:40:09.833] [scene compiler] [NOTICE] compiled scene in 80 ms
[14:40:09.834] [polaris] [NOTICE] scene information:
+----------------+----------------+-----------+
|   Asset Type   |     Asset      |   Size    |
+----------------+----------------+-----------+
| Geometry       | ---            | 98.4 kb   |
|                | Vertices       | 36.5 kb   |
|                | Normals        | 36.5 kb   |
|                | UVs            | 18.2 kb   |
|                | BVH            | 7.2 kb    |
|                |                |           |
| Mesh/emissives | ---            | 160 bytes |
|                | Mesh instances | 80 bytes  |
|                | Emissives      | 80 bytes  |
|                |                |           |
| Materials      | ---            | 3.2 kb    |
|                | Mat. indices   | 3.0 kb    |
|                | Mat. nodes     | 192 bytes |
|                |                |           |
| Textures       | ---            | 4.2 mb    |
|                | Metadata       | 32 bytes  |
|                | Data           | 4.2 mb    |
+----------------+----------------+-----------+
|     Total      |                |  4.3 mb   |
+----------------+----------------+-----------+
[14:40:09.834] [zip scene writer] [NOTICE] writing compressed scene to "../polaris-example-scenes/sphere/sphere.zip"
[14:40:10.058] [zip scene writer] [NOTICE] compressed scene in 223 ms
```

## Display scene details

To display information about a pre-compiled scene you can use the `scene info`
command:

```
polaris scene info ../polaris-example-scenes/sphere/sphere.zip

[14:41:41.592] [zip reader] [NOTICE] parsing compiled scene from "../polaris-example-scenes/sphere/sphere.zip"
[14:41:41.631] [zip reader] [NOTICE] loaded scene in 39 ms
[14:41:41.632] [polaris] [NOTICE] scene information:
+----------------+----------------+-----------+
|   Asset Type   |     Asset      |   Size    |
+----------------+----------------+-----------+
| Geometry       | ---            | 98.4 kb   |
|                | Vertices       | 36.5 kb   |
|                | Normals        | 36.5 kb   |
|                | UVs            | 18.2 kb   |
|                | BVH            | 7.2 kb    |
|                |                |           |
| Mesh/emissives | ---            | 160 bytes |
|                | Mesh instances | 80 bytes  |
|                | Emissives      | 80 bytes  |
|                |                |           |
| Materials      | ---            | 3.2 kb    |
|                | Mat. indices   | 3.0 kb    |
|                | Mat. nodes     | 192 bytes |
|                |                |           |
| Textures       | ---            | 4.2 mb    |
|                | Metadata       | 32 bytes  |
|                | Data           | 4.2 mb    |
+----------------+----------------+-----------+
|     Total      |                |  4.3 mb   |
+----------------+----------------+-----------+
```

# Render

## Single frame 

To render a single frame you can use the `render frame` command. The command 
accepts the following options (see `polaris render frame -h` for more details)

| Parameter           | Description         | Default value 
|---------------------|---------------------|--------------------
| width               | Output frame width                                     | 1024
| height              | Output frame height                                    | 1024
| spp                 | Trace samples per pixel                                | 16
| num-bounces, nb     | Number of ray bounces                                  | 5
| rr-bounces, nr      | Number of ray bounces before applying russian roulette to eliminate paths with small contribution | 3
| exposure            | Exposure value for HDR to LDR mapping                  | 1.2
| blacklist           | Blacklist one or more opencl devices                   | 
| force-primary       | Force an opencl device to be the primary tracer        | the device with max. estimated speed
| out                 | Specify the output filename for the rendered frame     | frame.png

The command expects a scene file as its last argument. The scene file can be either 
a standard wavefront object file or a pre-compiled scene zip archive. In the first 
case, polaris will automatically compile the scene before commencing rendering.

Polaris will automatically detect the available devices on the system, estimate 
each device's speed by querying opencl for the number of compute units and 
memory speed and then use this information to split the frame into blocks which 
are then distributed to the available devices. 

Device speed detection relies on the data returned by opencl and there are cases 
where its incorrectly estimated. For example, when running on a MBP (see example run below)
the integrated Iris device is incorrectly detected as being faster than the discreet 
Radeon device.

If you need to blacklist one or more devices you can use the `-blacklist` option 
to achive this. For example, when running on a MBP with a discreet and integrated GPU 
you can blacklist the CPU and Iris devices using `-blacklist CPU -blacklist Iris`.

```
polaris render frame --blacklist CPU ../polaris-example-scenes/sphere/sphere.obj

[14:49:26.338] [wavefront scene reader] [NOTICE] parsing scene from "../polaris-example-scenes/sphere/sphere.obj"
[14:49:26.340] [wavefront scene reader] [WARNING] dropping mesh "default" as it contains no polygons
[14:49:26.341] [wavefront scene reader] [NOTICE] pruned 3 unused materials
[14:49:26.341] [wavefront scene reader] [NOTICE] parsed scene in 2 ms
[14:49:26.341] [scene compiler] [NOTICE] compiling scene
[14:49:26.341] [scene compiler] [NOTICE] processing 6 materials
[14:49:26.360] [scene compiler] [NOTICE] processed 6 materials in 19 ms
[14:49:26.360] [scene compiler] [NOTICE] partitioning geometry
[14:49:26.423] [scene compiler] [NOTICE] partitioned geometry in 62 ms
[14:49:26.423] [scene compiler] [NOTICE] compiled scene in 82 ms
[14:49:30.984] [renderer] [NOTICE] using device "Iris Pro (0)"
[14:49:32.327] [renderer] [NOTICE] using device "AMD Radeon R9 M370X Compute Engine (1)"
[14:49:32.327] [renderer] [NOTICE] selected "Iris Pro (0)" as primary device
[14:49:36.315] [polaris] [NOTICE] frame statistics
+----------------------------------------+---------+--------------+------------+--------------+
|                 Device                 | Primary | Block height | % of frame | Render time  |
+----------------------------------------+---------+--------------+------------+--------------+
| Iris Pro (0)                           | true    |          878 | 85.7 %     | 3.578712899s |
| AMD Radeon R9 M370X Compute Engine (1) | false   |          146 | 14.3 %     | 181.523917ms |
+----------------------------------------+---------+--------------+------------+--------------+
|                                                                     TOTAL    | 3.987731331s |
+----------------------------------------+---------+--------------+------------+--------------+
```

## Interactive opengl-based renderer

Polaris also provides a progressive, interactive opengl-based renderer. To access 
it you need to use the `render interactive` command. The command accepts the 
following options (see `polaris render interactive -h` for more info):

| Parameter           | Description         | Default value 
|---------------------|---------------------|--------------------
| width               | Output frame width                                     | 1024
| height              | Output frame height                                    | 1024
| spp                 | Trace samples per pixel. When set to 0 progressive rendering is enabled. When set to non-zero, the renderer stop tracing after spp samples are collected | 0
| num-bounces, nb     | Number of ray bounces                                  | 5
| rr-bounces, nr      | Number of ray bounces before applying russian roulette to eliminate paths with small contribution | 3
| exposure            | Exposure value for HDR to LDR mapping                  | 1.2
| blacklist           | Blacklist one or more opencl devices                   | 
| force-primary       | Force an opencl device to be the primary tracer        | the device with max. estimated speed
| scheduler           | Specify the block scheduling algorithm to use: "naive", "perfect" | perfect

When running in interactive mode, you can select an algorithm (via the `-scheduler` option)
that decides how to distribute blocks to the available tracer devices. The following algorithms
are supported:
- `naive`. This is the scheduler used by the [single frame][#single-frame] rendering command. It 
estimates the speed for each available device and distributes the frame blocks accordingly. This
is a **static** algorithm as block distribution is estimated only once and does not change between
subsequent frames.
- `perfect`. For the first frame, the `naive` scheduler is used to get an initial 
block distribution based on the estimated device speed. For each subsequent frame, 
the algorithm calculates the *work* (`w_i = blocks_i / time_i`) performed by each tracer 
in the previous frame as well as the total work performed by all tracers (`W = Σw_i`). 
Based on this information, it emits a new block distribution for the upcoming frame. For
a detailed explanation on how this algorithm works see [Brigade renderer: a path tracer for real-time games](https://www.hindawi.com/journals/ijcgt/2013/578269/)

While the renderer is running you can pan the view by `clicking` with the left 
mouse button and dragging the cursor around. You can also use the `arrow keys`
to move the camera around. The `shift` key can be used together with the arrow 
keys to double the camera move speed. Finally, pressing the `TAB` key will toggle 
a UI which overlays the block distributions for each frame on-top of the rendered frame.
The UI will also render a small stacked line-chart with a history of block distributions 
for the previous frames.

```
polaris render interactive --width 512 --height 512 ../polaris-example-scenes/sphere/sphere.obj
```

![interactive rendering demo](https://drive.google.com/uc?export=download&id=0Bz9Vk3E_v2HBVEY2aHB4bUwxQU0)
