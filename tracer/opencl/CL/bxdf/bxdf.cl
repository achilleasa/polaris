#ifndef BXDF_CL
#define BXDF_CL

#include "diffuse.cl"
#include "conductor.cl"
#include "dielectric.cl"
#include "rough_conductor.cl"
#include "rough_dielectric.cl"

#ifndef BXDF_INVALID
	#define BXDF_INVALID 0
#endif
#define BXDF_TYPE_EMISSIVE         1 << 1
#define BXDF_TYPE_DIFFUSE          1 << 2
#define BXDF_TYPE_CONDUCTOR        1 << 3
#define BXDF_TYPE_ROUGHT_CONDUCTOR 1 << 4
#define BXDF_TYPE_DIELECTRIC       1 << 5
#define BXDF_TYPE_ROUGH_DIELECTRIC 1 << 6

#define BXDF_IS_EMISSIVE(t) (t == BXDF_TYPE_EMISSIVE)
#define BXDF_IS_SINGULAR(t) ((t & (BXDF_TYPE_CONDUCTOR | BXDF_TYPE_DIELECTRIC)) != 0)

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
	switch(matNode->type){
		case BXDF_TYPE_DIFFUSE:
			return diffuseSample(surface, matNode, texMeta, texData, randSample, outRayDir, pdf);
		case BXDF_TYPE_CONDUCTOR:
			return conductorSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_DIELECTRIC:
			return dielecticSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_ROUGHT_CONDUCTOR:
			return roughConductorSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_ROUGH_DIELECTRIC:
			return roughDielectricSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
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
	switch(matNode->type){
		case BXDF_TYPE_DIFFUSE:
			return diffusePdf(surface, matNode, outRayDir);
		case BXDF_TYPE_CONDUCTOR:
			return conductorPdf(surface, inRayDir, outRayDir);
		case BXDF_TYPE_DIELECTRIC:
			return dielectricPdf(surface, matNode, inRayDir, outRayDir);
		case BXDF_TYPE_ROUGHT_CONDUCTOR:
			return roughConductorPdf(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_ROUGH_DIELECTRIC:
			return roughDielectricPdf(surface, matNode, texMeta, texData, inRayDir, outRayDir);
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
	switch(matNode->type){
		case BXDF_TYPE_DIFFUSE:
			return diffuseEval(surface, matNode, texMeta, texData, outRayDir);
		case BXDF_TYPE_CONDUCTOR:
			return conductorEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_DIELECTRIC:
			return dielectricEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_ROUGHT_CONDUCTOR:
			return roughConductorEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_ROUGH_DIELECTRIC:
			return roughDielectricEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
	}

	return (float3)(0.0f, 0.0f, 0.0f);
} 


#endif
