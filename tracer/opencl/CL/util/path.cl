#ifndef PATH_CL
#define PATH_CL

#define PATH_FLAG_DISPERSE_R 1 << 0
#define PATH_FLAG_DISPERSE_G 1 << 1
#define PATH_FLAG_DISPERSE_B 1 << 2

void pathNew(__global Path *path, uint pixelIndex);
void pathMulThroughput(__global Path *path, float3 fragColor);
void pathSetThroughput(__global Path *path, float3 throughput);

// Initialize path.
inline void pathNew(__global Path *path, uint pixelIndex){
	path->throughput = (float3)(1.0f, 1.0f, 1.0f);
	path->pixelIndex = pixelIndex;
	path->flags = 0;
}

// Multiply a fragment color with the current path throughput.
void pathMulThroughput(__global Path *path, float3 fragColor){
	path->throughput *= fragColor;
}

// Set path throughput
void pathSetThroughput(__global Path *path, float3 throughput){
	path->throughput = throughput;
}

#endif
