#ifndef BXDF_CL
#define BXDF_CL

#include "lambert.cl"

#define BXDF_TYPE_DIFFUSE 1 << 0
#define BXDF_TYPE_SPECULAR 1 << 1
#define BXDF_TYPE_REFRACTIVE 1 << 2
#define BXDF_TYPE_EMISSIVE 1 << 3

float3 bxdfGetSample(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float bxdfGetPdf(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir );
float3 bxdfValue(Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir); 

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
	}

	return 0.0f;
}

// Given a pre-calculated bounce ray evaluate the BXDF for the surface based on 
// the surface material, inRayDir and outRayDir
float3 bxdfValue(
		Surface *surface, 
		MaterialNode *matNode, 
		__global TextureMetadata *texMeta, 
		__global uchar *texData, 
		float3 inRayDir,
		float3 outRayDir 
){
	switch(matNode->bxdfType){
		case BXDF_TYPE_DIFFUSE:
			return lambertDiffuseValue(surface, matNode, texMeta, texData, outRayDir);
	}

	return (float3)(0.0f, 0.0f, 0.0f);

} 


#endif
