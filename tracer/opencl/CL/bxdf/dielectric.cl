#ifndef BXDF_DIELECTRIC_CL
#define BXDF_DIELECTRIC_CL

float3 dielecticSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float dielectricPdf(Surface *surface, MaterialNode *matNode, float3 inRayDir, float3 outRayDir);
float3 dielectricEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Ideal dielectric
//
// BXDF = 1 / cos(theta)
// PDF = 1
float3 dielecticSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	float iDotN = dot(inRayDir, surface->normal);
	float etaI = matNode->extIOR;
	float etaT = matNode->intIOR;

	// If hitting from the inside we need to swap the eta 
	if( iDotN < 0.0f ){
		float tmp = etaI;
		etaI = etaT;
		etaT = tmp;
	}

	float eta = etaI / etaT;

	// Calculate fresnel 
	float f = fresnelForDielectric(etaI, etaT, iDotN);
	
	float3 kVal;
	float cosTSq = 1.0f + eta * (iDotN * iDotN - 1.0f);

	// Based on the fresnel value randomly sample the reflection ray.
	// In the case where the ray undergoes total internal reflection we 
	// always pick the reflection ray
	if( cosTSq <= 0.0f|| randSample.x <= f ){
		*outRayDir = -sign(iDotN) * 2.0f * iDotN * surface->normal - inRayDir;
		kVal = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);
		*pdf = cosTSq <= 0.0f ? 1.0 : f;
	} else {
		*outRayDir = (eta * iDotN - sign(iDotN)*sqrt(cosTSq))*surface->normal - eta * inRayDir;
		kVal = eta * eta * matGetSample3f(surface->uv, matNode->transmittance, matNode->transmittanceTex, texMeta, texData);
		*pdf = 1.0f - f;
	}
	
	return iDotN != 0.0f ? *pdf * kVal / fabs(iDotN): 0.0f;
}

// Get PDF for dielectic surface given a pre-calculated bounce ray.
// PDF = 0 unless outRayDir = refract(inRayDir, surface->normal)
float dielectricPdf(Surface *surface, MaterialNode *matNode, float3 inRayDir, float3 outRayDir){
	// Cheat and always return 0
	return 0.0f;
}

// Evaluate BXDF for dielectic surface given a pre-calculated bounce ray.
// Similar to the PDF evaluator above this is always 0 unless we use the 
// refracted ray.
float3 dielectricEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	return (float3)(0.0f, 0.0f, 0.0f);
}
#endif
