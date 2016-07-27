#ifndef EMISSIVE_SAMPLER_CL
#define EMISSIVE_SAMPLER_CL

#define EMISSIVE_TYPE_AREA_LIGHT 0
#define EMISSIVE_TYPE_ENVIRONMENT_LIGHT 1

float3 environmentLightGetSample( Surface *surface, __global Emissive *emissive, __global MaterialNode *materialNodes, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *outRayDir, float *pdf, float *distToEmissive); 
float environmentLightGetPdf( Surface *surface, __global Emissive *emissive, float3 outRayDir);
float3 areaLightGetSample( Surface *surface, __global Emissive *emissive, __global float4 *vertices, __global float4 *normals, __global float2 *uv, __global MaterialNode *materialNodes, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *outRayDir, float *pdf, float *distToEmissive);
float areaLightGetPdf( Surface *surface, __global Emissive *emissive, __global float4 *vertices, __global float4 *normals, float3 outRayDir);

float3 emissiveGetSample( Surface *surface, __global Emissive *emissive, __global float4 *vertices, __global float4 *normals, __global float2 *uv, __global MaterialNode *materialNodes, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 *outRayDir, float *pdf, float *distToEmissive);
float emissiveGetPdf( Surface *surface, __global Emissive *emissive, __global float4 *vertices, __global float4 *normals, float3 outRayDir);
uint emissiveSelect( const int numLights, float randSample, float *pdf);

float3 environmentLightGetSample(
		Surface *surface,
		__global Emissive *emissive,
		__global MaterialNode *materialNodes,
		__global TextureMetadata *texMeta,
		__global uchar *texData,
		float2 randSample,
		float3 *outRayDir,
		float *pdf,
		float *distToEmissive
		){

	*outRayDir = rayGetCosWeightedHemisphereSample(surface->normal, randSample);
	float cosTheta = max(0.0f, dot(surface->normal, *outRayDir));
	*pdf = cosTheta * C_1_PI;
	*distToEmissive = FLT_MAX;

	// Convert ray direction vector into spherical UV and use that to sample the env map
	float2 uv = rayToLatLongUV(*outRayDir);
	MaterialNode matNode = materialNodes[emissive->matNodeIndex];

	return matGetSample3f(uv, matNode.kval, matNode.kvalTex, texMeta, texData) * cosTheta * C_1_PI;
}

float environmentLightGetPdf(
		Surface *surface,
		__global Emissive *emissive,
		float3 outRayDir
		){

	// We use the same formula as for lambert shading: cos(theta) / PI
	return max(0.0f, dot(surface->normal, outRayDir) * C_1_PI);
}

// Generate a out ray direction towards a random point on the emissive primitive
// and return a emission material sample from that point.
float3 areaLightGetSample(
		Surface *surface,
		__global Emissive *emissive,
		__global float4 *vertices, 
		__global float4 *normals,
		__global float2 *uv,
		__global MaterialNode *materialNodes,
		__global TextureMetadata *texMeta,
		__global uchar *texData,
		float2 randSample,
		float3 *outRayDir,
		float *pdf,
		float *distToEmissive
		){

	// Select a random point on the emissive and get its *world* xyz/normal coordinates
	float3 wuv = (float3)(1.0f - (randSample.x + randSample.y), randSample);
	int offset = emissive->triIndex * 3;

	float3 emissivePoint = mul4x1(
			(wuv.x * vertices[offset] + wuv.y * vertices[offset+1] + wuv.z * vertices[offset+2]).xyz,
		    emissive->transformMat0,
			emissive->transformMat1,
			emissive->transformMat2,
			emissive->transformMat3
	);

	float3 emissiveNormal = mul4x1(
			(wuv.x * normals[offset] + wuv.y * normals[offset+1] + wuv.z * normals[offset+2]).xyz,
		    emissive->transformMat0,
			emissive->transformMat1,
			emissive->transformMat2,
			emissive->transformMat3
	);

	float2 emissiveUV = wuv.x * uv[offset] + 
		        wuv.y * uv[offset+1] + 
				wuv.z * uv[offset+2];


	// The selection PDF depends on the projected solid angle given by:
	// dot(outRayDir, emissiveNormal) * area / distToEmissive * distToEmissive
	float3 emissiveRay = emissivePoint - surface->point;
	float squaredDistToLight = dot(emissiveRay, emissiveRay);
	*outRayDir = normalize(emissiveRay);
	*pdf = dot(*outRayDir, emissiveNormal) * emissive->area / squaredDistToLight;
	*distToEmissive = native_sqrt(squaredDistToLight);

	MaterialNode matNode = materialNodes[emissive->matNodeIndex];
	return matGetSample3f(emissiveUV, matNode.kval, matNode.kvalTex, texMeta, texData) * dot(surface->normal, *outRayDir);
}

