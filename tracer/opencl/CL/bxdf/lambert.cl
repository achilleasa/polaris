#ifndef BXDF_LAMBERT_CL
#define BXDF_LAMBERT_CL

float3 lambertDiffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf);
float lambertDiffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir);
float3 lambertDiffuseValue(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir);

// Sample ideal diffuse (lambert) surface:
//
// BXDF = F * kval * cos(theta) / PI
// PDF = cos(theta) / PI
//
// Note: We pre-multiply the BXDF with cos(theta) as we calculate it anyway in the function.
float3 lambertDiffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf){
	*rayOutDir = rayGetCosWeightedHemisphereSample(surface->normal, randSample);	

	float cosTheta = dot(surface->normal, *rayOutDir);

	*pdf = cosTheta * C_1_PI;

	// We also need to multiply by the matNode fresnel which will be set to the 
	// probability of selecting this matNode (if this is a layered material)
	return matNode->fresnel * matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData) * cosTheta * C_1_PI;
}

// Get PDF for lambert surface given a pre-calculated bounce ray.
// PDF = cos(theta) / PI
float lambertDiffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir){
	return dot(surface->normal, rayOutDir) * C_1_PI;
}

// Evaluate BXDF for lambert surface given a pre-calculated bounce ray.
float3 lambertDiffuseValue(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir){
	return matNode->fresnel * matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData) * dot(surface->normal, rayOutDir) * C_1_PI;
}
#endif
