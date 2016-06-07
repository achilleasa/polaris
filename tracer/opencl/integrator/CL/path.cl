#ifndef PATH_CL
#define PATH_CL

#define PATH_TERMINATED 0
#define PATH_ACTIVE 1

typedef struct {
	// The accumulated color along this path. This uses the same space as a float4
	float3 throughput;

	// Path status; active/terminated
	uint status;

	// Shaded pixel index for this path 
	uint pixelIndex;

	// Padding; reseved for future use
	uint _reserved1;
	uint _reserved2;
} Path;

void pathNew(Path *path, uint status, uint pixelIndex);
void pathTerminate(Path *path);
void pathMulThroughput(__global Path *path, float3 fragColor);

// Initialize path.
inline void pathNew(Path *path, uint status, uint pixelIndex){
	path->throughput = (float3)(1.0, 1.0, 1.0);
	path->status = status;
	path->pixelIndex = pixelIndex;
}

// Flag path as terminated.
void pathTerminate(Path *path){
	path->status = PATH_TERMINATED;
}

// Multiply a fragment color with the current path throughput.
void pathMulThroughput(__global Path *path, float3 fragColor){
	path->throughput *= fragColor;
}

#endif
