#ifndef SURFACE_CL
#define SURFACE_CL

void surfaceInit(Surface *surface, __global Intersection *intersection, __global float4 *vertices, __global float4 *normals, __global float2 *uv, __global uint *matIndices);
void printSurface(Surface *surface);

// Initialize surface parameters
void surfaceInit(Surface *surface, __global Intersection *intersection, __global float4 *vertices, __global float4 *normals, __global float2 *uv, __global uint *matIndices){
	float3 wuv = intersection->wuvt.xyz;
	int offset = intersection->triIndex * 3;

	// Lerp barycentric coords to get point/normal and uv coords
	surface->point = (wuv.x * vertices[offset] + 
		              wuv.y * vertices[offset+1] + 
					  wuv.z * vertices[offset+2]).xyz;

	surface->normal = normalize(
					  (wuv.x * normals[offset] + 
		               wuv.y * normals[offset+1] + 
					   wuv.z * normals[offset+2]).xyz
			);

	surface->uv = wuv.x * uv[offset] + 
		          wuv.y * uv[offset+1] + 
				  wuv.z * uv[offset+2];

	// Fetch material root node index
	surface->matNodeIndex = matIndices[intersection->triIndex];
}

void printSurface(Surface *surface){
	printf("[tid: %03d] surface (point: %2.2v3hlf, normal: %2.2v3hlf, uv: %2.2v2hlf, matRootNode: %d)\n",
			get_global_id(0),
			surface->point,
			surface->normal,
			surface->uv,
			surface->matNodeIndex
	);
}

#endif
