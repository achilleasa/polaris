#ifndef MATERIAL_SAMPLER_CL
#define MATERIAL_SAMPLER_CL

#define MAT_BLEND_FUNC_MIX 0
#define MAT_BLEND_FUNC_FRESNEL 1

#define MAT_BXDF_DIFFUSE (1 << 0)
#define MAT_BXDF_SPECULAR (1 << 1)
#define MAT_BXDF_REFRACTIVE (1 << 2)
#define MAT_BXDF_EMISSIVE (1 << 3)

#define MAT_IS_EMISSIVE(matNode) (matNode.bxdfType == MAT_BXDF_EMISSIVE)

void matSelectNode(Surface *surface, MaterialNode *selectedMaterial, __global MaterialNode* materialNodes );
float3 matGetSample3f(float2 uv, float3 defaultValue, int texIndex, __global TextureMetadata *metadata, __global uchar* texData);
float matGetSample1f(float2 uv, float defaultValue, int texIndex, __global TextureMetadata *metadata, __global uchar* texData);
float3 matGetNormalSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* texData);
void printMaterialNode(MaterialNode *node);

// Traverse the layered material tree for this surface and select a leaf node
void matSelectNode(Surface *surface, MaterialNode *selectedMaterial, __global MaterialNode* materialNodes ){
	__global MaterialNode* node = materialNodes + surface->matNodeIndex;
	while( node->isNode ){
		node = materialNodes + node->leftChild;
	}

	*selectedMaterial = *node;
}

// Sample texture using the supplied uv coordinates and return a float3 vector. 
// If texIndex is -1 then fall-back to the supplied default value.
float3 matGetSample3f(float2 uv, float3 defaultValue, int texIndex, __global TextureMetadata *metadata, __global uchar* texData){
	if( texIndex == -1 ){
		return defaultValue;
	}

	return texGetSample3f( uv, texIndex, metadata, texData );
}

// Sample texture using the supplied uv coordinates and return a float value.
// If texIndex is -1 then fall-back to the supplied default value.
float matGetSample1f(float2 uv, float defaultValue, int texIndex, __global TextureMetadata *metadata, __global uchar* texData){
	if( texIndex == -1 ){
		return defaultValue;
	}

	return texGetSample1f( uv, texIndex, metadata, texData );
}

// Apply normal map to intersection normal.
float3 matGetNormalSample3f(float3 normal, float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* texData){
	// Generate tangent, bi-tangent vectors
	float3 u = normalize(cross((fabs(normal.x) > .1f ? (float3)(0.0f, 1.0f, 0.0f) : (float3)(1.0f, 0.0f, 0.0f)), normal));
	float3 v = cross(normal,u);

	// Sample normal map and convert it into the [-1, 1] range. 
	// R, G components encode the range [-1, 1] into a value [0, 255]
	// B component encodes the range [0, 1] into [128, 255]
	float3 sample = (texGetSample3f( uv, texIndex, metadata, texData ) * 2.0f) - 1.0f;
	return normalize(u * sample.x + v * sample.y + 0.5f * normal * sample.z);
}

void printMaterialNode(MaterialNode *node){
	if( node->isNode ){
		printf("[tid: %03d] material intermediate node (blendFunc: %d, lProb/IOR: %f, left: %d, right: %d)\n", 
				get_global_id(0),
				node->blendFunc,
				node->leftChildProb,
				node->leftChild,
				node->rightChild
		);
	} else {
		printf("[tid: %03d] material leaf node (bxdfType: %d, kval: %2.2v4hlf, nval: %f, kValTex: %d, normalTex: %d, nvalTex: %d)\n",
				get_global_id(0),
				node->bxdfType,
				node->kval,
				node->nval,
				node->kvalTex,
				node->normalTex,
				node->nvalTex
		);
	}
}

#endif