// Given a pre-calculated bounce ray, calculate a PDF for hitting this 
// emissive primitive.
float areaLightGetPdf(
		Surface *surface,
		__global Emissive *emissive,
		__global float4 *vertices, 
		__global float4 *normals,
		float3 outRayDir
		){

	// Transform vertices to world space and check for ray/tri intersection
	// using Moller-Trumbore algorithm
	int offset = emissive->triIndex * 3;
	float3 v0 = vertices[offset].xyz;
	float3 edge01 = vertices[offset+1].xyz - v0;
	float3 edge02 = vertices[offset+2].xyz - v0;

	v0 = mul4x1(v0, emissive->transformMat0, emissive->transformMat1, emissive->transformMat2, emissive->transformMat3);
	edge01 = mul4x1(edge01, emissive->transformMat0, emissive->transformMat1, emissive->transformMat2, emissive->transformMat3);
	edge02 = mul4x1(edge02, emissive->transformMat0, emissive->transformMat1, emissive->transformMat2, emissive->transformMat3);

	float3 pVec = cross(outRayDir, edge02);
	float det = dot(edge01, pVec);

	if (fabs(det) < INTERSECTION_EPSILON){
		return 0.0f;
	}
		
	float invDet = native_recip(det);

	// Calculate barycentric coords
	float3 tVec = surface->point - v0;
	float u = dot(tVec, pVec) * invDet;
	if( u < 0.0f || u > 1.0f ){
		return 0.0f;
	}

	float3 qVec = cross(tVec, edge01);
	float v = dot(outRayDir, qVec) * invDet;
	if( v < 0.0f || u+v > 1.0f ){
		return 0.0f;
	}

	float t = dot(edge02, qVec) * invDet;
	if (t < INTERSECTION_EPSILON){
		return 0.0f;
	}

	return dot(outRayDir, cross(edge01, edge02)) * emissive->area / t * t;
}


// Generate a out ray direction towards a random point on the emissive primitive
// and return a emission material sample from that point.
float3 emissiveGetSample(
		Surface *surface,
		__global Emissive *emissive,
		__global float4 *vertices, 
		__global float4 *normals,
		__global float2 *uv,
		__global MaterialNode *materialNodes,
		__global TextureMetadata *texMeta,
		__global uchar *texData,
		float2 randSample,
		float3 *outRayDir,
		float *pdf,
		float *distToEmissive
		){

	switch( emissive->type ){
		case EMISSIVE_TYPE_AREA_LIGHT:
			return areaLightGetSample(surface, emissive, vertices, normals, uv, materialNodes, texMeta, texData, randSample, outRayDir, pdf, distToEmissive);
		case EMISSIVE_TYPE_ENVIRONMENT_LIGHT:
			return environmentLightGetSample(surface, emissive, materialNodes, texMeta, texData, randSample, outRayDir, pdf, distToEmissive);
	}
	return (float3)(0.0f, 0.0f, 0.0f);
}

// Given a pre-calculated bounce ray, calculate a PDF for hitting this 
// emissive primitive.
float emissiveGetPdf(
		Surface *surface,
		__global Emissive *emissive,
		__global float4 *vertices, 
		__global float4 *normals,
		float3 outRayDir
		){

	switch( emissive->type ){
		case EMISSIVE_TYPE_AREA_LIGHT:
			return areaLightGetPdf(surface, emissive, vertices, normals, outRayDir);
		case EMISSIVE_TYPE_ENVIRONMENT_LIGHT:
			return environmentLightGetPdf(surface, emissive, outRayDir);
	}

	return 0.0f;
}

// Select a random emissive surface from the set of emissive primitives
uint emissiveSelect(
		const int numLights,
		float randSample,
		float *pdf
		){

	// Till I implement RIS, selection probability is as simple as 1/numLights
	*pdf = native_recip((float)numLights);

	return clamp(int(randSample * numLights), 0, numLights - 1);
}

#endif
