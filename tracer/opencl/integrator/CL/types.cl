#ifndef TYPES_CL
#define TYPES_CL

typedef struct {
	// origin.w stores the max allowed distance for intersection queries.
	float4 origin;

	// dir.w is currently unused.
	float4 dir;
} Ray;

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

typedef struct {
	union {
		float4 minExtent;

		// The W coordinate points to the left child node index if this is a node
		int4 leftChild;

		// The W coordinate points to the mesh instance for top-level BVH leafs
		int4 meshInstance;

		// The W coordinate points to the first triangle in this leaf for bottom-level BVH leafs
		int4 firstTriIndex;
	};

	union {
		float4 maxExtent;

		// The W coordinate points to the right child node index if this is a node
		int4 rightChild;

		// The W coordinate points to the triangle count for this bottom-level BVH leaf
		int4 numTriangles;
	};
} BvhNode;

typedef struct {
	uint meshIndex;

	// BVH root node index for mesh BVH
	uint bvhRoot;

	// padding
	uint _reserved1;
	uint _reserved2;

	// inverted mesh transformation matrix for transforming rays to mesh space
	float4 transformMat[4];
} MeshInstance;

typedef struct {
	// XYZ stores barycentric coords (w,u,v) of hit and 
	// W stores distance from ray origin to hit (t)
	float4 wuvt;
	//
	// Mesh instance that registered hit
	uint meshInstance;

	// Index to triangle that was intersected
	uint triIndex;
	
	// padding
	uint _reserved1;
	uint _reserved2;
} Intersection;

#endif
