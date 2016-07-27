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
		const int blockStartY,
		const uint frameW,
		const uint frameH
		){

	uint2 globalId;
	globalId.x = get_global_id(0);
	globalId.y = get_global_id(1);

	if(globalId.x == 0 && globalId.y == 0){
		*numRays = get_global_size(0) * get_global_size(1);
	}

	if( globalId.x < frameW && globalId.y < frameH ){
		uint index = morton2d(globalId);
		uint pixelIndex = ((globalId.y + blockStartY) * frameW) + globalId.x;

		// Get ray direction using trilinear interpolation
		float2 texel = ((float2)(globalId.x, globalId.y) + 0.5f) * texelDims;
		float4 dir = normalize(
			mix(
				mix(frustrumTL, frustrumBL, texel.y),
				mix(frustrumTR, frustrumBR, texel.y),
				texel.x
			)
		);

		rayNew(rays + index,  eyePos, dir.xyz, FLT_MAX, index);
		pathNew(paths + index, PATH_ACTIVE, pixelIndex);
	}
}

#endif
