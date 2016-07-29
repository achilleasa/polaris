#ifndef PT_INTEGRATOR_KERNEL_CL
#define PT_INTEGRATOR_KERNEL_CL

#define MAX_VEC3_COMPONENT(v) (max(v.x,max(v.y,v.z)))
#define DISPLACE_BY_EPSILON(v,n) (v + n * INTERSECTION_EPSILON)

// For each intersection, calculate an outgoing indirect ray based on the 
// surface PDF and also perform direct light sampling emitting occlusion
// rays and light samples. 
//
// If a ray hits an emissive surface, we update the accumulator with emissive
// output multiplied by the current throughput and kill the ray.
__kernel void shadeHits(
		__global Ray *rays,
		global const int *numRays,
		__global Path *paths,
		__global uint *hitFlags,
		__global Intersection *intersections,
		// scene data
		__global float4 *vertices,
		__global float4 *normals,
		__global float2 *uv,
		__global uint *materialIndices,
		__global MaterialNode *materialNodes,
		__global Emissive *emissives,
		const uint numEmissives,
		// texture data
		__global TextureMetadata *texMeta,
		__global uchar *texData,
		// state
		const uint bounce,
		const uint randSeed,
		// occlusion rays and samples
		__global Ray *occlusionRays,
		volatile global int *numOcclusionRays,
		__global float3 *emissiveSamples,
		// indirect rays
		__global Ray *indirectRays,
		volatile global int *numIndirectRays,
		// output accumulator
		__global float3 *accumulator
		){

	// Local counters used to perform atomics inside this WG
	volatile __local int wgNumOcclusionRays;
	volatile __local int wgNumIndirectRays;
	int wgOcclusionRayIndex = -1;
	int wgIndirectRayIndex = -1;

	int localId = get_local_id(0);
	int globalId = get_global_id(0);

	// No work for this thread
	if(globalId >= *numRays){
		return;
	}

	// The first thread should initialize the global counters
	if(globalId == 0){
		*numOcclusionRays = 0;
		*numIndirectRays = 0;
	}

	// The first thread in this WG should initialize the local counters
	if(localId == 0){
		wgNumOcclusionRays = 0;
		wgNumIndirectRays = 0;
	}

	barrier(CLK_LOCAL_MEM_FENCE);

	Surface surface;
	int rayPathIndex;
	float3 curPathThroughput;
	float3 bxdfOutRayDir, bxdfSample;
	float bxdfPdf;
	float3 emissiveOutRayDir, emissiveSample;
	float emissiveSelectionPdf, emissivePdf, distToEmissive;

	if( hitFlags[globalId] ){
		// Init PRNG and generate required samples
		uint2 rndState = (uint2)(randSeed, globalId);
		float2 sample0 = randomGetSample2f(&rndState);
		float2 sample1 = randomGetSample2f(&rndState);

		// Load incoming ray direction and invert it so it points away
		// from the surface. All BxDF formulas use in/out rays that 
		// are going outwards from the surface.
		float3 inRayDir = -rayGetDirAndPathIndex(rays + globalId, &rayPathIndex);
		curPathThroughput = paths[rayPathIndex].throughput;

		// Fill surface data and calculate cos(n, inRay)
		surfaceInit(&surface, intersections + globalId, vertices, normals, uv, materialIndices);
		float inRayDotSurfNormal = dot(inRayDir, surface.normal);

		// Select material
		MaterialNode materialNode;
		matSelectNode(&surface, &materialNode, materialNodes);

		// Check if we hit an emissive node
		if( MAT_IS_EMISSIVE(materialNode) ){
			// Emissive is facing ray and this is the first bounce we need to sample
			// the light and add its contribution to the output. 
			if( inRayDotSurfNormal > 0 ){
				accumulator[rayPathIndex] += curPathThroughput * matGetSample3f(surface.uv, materialNode.kval, materialNode.kvalTex, texMeta, texData);
			}
		} else {
			// Get BXDF sample and generate outgoing ray based on surface BXDF
			bxdfSample = bxdfGetSample(
					&surface, 
					&materialNode, 
					texMeta, 
					texData, 
					sample0, 
					inRayDir, 
					&bxdfOutRayDir, 
					&bxdfPdf
			);

			// To calculate the origin for occlusion/indirect rays we displace the 
			// surface hit point by a small epsilon along the normal to ensure that 
			// we don't register an intersection with the same surface
			float3 outRayOrigin = DISPLACE_BY_EPSILON(surface.point, surface.normal);

			// Select and sample emissive source
			int emissiveIndex = numEmissives > 0 ? emissiveSelect(numEmissives, sample1.x, &emissiveSelectionPdf) : -1;
			if( emissiveIndex > -1 ){
				emissiveSample = emissiveGetSample(
						&surface,
						emissives + emissiveIndex,
						vertices,
						normals,
						uv,
						materialNodes,
						texMeta,
						texData,
						sample1,
						&emissiveOutRayDir,
						&emissivePdf,
						&distToEmissive
				);
			}

			// If we have a valid emissive sample allocate an occlusion ray.
			if( MAX_VEC3_COMPONENT(emissiveSample) > 0.0f && emissivePdf > 0.0f ){
				// Evaluate surface BXDF for the outgoing ray
				// This is light importance sampling
				float3 emissiveBxdf = bxdfEval(
						&surface,
						&materialNode,
						texMeta,
						texData,
						inRayDir,
						emissiveOutRayDir
						);

				emissiveSample = curPathThroughput * emissiveSample * emissiveBxdf * dot(surface.normal, emissiveOutRayDir) / (emissiveSelectionPdf * emissivePdf);
				wgOcclusionRayIndex = atomic_inc(&wgNumOcclusionRays);
			}

			// If we got a valid bxdf sample update the path throughput
			float3 throughput = bxdfSample * dot(surface.normal, bxdfOutRayDir);
			if (MAX_VEC3_COMPONENT(throughput) > 0.0f && bxdfPdf > 0.0f){
				pathSetThroughput(paths + rayPathIndex, curPathThroughput * throughput / bxdfPdf);
				wgIndirectRayIndex = atomic_inc(&wgNumIndirectRays);
			}
		} // if(!MAT_IS_EMISSIVE)
	} // if(hitFlags)

	// When we reach this point, all local threads will have added their occlusion and 
	// indirect ray requirements to the local ray counters. The first thread in this 
	// WG should perform an atomic_add to the global ray counters to get back an 
	// offset which will then be used by the local threads to emit their generated 
	// rays.
	barrier(CLK_LOCAL_MEM_FENCE);
	if(localId == 0){
		if( wgNumOcclusionRays > 0 ){
			wgNumOcclusionRays = atomic_add(numOcclusionRays, wgNumOcclusionRays);
		}
		if( wgNumIndirectRays > 0 ){
			wgNumIndirectRays = atomic_add(numIndirectRays, wgNumIndirectRays);
		}
	}
	barrier(CLK_LOCAL_MEM_FENCE);

	// Emit occlusion ray and sample
	if( wgOcclusionRayIndex != -1 && MAX_VEC3_COMPONENT(emissiveSample) > 0.0f ){
		wgOcclusionRayIndex += wgNumOcclusionRays;
		emissiveSamples[wgOcclusionRayIndex] = emissiveSample;
		rayNew(occlusionRays + wgOcclusionRayIndex, DISPLACE_BY_EPSILON(surface.point, surface.normal), emissiveOutRayDir, distToEmissive - INTERSECTION_EPSILON_X2, rayPathIndex);
	}

	// Emit indirect ray
	if( wgIndirectRayIndex != -1 ){
		wgIndirectRayIndex += wgNumIndirectRays;
		rayNew(indirectRays + wgIndirectRayIndex, DISPLACE_BY_EPSILON(surface.point, surface.normal), bxdfOutRayDir, FLT_MAX, rayPathIndex);
	}
}

