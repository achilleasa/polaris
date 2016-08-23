#ifndef BXDF_SPECULAR_TRANSMISSION_CL
#define BXDF_SPECULAR_TRANSMISSION_CL

float3 refractiveSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float refractivePdf();
float3 refractiveEval();

// Sample ideal diffuse (refractive) surface:
//
// BXDF = 1 / cos(theta)
// PDF = 1
float3 refractiveSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	float iDotN = dot(inRayDir, surface->normal);

	// Easy case. I perpendicular to N
	if( iDotN == 0.0f ){
		*pdf = 0.0f;
		return (float3)(0.0f, 0.0f, 0.0f);
	}

	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->ior, matNode->iorTex, texMeta, texData);
	if(iDotN < 0.0f){
		// We are exiting the surface so we need to flip the eta bits
		etaI = etaT;
		etaT = 1.0f;
	}
	float eta = etaI / etaT;
	
	// Refract I over n to get O
	float cosTSq = 1.0f + eta * eta * (iDotN * iDotN - 1.0f);
	if( cosTSq < 0.0f ){
		*pdf = 0.0f;
		return (float3)(0.0f, 0.0f, 0.0f);
	}
	*outRayDir = (eta * iDotN - sign(iDotN)*sqrt(cosTSq))*surface->normal - eta * inRayDir;
	*pdf = 1.0f;

	float3 tf = eta * eta * matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	return iDotN != 0.0f ? tf / fabs(iDotN): 0.0f;
}

// Get PDF for refractive surface given a pre-calculated bounce ray.
// PDF = 0 unless outRayDir = refract(inRayDir, surface->normal)
float refractivePdf(){
	// Cheat and always return 0
	return 0.0f;
}

// Evaluate BXDF for refractive surface given a pre-calculated bounce ray.
// Similar to the PDF evaluator above this is always 0 unless we use the 
// refracted ray.
float3 refractiveEval(){
	return (float3)(0.0f, 0.0f, 0.0f);
}
#endif
