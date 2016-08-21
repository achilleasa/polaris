#ifndef PT_INTEGRATOR_KERNEL_CL
#define PT_INTEGRATOR_KERNEL_CL

#define MAX_VEC3_COMPONENT(v) (max(v.x,max(v.y,v.z)))
#define MIN_VEC3_COMPONENT(v) (min(v.x,min(v.y,v.z)))
#define DISPLACE_BY_EPSILON(v,n) (v + n * INTERSECTION_EPSILON)

#define MIN_BOUNCES_FOR_RR 3

#define BALANCE_HEURISTIC(a,b) a/(a+b)
#define POWER_HEURISTIC(a,b) (a*a)/(a*a+b*b)

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
	float3 outBxdfRayOrigin, outEmissiveRayOrigin;
	float3 curPathThroughput;
	float3 bxdfOutRayDir, bxdfSample, bxdfEmissiveSample, emissiveOutRayDir, emissiveSample;
	float bxdfPdf, bxdfEmissivePdf, emissivePdf, emissiveBxdfPdf, emissiveSelectionPdf;
	float emissiveWeight, bxdfWeight, distToEmissive;

	if(globalId < *numRays){
		if( hitFlags[globalId] ){
			bxdfPdf = 1.0f;
			bxdfWeight = 1.0f;

			// Init PRNG and generate required samples
			uint2 rndState = (uint2)(randSeed, globalId);
			float2 sample0 = randomGetSample2f(&rndState);
			float2 sample1 = randomGetSample2f(&rndState);
			float2 sample2 = randomGetSample2f(&rndState);

			// Load incoming ray direction and invert it so it points away
			// from the surface. All BxDF formulas use in/out rays that 
			// are going outwards from the surface.
			float3 inRayDir = -rayGetDirAndPathIndex(rays + globalId, &rayPathIndex);
			curPathThroughput = paths[rayPathIndex].throughput;

			// Fill surface data and calculate cos(n, inRay)
			surfaceInit(&surface, intersections + globalId, vertices, normals, uv, materialIndices);

			// Select material
			MaterialNode materialNode;
			matSelectNode(&surface, inRayDir, &materialNode, materialNodes, &rndState, texMeta, texData);

			// Apply normal map
			if(materialNode.normalTex != -1){
				surface.normal = matGetNormalSample3f(surface.normal, surface.uv, materialNode.normalTex, texMeta, texData);
			}

			float inRayDotNormal = dot(inRayDir, surface.normal);

			// Check if we hit an emissive node. If so, we need to accumulate implicit
			// light and terminate the path.
			if( BXDF_IS_EMISSIVE(materialNode.bxdfType) ){
				// Make sure that the incoming ray is facing the emissive.
				if( inRayDotNormal > 0.0f ){
					accumulator[rayPathIndex] += curPathThroughput * matGetSample3f(surface.uv, materialNode.kval, materialNode.kvalTex, texMeta, texData);
				}
			} else {
				// Implement RR to terminate paths with no significant contribution
				// killing paths with a probability less than sample2.x while also
				// boosting surving paths by the same probablility.
				bool rejectSample = false;
				if(bounce >= MIN_BOUNCES_FOR_RR) {
					float rrProbability = max(
							// convert throughput to luminance
							min(0.5f, 0.2126f * curPathThroughput.x + 0.7152f * curPathThroughput.y + 0.0722f * curPathThroughput.z),
							0.01f
							);
					if (rrProbability < sample2.x){
						rejectSample = true;
					} else {
						curPathThroughput /= rrProbability;
					}
				}

				if( !rejectSample ){
					// To calculate the origin for occlusion/indirect rays we displace the 
					// surface hit point by a small epsilon along the normal to ensure that 
					// we don't register an intersection with the same surface.  If this 
					// material is refractive and we are hitting it from the outside we 
					// need to ensure that the outgoing ray starts inside the surface.
					float displaceDir = BXDF_IS_TRANSMISSION(materialNode.bxdfType) ? -sign(inRayDotNormal) : 1.0f;
					outBxdfRayOrigin = DISPLACE_BY_EPSILON(surface.point, surface.normal * displaceDir);
					// The emissive ray always starts away from the surface. This allows us to shade BTDFs
					outEmissiveRayOrigin = DISPLACE_BY_EPSILON(surface.point, surface.normal);

					// Get BXDF sample and generate outgoing ray based on surface BXDF
					bxdfSample = bxdfGetSample(&surface, &materialNode, texMeta, texData, sample0, inRayDir, &bxdfOutRayDir, &bxdfPdf);

					// Select and sample emissive source
					int emissiveIndex = numEmissives > 0 ? emissiveSelect(numEmissives, sample1.x, &emissiveSelectionPdf) : -1;
					if( emissiveIndex > -1 ){
						emissiveSample = emissiveGetSample(&surface, emissives + emissiveIndex, vertices, normals, uv, materialNodes, texMeta, texData, sample1, &emissiveOutRayDir, &emissivePdf, &distToEmissive);

						// MIS: we already have a PDF for generating emissiveOutRayDir.
						// Calculate a PDF for the BXDF sampler generating the same ray 
						// and generate sampling weights using the power heuristic.
						bxdfEmissivePdf = bxdfGetPdf(&surface, &materialNode, texMeta, texData, inRayDir, emissiveOutRayDir);
						emissiveWeight = POWER_HEURISTIC(emissivePdf, bxdfEmissivePdf);

						// We use the same approach to calculate a weight for the BXDF sample by 
						// calculating the PDF for the emissive sampler generating bxdfOutRayDir
						emissiveBxdfPdf = emissiveGetPdf(&surface, emissives + emissiveIndex, vertices, normals, uv, materialNodes, texMeta, texData, bxdfOutRayDir);
						bxdfWeight = POWER_HEURISTIC(bxdfPdf, emissiveBxdfPdf);
					}

					// If we have a valid emissive sample allocate an occlusion ray.
					float nDotEmissiveOutRay = max(0.0f, dot(surface.normal, emissiveOutRayDir));
					if( MAX_VEC3_COMPONENT(emissiveSample) > 0.0f && emissivePdf > 0.0f && nDotEmissiveOutRay > 0.0f){
						float3 bxdfEmissiveSample = bxdfEval(&surface, &materialNode, texMeta, texData, inRayDir, emissiveOutRayDir);
						emissiveSample *= emissiveWeight * bxdfEmissiveSample / (emissivePdf * emissiveSelectionPdf) * curPathThroughput * nDotEmissiveOutRay;
						wgOcclusionRayIndex = atomic_inc(&wgNumOcclusionRays);
					}

					// If we got a valid bxdf sample update the path throughput
					// Note: we are using the abs value of the dot product as 
					// it will be negative for rays entering into refractive surfaces
					float3 throughput = bxdfWeight * bxdfSample * fabs(dot(surface.normal, bxdfOutRayDir));
					if (MAX_VEC3_COMPONENT(throughput) > 0.0f && bxdfPdf > 0.0f){
						pathSetThroughput(paths + rayPathIndex, curPathThroughput * throughput / bxdfPdf);
						wgIndirectRayIndex = atomic_inc(&wgNumIndirectRays);
					} 
				} // if(!rejectSample)
			} // if(BXDF_IS_EMISSIVE)
		} // if(hitFlags)
	} // if(globalId < *numRays)

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
		rayNew(occlusionRays + wgOcclusionRayIndex, outEmissiveRayOrigin, emissiveOutRayDir, distToEmissive - INTERSECTION_WITH_LIGHT_EPSILON, rayPathIndex);
	}

	// Emit indirect ray
	if( wgIndirectRayIndex != -1 ){
		wgIndirectRayIndex += wgNumIndirectRays;
		rayNew(indirectRays + wgIndirectRayIndex, outBxdfRayOrigin, bxdfOutRayDir, FLT_MAX, rayPathIndex);
	}
}

// Shade primary ray misses by sampling the scene background.
__kernel void shadePrimaryRayMisses(
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

	float3 kd = matGetSample3f(uv, matNode.kval, matNode.kvalTex, texMeta, texData);
	accumulator[rayPathIndex] += kd;
}

// Shade indirect ray misses by sampling the scene background.
__kernel void shadeIndirectRayMisses(
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

	// As this is an indirect ray we need to multiply the path throughput with the diffuse sample
	// and accumulate that.
	float3 kd = matGetSample3f(uv, matNode.kval, matNode.kvalTex, texMeta, texData);
	accumulator[rayPathIndex] += paths[rayPathIndex].throughput * kd;
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
