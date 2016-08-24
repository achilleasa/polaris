#ifndef BXDF_SPECULAR_MICROFACET_REFLECTION_CL
#define BXDF_SPECULAR_MICROFACET_REFLECTION_CL

float3 microfacetReflectionSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float microfacetReflectionPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);
float3 microfacetReflectionEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Sample microfacet surface
float3 microfacetReflectionSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	// Sample GGX distribution to get halfway vector
	float3 h = ggxGetSample(roughness, inRayDir, surface->normal, randSample);
	
	// Reflect I over h to get O
	*outRayDir = 2.0f * dot(inRayDir, h) * h - inRayDir;
	*pdf = ggxGetReflectionPdf(roughness, inRayDir, *outRayDir, surface->normal, h);

	// Eval sample
	return microfacetReflectionEval(surface, matNode, texMeta, texData, inRayDir, *outRayDir);
}

// Get PDF given an outbound ray
float microfacetReflectionPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float3 h = normalize(inRayDir + outRayDir);

	return ggxGetReflectionPdf(roughness, inRayDir, outRayDir, surface->normal, h);
}

// Evaluate microfacet BXDF for the selected outgoing ray.
float3 microfacetReflectionEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(outRayDir, surface->normal);

	// Eval fresnel term (use ks as r0)
	float3 f = fresnelForDielectricFromSpecular(matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData), iDotN);
	
	float3 h = normalize(inRayDir + outRayDir);

	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

	// Eval sample (equation 20)
	float denom = 4.0f * iDotN * oDotN;
	return denom > 0.0f ?  f * d * g / denom : 0.0f;
}

#endif
