#ifndef RAY_CL
#define RAY_CL

void rayNew(__global Ray* ray, float3 origin, float3 dir, float maxDist, uint pathIndex);
inline float3 rayGetDirAndPathIndex(__global Ray *ray, uint *pathIndex);
inline uint rayGetdPathIndex(__global Ray *ray);

// Initialize ray.
inline void rayNew(__global Ray *ray, float3 origin, float3 dir, float maxDist, uint pathIndex){
	ray->origin = (float4)(origin, maxDist);
	ray->dir = (float4)(dir, (float)pathIndex);
}

// Get dir and path index associated with ray.
inline float3 rayGetDirAndPathIndex(__global Ray *ray, uint *pathIndex){
	float4 dir = ray->dir;
	*pathIndex = (uint)dir.w;
	return dir.xyz;
}

// Get path index associated with ray.
inline uint rayGetPathIndex(__global Ray *ray){
	return (uint)ray->dir.w;
}
#endif
