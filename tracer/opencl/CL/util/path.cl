#ifndef PATH_CL
#define PATH_CL

void pathNew(__global Path *path, uint status, uint pixelIndex);
void pathMulThroughput(__global Path *path, float3 fragColor);

// Initialize path.
inline void pathNew(__global Path *path, uint status, uint pixelIndex){
	path->throughput = (float3)(1.0f, 1.0f, 1.0f);
	path->status = status;
	path->pixelIndex = pixelIndex;
}

// Multiply a fragment color with the current path throughput.
void pathMulThroughput(__global Path *path, float3 fragColor){
	path->throughput *= fragColor;
}

#endif