// Shade ray misses by sampling the scene background.
__kernel void shadeMisses(
		__global Ray *rays,
		__global const int *numRays,
		__global Path *paths,
		__global uint *hitFlags,
		__global MaterialNode *materialNodes,
		const uint sceneDiffuseMatNodeIndex,
		// Texture data
		__global TextureMetadata *texMeta,
		__global uchar *texData,
		// Output
		__global float3 *accumulator
		){

	int globalId = get_global_id(0);

	// If this thread is inactive or we hit something then ignore
	if( globalId >= *numRays || hitFlags[globalId] ){
		return;
	}

	// Just sample global env map or use scene bg color
	MaterialNode matNode = materialNodes[sceneDiffuseMatNodeIndex];
	int rayPathIndex;
	float2 uv = rayToLatLongUV(rayGetDirAndPathIndex(rays + globalId, &rayPathIndex));

	pathMulThroughput(paths + rayPathIndex, (float3)(0.0f, 0.0f, 0.0f));
	accumulator[rayPathIndex] = matGetSample3f(uv, matNode.kval, matNode.kvalTex, texMeta, texData);
}

// Accumulate emissive samples for emissive surfaces that are not occluded.
__kernel void accumulateEmissiveSamples(
		__global Ray *rays,
		__global const int *numRays,
		__global uint *hitFlags,
		__global float3 *emissiveSamples,
		__global float3 *accumulator
		){

	int globalId = get_global_id(0);

	// If this thread is inactive or we hit something then there is no clear line of sight to the emissive
	if( globalId >= *numRays || hitFlags[globalId] ){
		return;
	}

	int pathIndex = rayGetdPathIndex(rays + globalId);
	accumulator[pathIndex] += emissiveSamples[globalId];
}

#endif
