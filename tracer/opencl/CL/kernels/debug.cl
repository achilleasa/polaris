#ifndef DEBUG_KERNELS_CL
#define DEBUG_KERNELS_CL

// Clear debug buffer
__kernel void debugClearBuffer(
		__global uchar4 *output
		){
	output[get_global_id(0)] = (uchar4)(0,0,0,255);
}

// Generate a depth map for primary ray intersections
__kernel void debugPrimaryRayIntersectionDepth(
		__global const int *numRays,
		__global Path *paths,
		__global uint *hitFlags,
		__global Intersection *intersections,
		const float maxDepth,
		__global uchar4 *output
		){

	int globalId = get_global_id(0);
	if(globalId >= *numRays){
		return;
	}

	uint pixelIndex = paths[globalId].pixelIndex;
	float hitDist = intersections[globalId].wuvt.w;

	// No hit
	if(!hitFlags[globalId] || hitDist == FLT_MAX) {
		output[pixelIndex] = (uchar4)(0, 0, 0, 255);
		return;
	}

	uchar sd = uchar(255.0f * (1.0f - hitDist / (maxDepth + 1.0f)));
	output[pixelIndex] = (uchar4)(sd, sd, sd, 255);
}

// Render surface normals for primary ray hits.
__kernel void debugPrimaryRayIntersectionNormals(
		__global const int *numRays,
		__global Path *paths,
		__global uint *hitFlags,
		__global Intersection *intersections,
		__global float4 *vertices,
		__global float4 *normals,
		__global float2 *uv,
		__global uint *materialIndices,
		__global uchar4 *output
		){

	int globalId = get_global_id(0);
	if(globalId >= *numRays){
		return;
	}

	uint pixelIndex = paths[globalId].pixelIndex;
	float hitDist = intersections[globalId].wuvt.w;

	// No hit
	if(!hitFlags[globalId] || hitDist == FLT_MAX) {
		output[pixelIndex] = (uchar4)(0, 0, 0, 255);
		return;
	}

	Surface surface;
	surfaceInit(&surface, intersections + globalId, vertices, normals, uv, materialIndices);

	// convert normal from [-1, 1] -> [0, 255]
	float3 val = (surface.normal + 1.0f) * 255.0f * 0.5f;
	output[pixelIndex] = (uchar4)((uchar)val.x, (uchar)val.y, (uchar)val.z, 255);
}

// Render emissive samples with optional masking for occluded/not-occluded rays.
__kernel void debugEmissiveSamples(
		__global Ray *rays,
		__global const int *numRays,
		__global Path *paths,
		__global uint *hitFlags,
		__global float3 *emissiveSamples,
		const uint maskOccluded,
		const uint maskNotOccluded,
		__global uchar4 *output
		){

	int globalId = get_global_id(0);
	if(globalId >= *numRays){
		return;
	}
	
	int pathIndex = rayGetdPathIndex(rays + globalId);
	uint pixelIndex = paths[pathIndex].pixelIndex;

	// Masked output
	if((maskOccluded && hitFlags[globalId]) || (maskNotOccluded && !hitFlags[globalId])) {
		output[pixelIndex] = (uchar4)(0, 0, 0, 255);
		return;
	} 

	// gamma correct and clamp
	float3 val = clamp(native_powr(emissiveSamples[globalId], 1.0f / 2.2f), 0.0f, 1.0f) * 255.0f;
	output[pixelIndex] = (uchar4)((uchar)val.x, (uchar)val.y, (uchar)val.z, 255);
}

// Visualize throughput
__kernel void debugThroughput(
		__global Path *paths,
		__global uchar4 *output
		){

	int globalId = get_global_id(0);
	uint pixelIndex = paths[globalId].pixelIndex;

	// gamma correct and clamp
	float3 val = clamp(native_powr(paths[globalId].throughput, 1.0f / 2.2f), 0.0f, 1.0f) * 255.0f;
	output[pixelIndex] = (uchar4)((uchar)val.x, (uchar)val.y, (uchar)val.z, 255);
}

// Render accumulator contents
__kernel void debugAccumulator(
		const float sampleWeight,
		__global Path *paths,
		__global float3 *accumulator,
		__global uchar4 *output
		){

	int globalId = get_global_id(0);
	uint pixelIndex = paths[globalId].pixelIndex;

	// gamma correct and clamp
	float3 val = clamp(native_powr(accumulator[globalId] * sampleWeight, 1.0f / 2.2f), 0.0f, 1.0f) * 255.0f;
	output[pixelIndex] = (uchar4)((uchar)val.x, (uchar)val.y, (uchar)val.z, 255);
}

#endif
