#ifndef MATERIAL_SAMPLER_CL
#define MATERIAL_SAMPLER_CL

#define MAT_OP_MIX 		  10001
#define MAT_OP_MIX_MAP    10002
#define MAT_OP_BUMP_MAP   10003
#define MAT_OP_NORMAL_MAP 10004
#define MAT_OP_DISPERSE   10005
#define MAT_NODE_IS_OP(node) (node->type >= MAT_OP_MIX)
#ifndef BXDF_INVALID
	#define BXDF_INVALID 0
#endif

void matSelectNode(__global Path *path, Surface *surface, float3 inRayDir, MaterialNode *selectedMaterial, float3 *tint, __global MaterialNode* materialNodes, uint2 *rndState, __global TextureMetadata *texMeta, __global uchar *texData );
float3 matGetSample3f(float2 uv, float3 defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);
float matGetSample1f(float2 uv, float defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);
float3 matGetBumpSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);
float3 matGetNormalSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);

// Traverse the layered material tree for this surface and select a leaf node
void matSelectNode(__global Path *path, Surface *surface, float3 inRayDir, MaterialNode *selectedMaterial, float3 *tint, __global MaterialNode* materialNodes, uint2 *rndState, __global TextureMetadata *texMeta, __global uchar *texData ){
	__global MaterialNode* node = materialNodes + surface->matNodeIndex;
	float2 sample;
	float2 forceIOR = (float2)(0.0f, 0.0f);
	uint flags;
	while(MAT_NODE_IS_OP(node)) {
		switch(node->type){
			case MAT_OP_MIX: 
				// Depending on the sample, follow left or right
				sample = randomGetSample2f(rndState);
				node = materialNodes + (sample.x < node->mixWeight ? node->leftChild : node->rightChild);
				break;
			case MAT_OP_MIX_MAP: 
				// Sample weight from texture
				sample = randomGetSample2f(rndState);
				sample.y = texGetSample1f(surface->uv, node->mixWeightsTex, texMeta, texData);
				node = materialNodes + (sample.x < sample.y ? node->leftChild : node->rightChild);
				break;
			case MAT_OP_BUMP_MAP:
				surface->normal = matGetBumpSample3f(surface->normal, surface->uv, node->bumpTex, texMeta, texData);
				node = materialNodes + node->leftChild;
				break;
			case MAT_OP_NORMAL_MAP:
				surface->normal = matGetNormalSample3f(surface->normal, surface->uv, node->bumpTex, texMeta, texData);
				node = materialNodes + node->leftChild;
				break;
			case MAT_OP_DISPERSE:
				flags = path->flags;
				// If the path already has a disperse flag set use it to selecte
				// the IOR values and use the tint value as a primary color filter.
				// Otherwise, we randomly select a channel and set the proper 
				// dispersion flag so it can be reused when exiting the material.
				if( (flags & PATH_FLAG_DISPERSE_R) != 0 ){
						*tint = (float3)(1.0f, 0.0f, 0.0f);
						forceIOR = (float2)(node->intDispersionIORs.x, node->extDispersionIORs.x);
				} else if( (flags & PATH_FLAG_DISPERSE_G) != 0 ){
						*tint = (float3)(0.0f, 1.0f, 0.0f);
						forceIOR = (float2)(node->intDispersionIORs.y, node->extDispersionIORs.y);
				} else if( (flags & PATH_FLAG_DISPERSE_B) != 0 ){
						*tint = (float3)(0.0f, 0.0f, 1.0f);
						forceIOR = (float2)(node->intDispersionIORs.z, node->extDispersionIORs.z);
				} else {
					sample = randomGetSample2f(rndState);
					if( sample.x < 0.333f ){
						*tint = (float3)(1.0f, 0.0f, 0.0f);
						forceIOR = (float2)(node->intDispersionIORs.x, node->extDispersionIORs.x);
						path->flags |= PATH_FLAG_DISPERSE_R;
					} else if (sample.x < 0.666f) {
						*tint = (float3)(0.0f, 1.0f, 0.0f);
						forceIOR = (float2)(node->intDispersionIORs.y, node->extDispersionIORs.y);
						path->flags |= PATH_FLAG_DISPERSE_G;
					} else {
						*tint = (float3)(0.0f, 0.0f, 1.0f);
						forceIOR = (float2)(node->intDispersionIORs.z, node->extDispersionIORs.z);
						path->flags |= PATH_FLAG_DISPERSE_B;
					}
				}
				node = materialNodes + node->leftChild;
				break;
		}
	}

	*selectedMaterial = *node;

	// Apply dispersion IORs
	selectedMaterial->intIOR = max(selectedMaterial->intIOR, forceIOR.x);
	selectedMaterial->extIOR = max(selectedMaterial->extIOR, forceIOR.y);
}

// Sample texture using the supplied uv coordinates and return a float3 vector. 
// If texIndex is -1 then fall-back to the supplied default value.
float3 matGetSample3f(float2 uv, float3 defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData){
	if( texIndex == -1 ){
		return defaultValue;
	}

	return texGetSample3f( uv, texIndex, texMeta, texData );
}

// Sample texture using the supplied uv coordinates and return a float value.
// If texIndex is -1 then fall-back to the supplied default value.
float matGetSample1f(float2 uv, float defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData){
	if( texIndex == -1 ){
		return defaultValue;
	}

	return texGetSample1f( uv, texIndex, texMeta, texData );
}

// Apply normal map to intersection normal.
float3 matGetNormalSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData){
	// Generate tangent, bi-tangent vectors
	float3 u,v;
	TANGENT_VECTORS(normal, u, v);

	// Sample normal map and convert it into the [-1, 1] range. 
	// R, G components encode the range [-1, 1] into a value [0, 255]
	// B component encodes the range [0, 1] into [128, 255]
	float3 sample = (texGetSample3f( uv, texIndex, texMeta, texData ) * 2.0f) - 1.0f;
	return normalize(u * sample.x + v * sample.y + 0.5f * normal * sample.z);
}

// Apply bump map to intersection normal.
float3 matGetBumpSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData){
	// Generate tangent, bi-tangent vectors
	float3 u,v;
	TANGENT_VECTORS(normal, u, v);

	float3 sample = (texGetBumpSample3f( uv, texIndex, texMeta, texData ) * 2.0f) - 1.0f;
	return normalize(u * sample.x + v * sample.y + normal * sample.z);
}
#endif
