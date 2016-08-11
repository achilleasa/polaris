#ifndef BXDF_CL
#define BXDF_CL

#include "lambert.cl"
#include "ideal_specular.cl"
#include "refractive.cl"

#define BXDF_TYPE_DIFFUSE 1 << 0
#define BXDF_TYPE_SPECULAR 1 << 1
#define BXDF_TYPE_REFRACTIVE 1 << 2
#define BXDF_TYPE_EMISSIVE 1 << 3

// Returns true if BXDF models a transmission
#define BXDF_IS_TRANSMISSION(t) (t == BXDF_TYPE_REFRACTIVE)

float3 bxdfGetSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float bxdfGetPdf(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir );
float3 bxdfEval(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir); 

// Sample BXDF for this surface and material and also generate a bounce ray
// with a PDF that approximates the material BXDF.
float3 bxdfGetSample(
		Surface *surface, 
		MaterialNode *matNode, 
		__global TextureMetadata *texMeta, 
		__global uchar *texData, 
		float2 randSample, 
		float3 inRayDir,
		float3 *outRayDir, 
		float *pdf
){
	switch(matNode->bxdfType){
		case BXDF_TYPE_DIFFUSE:
			return lambertDiffuseSample(surface, matNode, texMeta, texData, randSample, outRayDir, pdf);
		case BXDF_TYPE_SPECULAR:
			return idealSpecularSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_REFRACTIVE:
			return refractiveSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
	}

	return (float3)(0.0f, 0.0f, 0.0f);

} 

// Get PDF for selecting a pre-calculated bounce ray based on the surface BXDF
float bxdfGetPdf(
		Surface *surface, 
		MaterialNode *matNode, 
		__global TextureMetadata *texMeta, 
		__global uchar *texData, 
		float3 inRayDir,
		float3 outRayDir 
){
	switch(matNode->bxdfType){
		case BXDF_TYPE_DIFFUSE:
			return lambertDiffusePdf(surface, matNode, outRayDir);
		case BXDF_TYPE_SPECULAR:
			return idealSpecularPdf();
		case BXDF_TYPE_REFRACTIVE:
			return refractivePdf();
	}

	return 0.0f;
}

// Given a pre-calculated bounce ray evaluate the BXDF for the surface based on 
// the surface material, inRayDir and outRayDir
float3 bxdfEval(
		Surface *surface, 
		MaterialNode *matNode, 
		__global TextureMetadata *texMeta, 
		__global uchar *texData, 
		float3 inRayDir,
		float3 outRayDir 
){
	switch(matNode->bxdfType){
		case BXDF_TYPE_DIFFUSE:
			return lambertDiffuseEval(surface, matNode, texMeta, texData, outRayDir);
		case BXDF_TYPE_SPECULAR:
			return idealSpecularEval();
		case BXDF_TYPE_REFRACTIVE:
			return refractiveEval();
	}

	return (float3)(0.0f, 0.0f, 0.0f);

} 


#endif
