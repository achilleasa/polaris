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
void printMaterialNode(MaterialNode *node);

// Traverse the layered material tree for this surface and select a leaf node
void matSelectNode(Surface *surface, MaterialNode *selectedMaterial, __global MaterialNode* materialNodes ){
	__global MaterialNode* node = materialNodes + surface->matNodeIndex;
	while( node->isNode ){
		node = materialNodes + node->leftChild;
	}

	*selectedMaterial = *node;
	selectedMaterial->fresnel = 1.0f;
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
