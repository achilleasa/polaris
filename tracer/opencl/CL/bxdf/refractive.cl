#ifndef BXDF_REFRACTIVE_CL
#define BXDF_REFRACTIVE_CL

float3 refractiveSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 rayInDir, float3 *rayOutDir, float *pdf);
float refractivePdf();
float3 refractiveEval();

// Sample ideal diffuse (refractive) surface:
//
// BXDF = 1 / cos(theta)
// PDF = 1
float3 refractiveSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 rayInDir, float3 *rayOutDir, float *pdf){
	*pdf = 1.0f;

	// rayInDir point *away* from surface
	float cosI = dot(rayInDir, surface->normal);
	bool entering = cosI > 0.0f;

	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData);
	float3 surfNormal = surface->normal;

	// If we are exiting the area we need to flip the normal and incident/transmission etas 
	if( !entering ) {
		etaI = etaT;
		etaT = 1.0f;
		surfNormal = -surfNormal;
		cosI = -cosI;
	}

	// The following formulas are taken from: https://en.wikipedia.org/wiki/Refractive_index
	float eta = etaI / etaT;
	float sqISinI = 1.0f - cosI * cosI;
	float sqSinT = eta * eta * sqISinI;

	// According to Snell's law:
	// etaI * sinI = etaT * sinT <=> sinT = eta * sinI
	// If sinT > 1 then the ray undergoes total internal refleciton
	if( sqSinT > 1.0f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}

	// cos = 1 - sin*sin
    float cosTransimission = native_sqrt(max(0.0f, 1.0f - sqSinT));
	float sinT = native_sqrt(max(0.0f, sqSinT));
	*rayOutDir = normalize(normalize(surfNormal * cosI - rayInDir) * sinT - surfNormal * cosTransimission);

	float3 ks = eta * eta * matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	return cosTransimission > 0.0f ? ks / cosTransimission: 0.0f;
}

// Get PDF for refractive surface given a pre-calculated bounce ray.
// PDF = 0 unless rayOutDir = refract(rayInDir, surface->normal)
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
