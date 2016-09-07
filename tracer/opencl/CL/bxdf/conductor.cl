#ifndef BXDF_CONDUCTOR_REFLECTION_CL
#define BXDF_CONDUCTOR_REFLECTION_CL

float3 conductorSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float conductorPdf(Surface *surface, float3 inRayDir, float3 outRayDir);
float3 conductorEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Sample conductor bxdf
//
// BXDF = kval / cosI
// PDF = 1
float3 conductorSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	float iDotN = dot(inRayDir, surface->normal);
	
	// To generate the out ray we need to reflect the input ray around the normal
	// inRayDir points *away* from the surface so we need to flip the sign of the 
	// reflection formula: I - 2*dot(I,N) * N
	*outRayDir = 2.0f * iDotN * surface->normal - inRayDir;

	*pdf = 1.0f;

	// Calculate fresnel unless no IOR is specified
	float f = matNode->intIOR != 0.0f 
		? fresnelForDielectric(matNode->extIOR, matNode->intIOR, iDotN)
		: 1.0f;

	float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);
	return iDotN != 0.0f ? f * ks / iDotN : 0.0f;
}

// Get PDF for conductor surface given a pre-calculated bounce ray.
// PDF = 0 unless outRayDir = reflect(inRayDir, surface->normal)
float conductorPdf(Surface *surface, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	float3 expOutDir = 2.0f * iDotN * surface->normal - inRayDir;
	
	// Allow a small margin of error when matching the two vectors
	float expDot = dot(expOutDir, outRayDir);
	return expDot >= 0.0f && expDot <= 0.001f ? 1.0f : 0.0f;
}

// Evaluate BXDF for conductor surface given a pre-calculated bounce ray.
// Similar to the PDF evaluator above this is always 0 unless we use the 
// reflected ray with a small error margin.
float3 conductorEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	float3 expOutDir = 2.0f * iDotN * surface->normal - inRayDir;
	
	// Allow a small margin of error when matching the two vectors
	float expDot = dot(expOutDir, outRayDir);
	if(expDot < 0.0f || expDot > 0.001f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}

	// Calculate fresnel unless no IOR is specified
	float f = matNode->intIOR != 0.0f 
		? fresnelForDielectric(matNode->extIOR, matNode->intIOR, iDotN)
		: 1.0f;

	float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);
	return iDotN != 0.0f ? f * ks / iDotN : 0.0f;
}
#endif
