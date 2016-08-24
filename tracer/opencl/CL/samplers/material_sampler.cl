#ifndef MATERIAL_SAMPLER_CL
#define MATERIAL_SAMPLER_CL

#define MAT_BLEND_FUNC_MIX 0
#define MAT_BLEND_FUNC_FRESNEL 1

#define MAT_FLAG_IS_NODE        1 << 0
#define MAT_FLAG_USE_NORMAL_MAP 1 << 1
#define MAT_FLAG_USE_BUMP_MAP   1 << 2

void matSelectNode(Surface *surface, float3 inRayDir, MaterialNode *selectedMaterial, __global MaterialNode* materialNodes, uint2 *rndState, __global TextureMetadata *texMeta, __global uchar *texData );
float3 matGetSample3f(float2 uv, float3 defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);
float matGetSample1f(float2 uv, float defaultValue, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);
float3 matGetNormalSample3f(uint matFlags, float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData);

// Traverse the layered material tree for this surface and select a leaf node
void matSelectNode(Surface *surface, float3 inRayDir, MaterialNode *selectedMaterial, __global MaterialNode* materialNodes, uint2 *rndState, __global TextureMetadata *texMeta, __global uchar *texData ){
	__global MaterialNode* node = materialNodes + surface->matNodeIndex;
	float2 sample;
	float nval;
	while( (node->flags & MAT_FLAG_IS_NODE) != 0 ){
		sample = randomGetSample2f(rndState);
		nval = matGetSample1f(surface->uv, node->nval, node->nvalTex, texMeta, texData);

		if(node->blendFunc == MAT_BLEND_FUNC_FRESNEL){
			// inRayDir point *away* from surface
			float iDotN = dot(inRayDir, surface->normal);

			float etaI = 1.0f;
			float etaT = nval;

			// If we are exiting the area we need to flip the normal and incident/transmission etas 
			if( iDotN < 0.0f ) {
				etaI = etaT;
				etaT = 1.0f;
			}

			// The following formulas are taken from: https://en.wikipedia.org/wiki/Refractive_index
			float eta = etaI / etaT;
			float sqISinI = 1.0f - iDotN * iDotN;
			float sqSinT = eta * eta * sqISinI;

			// Calculate the fresnel value if TIR does not occur (TIR when sqSinT > 1.0)
			nval = 1.0f;
			if( sqSinT <= 1.0f ){
				nval = fresnelForDielectric(eta, iDotN);
			}
		}

		// nval contains the selection probability for the left child
		node = materialNodes + ((sample.x < nval) ? node->leftChild : node->rightChild);
	}

	*selectedMaterial = *node;
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
float3 matGetNormalSample3f(uint matFlags, float3 normal, float2 uv, int texIndex, __global TextureMetadata *texMeta, __global uchar* texData){
	// Generate tangent, bi-tangent vectors
	float3 u,v;
	TANGENT_VECTORS(normal, u, v);

	// Depending on the surface flags we treat this either as a bump map or as a normal map
	float3 sample;
	if( (matFlags & MAT_FLAG_USE_BUMP_MAP) != 0 ){
		sample = (texGetBumpSample3f( uv, texIndex, texMeta, texData ) * 2.0f) - 1.0f;
		return normalize(u * sample.x + v * sample.y + normal * sample.z);
	} else {
		// Sample normal map and convert it into the [-1, 1] range. 
		// R, G components encode the range [-1, 1] into a value [0, 255]
		// B component encodes the range [0, 1] into [128, 255]
		sample = (texGetSample3f( uv, texIndex, texMeta, texData ) * 2.0f) - 1.0f;
		return normalize(u * sample.x + v * sample.y + 0.5f * normal * sample.z);
	}
}

#endif
