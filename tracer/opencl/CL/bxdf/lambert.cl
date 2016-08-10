#ifndef BXDF_LAMBERT_CL
#define BXDF_LAMBERT_CL

float3 lambertDiffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf);
float lambertDiffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir);
float3 lambertDiffuseEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir);

// Sample ideal diffuse (lambert) surface:
//
// BXDF = F * kval  / PI
// PDF = cos(theta) / PI
float3 lambertDiffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf){
	*rayOutDir = rayGetCosWeightedHemisphereSample(surface->normal, randSample);	
	
	*pdf = dot(surface->normal, *rayOutDir) * C_1_PI;

	float3 kd = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	
	return matNode->fresnel * kd * C_1_PI;
}

// Get PDF for lambert surface given a pre-calculated bounce ray.
// PDF = cos(theta) / PI
float lambertDiffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir){
	return dot(surface->normal, rayOutDir) * C_1_PI;
}

// Evaluate BXDF for lambert surface given a pre-calculated bounce ray.
float3 lambertDiffuseEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir){
	float3 kd = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	return matNode->fresnel * kd * C_1_PI;
}
#endif
