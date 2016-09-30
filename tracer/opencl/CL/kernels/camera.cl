#ifndef CAMERA_KERNEL_CL
#define CAMERA_KERNEL_CL

// Generate primary rays.
__kernel void generatePrimaryRays(
		__global Ray *rays, 
		__global int *numRays,
		__global Path *paths,
		const float4 frustrumTL,
		const float4 frustrumTR,
		const float4 frustrumBL,
		const float4 frustrumBR,
		const float3 eyePos,
		const float2 texelDims,
		const uint blockY,
		const uint blockH,
		const uint frameW,
		const uint frameH,
		const uint randSeed
		){

	uint2 globalId;
	globalId.x = get_global_id(0);
	globalId.y = get_global_id(1);

	if(globalId.x == 0 && globalId.y == 0){
		*numRays = frameW * blockH;
	}

	if( globalId.x < frameW && globalId.y < blockH ){
		uint index = (globalId.y * frameW) + globalId.x;
		uint pixelIndex = ((globalId.y + blockY) * frameW) + globalId.x;

		// Apply stratified sampling using a tent filter. This will wrap our
		// random numbers in the [-1, 1] range. X and Y point to the top corner
		// of the current texel so we need to add a bit of offset to get the coords
		// into the [-0.5, 1.5] range.
		uint2 rndState = globalId + randSeed;
		float2 sample0 = randomGetSample2f(&rndState);
		float2 offset = (float2)(
				sample0.x < 0.5f ? native_sqrt(2.0f * sample0.x) - 0.5f : 1.5f - native_sqrt(2.0f - 2.0f * sample0.x),
				sample0.y < 0.5f ? native_sqrt(2.0f * sample0.y) - 0.5f : 1.5f - native_sqrt(2.0f - 2.0f * sample0.y)
		);
		float2 texel = ((float2)(globalId.x, globalId.y + blockY) + offset) * texelDims;

		// Get ray direction using trilinear interpolation
		float4 dir = normalize(
			mix(
				mix(frustrumTL, frustrumBL, texel.y),
				mix(frustrumTR, frustrumBR, texel.y),
				texel.x
			)
		);

		rayNew(rays + index,  eyePos, dir.xyz, FLT_MAX, index);
		pathNew(paths + index, pixelIndex);
	}
}

#endif
