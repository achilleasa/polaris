#ifndef BXDF_DIFFUSE_CL
#define BXDF_DIFFUSE_CL

float3 diffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf);
float diffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir);
float3 diffuseEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir);

// Sample ideal diffuse (lambert) surface:
//
// BXDF = reflectance  / PI
// PDF = cos(theta) / PI
float3 diffuseSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *rayOutDir, float *pdf){
	*rayOutDir = cosWeightedHemisphereGetSample(surface->normal, randSample);	
	
	*pdf = dot(surface->normal, *rayOutDir) * C_1_PI;

	float3 kd = matGetSample3f(surface->uv, matNode->reflectance, matNode->reflectanceTex, texMeta, texData);
	
	return kd * C_1_PI;
}

// Get PDF for lambert surface given a pre-calculated bounce ray.
// PDF = cos(theta) / PI
float diffusePdf(Surface *surface, MaterialNode *matNode, float3 rayOutDir){
	return dot(surface->normal, rayOutDir) * C_1_PI;
}

// Evaluate BXDF for lambert surface given a pre-calculated bounce ray.
float3 diffuseEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 rayOutDir){
	float3 kd = matGetSample3f(surface->uv, matNode->reflectance, matNode->reflectanceTex, texMeta, texData);
	return kd * C_1_PI;
}
#endif
