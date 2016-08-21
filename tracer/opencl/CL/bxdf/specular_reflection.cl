#ifndef BXDF_SPECULAR_REFLECTION_CL
#define BXDF_SPECULAR_REFLECTION_CL

float3 idealSpecularSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 rayInDir, float3 *rayOutDir, float *pdf);
float idealSpecularPdf();
float3 idealSpecularEval();

// Sample ideal diffuse (idealSpecular) surface:
//
// BXDF = kval / cosI
// PDF = 1
float3 idealSpecularSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 rayInDir, float3 *rayOutDir, float *pdf){
	float cosI = dot(rayInDir, surface->normal);
	
	// When rendering surfaces like diamonds we need to handle internal reflections.
	// We need to flip the normal if we are hitting the surface from the inside
	float3 surfNormal = surface->normal;
	if(cosI < 0.0f ){
		cosI = -cosI;
		surfNormal = -surfNormal;
	}

	// To generate the out ray we need to reflect the input ray around the normal
	// rayInDir points *away* from the surface so we need to flip the sign of the 
	// reflection formula: I - 2*dot(I,N) * N
	*rayOutDir = 2.0f * cosI * surfNormal - rayInDir;

	*pdf = 1.0f;
	
	float3 ks = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	return cosI != 0.0f ? ks / cosI : 0.0f;
}

// Get PDF for idealSpecular surface given a pre-calculated bounce ray.
// PDF = 0 unless rayOutDir = reflect(rayInDir, surface->normal)
float idealSpecularPdf(){
	// Cheat and always return 0
	return 0.0f;
}

// Evaluate BXDF for idealSpecular surface given a pre-calculated bounce ray.
// Similar to the PDF evaluator above this is always 0 unless we use the 
// reflected ray.
float3 idealSpecularEval(){
	return (float3)(0.0f, 0.0f, 0.0f);
}
#endif
