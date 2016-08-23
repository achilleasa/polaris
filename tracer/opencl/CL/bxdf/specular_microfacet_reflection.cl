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

	float iDotN = dot(inRayDir, surface->normal);
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

	// Eval fresnel term
	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->ior, matNode->iorTex, texMeta, texData);
	float eta = etaI / etaT;
	float r0 = ((eta - 1.0f) * (eta - 1.0f)) / ((eta + 1.0f) * (eta + 1.0f));
	float c = 1.0f - fabs(iDotN);
	float c1 = c * c;
	float f = r0 + (1.0f - r0)*c1*c1*c;
	
	float3 h = normalize(inRayDir + outRayDir);

	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

	// Eval sample (equation 20)
	float denom = 4.0f * iDotN * oDotN;
	return denom > 0.0f 
		? matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData) * f * d * g / denom 
		: 0.0f;
}

#endif
