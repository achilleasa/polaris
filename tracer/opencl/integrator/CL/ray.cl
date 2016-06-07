typedef struct {
	// origin.w stores the max allowed distance for intersection queries.
	float4 origin;

	// dir.w is currently unused.
	float4 dir;
} Ray;

void rayNew(Ray* ray, float3 origin, float3 dir, float maxDist);

// Initialize ray.
inline void rayNew(Ray *ray, float3 origin, float3 dir, float maxDist){
	ray->origin.xyz = origin;
	ray->origin.w = maxDist;
	ray->dir.xyz = dir;
}


// Generate primary rays.
__kernel void generatePrimaryRays(
		__global Ray* rays, 
		__global Path* paths,
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

	if( globalId.x < frameW && globalId.y < frameH ){
		uint index = (globalId.y * frameW) + globalId.x;
		uint pixelIndex = index + (blockStartY * frameW);

		// Get ray direction using trilinear interpolation
		float2 texel = (float2)(globalId.x, globalId.y) * texelDims;
		float4 dir = normalize(
			mix(
				mix(frustrumTL, frustrumBL, texel.y),
				mix(frustrumTR, frustrumBR, texel.y),
				texel.x
			)
		);

		Ray ray;
		Path path;
		rayNew(&ray,  eyePos, dir.xyz, -1.0f);
		pathNew(&path, PATH_ACTIVE, pixelIndex);

		rays[index] = ray;
		paths[index] = path;
	}
}
