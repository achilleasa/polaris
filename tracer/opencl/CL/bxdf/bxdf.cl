#ifndef BXDF_CL
#define BXDF_CL

#include "lambert.cl"
#include "specular_reflection.cl"
#include "specular_transmission.cl"
#include "specular_microfacet_reflection.cl"
#include "specular_microfacet_transmission.cl"

#define BXDF_TYPE_EMISSIVE 1 << 0
#define BXDF_TYPE_DIFFUSE 1 << 1
#define BXDF_TYPE_SPECULAR_REFLECTION 1 << 2
#define BXDF_TYPE_SPECULAR_TRANSMISSION 1 << 3
#define BXDF_TYPE_SPECULAR_MICROFACET_REFLECTION 1 << 4
#define BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION 1 << 5

#define BXDF_IS_TRANSMISSION(t) ((t & (BXDF_TYPE_SPECULAR_TRANSMISSION | BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION)) != 0)
#define BXDF_IS_EMISSIVE(t) (t == BXDF_TYPE_EMISSIVE)

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
		case BXDF_TYPE_SPECULAR_REFLECTION:
			return idealSpecularSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_SPECULAR_TRANSMISSION:
			return refractiveSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_SPECULAR_MICROFACET_REFLECTION:
			return microfacetReflectionSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
		case BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION:
			return microfacetTransmissionSample(surface, matNode, texMeta, texData, randSample, inRayDir, outRayDir, pdf);
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
		case BXDF_TYPE_SPECULAR_REFLECTION:
			return idealSpecularPdf();
		case BXDF_TYPE_SPECULAR_TRANSMISSION:
			return refractivePdf();
		case BXDF_TYPE_SPECULAR_MICROFACET_REFLECTION:
			return microfacetReflectionPdf(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION:
			return microfacetTransmissionPdf(surface, matNode, texMeta, texData, inRayDir, outRayDir);
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
		case BXDF_TYPE_SPECULAR_REFLECTION:
			return idealSpecularEval();
		case BXDF_TYPE_SPECULAR_TRANSMISSION:
			return refractiveEval();
		case BXDF_TYPE_SPECULAR_MICROFACET_REFLECTION:
			return microfacetReflectionEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
		case BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION:
			return microfacetTransmissionEval(surface, matNode, texMeta, texData, inRayDir, outRayDir);
	}

	return (float3)(0.0f, 0.0f, 0.0f);
} 


#endif
