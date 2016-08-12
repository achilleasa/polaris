#ifndef TYPES_CL
#define TYPES_CL

typedef struct {
	// origin.w stores the max allowed distance for intersection queries.
	float4 origin;

	// the w coordinate stores the path index
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
	float4 transformMat0;
	float4 transformMat1;
	float4 transformMat2;
	float4 transformMat3;
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

typedef struct {
	// intersection point
	float3 point;

	// normal at intersection point
	float3 normal;

	// tangent/bitangent vectors (perpendicular to normal)
	// used for outbound ray generation
	float3 uVec;
	float3 vVec;

	// texture uv coords at intersection point
	float2 uv;

	// material node index
	uint matNodeIndex;
} Surface;

typedef struct {
	uint format;

	// texture dimensions
	uint width;
	uint height;

	// start offset in texture data
	uint dataOffset;
} TextureMetadata;

typedef struct {
	// "color" for node. Depends on BRDF type
	float3 kval;

	// BRDF-specific parameter (roughness e.t.c) for leafs and blend parameter
	// (IOR, mix probability) for nodes
	float nval;

	uint isNode;
	
	// alignment padding
	float _reserved[2];

	union {
		// For intermediate nodes
		uint leftChild;

		// texture index for node "color"
		int kvalTex;
	};

	union {
		// For intermediate nodes
		uint rightChild;

		// texture index for normal map
		int normalTex;
	};

	union {
		// texture index for overriding nval
		int nvalTex;
	};

	union {
		// For intermediate nodes; defines the blend function to use
		// for selecting the left or right child
		uint blendFunc;

		// BXDF type (diffuse, specular e.t.c.) if this is a leaf
		uint bxdfType;
	};
} MaterialNode;

typedef struct {
	// transformation matrix for transforming emissive vertices to world space
	// this is basically the inverse of the transformation matrix from the mesh
	// instance this triangle belongs to
	float4 transformMat0;
	float4 transformMat1;
	float4 transformMat2;
	float4 transformMat3;

	// The emissive area
	float area; 

	// The mesh triangle for this emissive surface
	uint triIndex;

	// The material node index for this emissive. Copied to save a lookup in materialNodeIndices
	uint matNodeIndex;

	// Emissive type
	uint type;
} Emissive;

#endif
