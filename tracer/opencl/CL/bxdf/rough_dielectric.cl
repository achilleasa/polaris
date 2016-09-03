#ifndef BXDF_ROUGH_DIELECTRIC_CL
#define BXDF_ROUGH_DIELECTRIC_CL

float3 roughDielectricSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float roughDielectricPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);
float3 roughDielectricEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Sample microfacet surface
float3 roughDielectricSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	float iDotN = dot(inRayDir, surface->normal);
	
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	// If hitting from the inside we need to swap the eta 
	float etaI = matNode->extIOR;
	float etaT = matNode->intIOR;
	if( iDotN < 0.0f ){
		float tmp = etaI;
		etaI = etaT;
		etaT = tmp;
	}

	float eta = etaI / etaT;
	
	// Sample GGX distribution to get halfway vector
	float3 h = ggxGetSample(roughness, inRayDir, surface->normal, randSample);

	// Calculate fresnel 
	float f = fresnelForDielectric(etaI, etaT, iDotN);
	
	float cosTSq = 1.0f + eta * (iDotN * iDotN - 1.0f);

	// Based on the fresnel value randomly sample the reflection ray.
	// In the case where the ray undergoes total internal reflection we 
	// always pick the reflection ray
	if( cosTSq <= 0.0f|| randSample.x <= f ){
		// Reflect I over h to get O
		*outRayDir = 2.0f * dot(inRayDir, h) * h - inRayDir;
		
		float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);
	
		// Recalculate halfway vector (equation 13)
		float iDotN = dot(inRayDir, surface->normal);
		float oDotN = dot(*outRayDir, surface->normal);
		h = normalize(inRayDir + *outRayDir);
		*pdf = cosTSq <= 0.0f ? 1.0f : ggxGetReflectionPdf(roughness, inRayDir, *outRayDir, surface->normal, h);
		
		// Calculate d and g for GGX
		float d = ggxGetD(roughness, surface->normal, h);
		float g = ggxGetG(roughness, inRayDir, *outRayDir, surface->normal, h);

		float denom = 4.0f * iDotN * oDotN;
		return denom > 0.0f ?  ks * f * d * g / denom : 0.0f;
	}

	// Refract I over h to get O
	*outRayDir = (eta * iDotN - sign(iDotN)*sqrt(cosTSq))*h - eta * inRayDir;

	// Recalculate halfway transmission vector (equation 16)
	h = normalize(-(etaI * inRayDir + etaT * *outRayDir));
	*pdf = ggxGetRefractionPdf(roughness, etaI, etaT, inRayDir, *outRayDir, surface->normal, h);

	float iDotH = fabs(dot(inRayDir, h));
	float oDotH = fabs(dot(*outRayDir, h));
	float oDotN = dot(*outRayDir, surface->normal);

	// Calc focus term (see equation 21)
	float focusTermDenom = iDotN * oDotN * (etaI * iDotH + etaT * oDotH) * (etaI * iDotH + etaT * oDotH);
	if( focusTermDenom == 0.0f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}
	float focusTerm = fabs(etaT * etaT * iDotH * oDotH / focusTermDenom);
	
	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, *outRayDir, surface->normal, h);

	// Eval sample (equation 21)
	float3 tf = matGetSample3f(surface->uv, matNode->transmittance, matNode->transmittanceTex, texMeta, texData);
	return tf * (1.0f - f) * d * g * focusTerm;
}

// Get PDF given an outbound ray
float roughDielectricPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	// This is a reflected ray
	if( iDotN > 0.0f ){
		float3 h = normalize(inRayDir + outRayDir);
		return ggxGetReflectionPdf(roughness, inRayDir, outRayDir, surface->normal, h);	
	}
	
	// If hitting from the inside we need to swap the eta 
	float etaI = matNode->extIOR;
	float etaT = matNode->intIOR;
	if( iDotN < 0.0f ){
		float tmp = etaI;
		etaI = etaT;
		etaT = tmp;
	}

	float3 h = normalize(-(etaI * inRayDir + etaT * outRayDir));
	return ggxGetRefractionPdf(roughness, etaI, etaT, inRayDir, outRayDir, surface->normal, h);
}

// Evaluate microfacet BXDF for the selected outgoing ray.
float3 roughDielectricEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(outRayDir, surface->normal);
	
	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->roughness, matNode->roughnessTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	// If hitting from the inside we need to swap the eta 
	float etaI = matNode->extIOR;
	float etaT = matNode->intIOR;
	if( iDotN < 0.0f ){
		float tmp = etaI;
		etaI = etaT;
		etaT = tmp;
	}

	// Calculate fresnel 
	float f = fresnelForDielectric(etaI, etaT, iDotN);

	// This is a reflected ray
	if(iDotN > 0.0f) {
		float3 ks = matGetSample3f(surface->uv, matNode->specularity, matNode->specularityTex, texMeta, texData);
		float3 h = normalize(inRayDir + outRayDir);

		// Calculate d and g for GGX
		float d = ggxGetD(roughness, surface->normal, h);
		float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

		// Eval sample (equation 20)
		float denom = 4.0f * iDotN * oDotN;
		return denom > 0.0f ?  ks * f * d * g / denom : 0.0f;
	}

	float3 h = normalize(-(etaI * inRayDir + etaT * outRayDir));

	float iDotH = fabs(dot(inRayDir, h));
	float oDotH = fabs(dot(outRayDir, h));

	// Calc focus term (see equation 21)
	float focusTermDenom = iDotN * oDotN * (etaI * iDotH + etaT * oDotH) * (etaI * iDotH + etaT * oDotH);
	if( focusTermDenom == 0.0f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}
	float focusTerm = fabs(etaT * etaT * iDotH * oDotH / focusTermDenom);
	
	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

	// Eval sample (equation 21)
	float3 tf = matGetSample3f(surface->uv, matNode->transmittance, matNode->transmittanceTex, texMeta, texData);
	return tf * (1.0f - f) * d * g * focusTerm;
}

#endif
