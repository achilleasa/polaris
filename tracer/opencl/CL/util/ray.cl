#ifndef RAY_CL
#define RAY_CL

void rayNew(__global Ray* ray, float3 origin, float3 dir, float maxDist, int pathIndex);
inline float3 rayGetDirAndPathIndex(__global Ray *ray, int *pathIndex);
inline int rayGetdPathIndex(__global Ray *ray);

// Initialize ray.
inline void rayNew(__global Ray *ray, float3 origin, float3 dir, float maxDist, int pathIndex){
	ray->origin = (float4)(origin, maxDist);
	ray->dir = (float4)(dir, (float)pathIndex);
}

// Get dir and path index associated with ray.
inline float3 rayGetDirAndPathIndex(__global Ray *ray, int *pathIndex){
	float4 dir = ray->dir;
	*pathIndex = (int)dir.w;
	return dir.xyz;
}

// Get path index associated with ray.
inline int rayGetdPathIndex(__global Ray *ray){
	return (int)ray->dir.w;
}
#endif
