#ifndef BXDF_ROUGH_CONDUCTOR_CL
#define BXDF_ROUGH_CONDUCTOR_CL

float3 roughConductorSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float roughConductorPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);
float3 roughConductorEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Sample microfacet surface
float3 roughConductorSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);

	// Sample GGX distribution to get halfway vector
	float3 h = ggxGetSample(roughness, inRayDir, surface->normal, randSample);

	// Reflect I over h to get O
	*outRayDir = 2.0f * dot(inRayDir, h) * h - inRayDir;
	*pdf = ggxGetReflectionPdf(roughness, inRayDir, *outRayDir, surface->normal, h);

	// Eval sample
	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(*outRayDir, surface->normal);
	h = normalize(inRayDir + *outRayDir);

	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, *outRayDir, surface->normal, h);

	// Calculate fresnel unless no IOR is specified
	float f = matNode->intIOR != 0.0f
		? fresnelForDielectric(matNode->extIOR, matNode->intIOR, iDotN)
		: 1.0f;

	// Eval sample (equation 20)
	float denom = 4.0f * iDotN * oDotN;
	return denom > 0.0f ?  ks * f * d * g / denom : 0.0f;
}

// Get PDF given an outbound ray
float roughConductorPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float3 h = normalize(inRayDir + outRayDir);

	return ggxGetReflectionPdf(roughness, inRayDir, outRayDir, surface->normal, h);
}

// Evaluate microfacet BXDF for the selected outgoing ray.
float3 roughConductorEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);

	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(outRayDir, surface->normal);

	// Calculate fresnel unless no IOR is specified
	float f = matNode->intIOR != 0.0f
		? fresnelForDielectric(matNode->extIOR, matNode->intIOR, iDotN)
		: 1.0f;

	float3 h = normalize(inRayDir + outRayDir);

	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

	// Eval sample (equation 20)
	float denom = 4.0f * iDotN * oDotN;
	return denom > 0.0f ?  ks * f * d * g / denom : 0.0f;
}

#endif
