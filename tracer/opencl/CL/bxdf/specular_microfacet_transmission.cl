#ifndef BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION_CL
#define BXDF_TYPE_SPECULAR_MICROFACET_TRANSMISSION_CL

float3 microfacetTransmissionSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float microfacetTransmissionPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);
float3 microfacetTransmissionEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// Sample microfacet surface
float3 microfacetTransmissionSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	float iDotN = dot(inRayDir, surface->normal);

	// Easy case. I perpendicular to N
	if( iDotN == 0.0f ){
		*pdf = 0.0f;
		return (float3)(0.0f, 0.0f, 0.0f);
	}

	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->ior, matNode->iorTex, texMeta, texData);
	if(iDotN < 0.0f){
		// We are exiting the surface so we need to flip the eta bits
		etaI = etaT;
		etaT = 1.0f;
	}
	float eta = etaI / etaT;

	// Sample GGX distribution to get halfway vector
	float3 h = ggxGetSample(roughness, inRayDir, surface->normal, randSample);

	// Refract I over h to get O
	float iDotH = dot(inRayDir, h);
	float cosTSq = 1.0f + eta * eta * (iDotH * iDotH - 1.0f);
	if( cosTSq < 0.0f ){
		*pdf = 0.0f;
		return (float3)(0.0f, 0.0f, 0.0f);
	}
	*outRayDir = (eta * iDotN - sign(iDotN)*sqrt(cosTSq))*h - eta * inRayDir;

	// Recalculate halfway transmission vector (equation 16)
	h = normalize(-(etaI * inRayDir + etaT * *outRayDir));
	*pdf = ggxGetRefractionPdf(roughness, etaI, etaT, inRayDir, *outRayDir, surface->normal, h);

	// Eval sample
	return microfacetTransmissionEval(surface, matNode, texMeta, texData, inRayDir, *outRayDir);
}

// Get PDF given an outbound ray
float microfacetTransmissionPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(outRayDir, surface->normal);
	
	// If they are on the same side of the surface PDF should be 0
	// so bail out without sampling anything.
	if( iDotN * oDotN >= 0.0f ){
		return 0.0f;
	}

	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;

	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->ior, matNode->iorTex, texMeta, texData);
	if(dot(inRayDir, surface->normal) < 0.0f){
		// We are exiting the surface so we need to flip the eta
		etaI = etaT;
		etaT = 1.0f;
	}

	// Calculate halfway transmission vector (equation 16)
	float3 h = -(etaI * inRayDir + etaT * outRayDir);

	return ggxGetRefractionPdf(roughness, etaI, etaT, inRayDir, outRayDir, surface->normal, h);
}

// Evaluate microfacet BXDF for the selected outgoing ray.
float3 microfacetTransmissionEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	float iDotN = dot(inRayDir, surface->normal);
	float oDotN = dot(outRayDir, surface->normal);
	
	// If they are on the same side of the surface PDF should be 0
	// so bail out without sampling anything.
	if( iDotN * oDotN >= 0.0f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}

	// Use Disney's remapping: a = roughness^2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	roughness *= roughness;
	
	float etaI = 1.0f;
	float etaT = matGetSample1f(surface->uv, matNode->ior, matNode->iorTex, texMeta, texData);
	if(iDotN < 0.0f){
		// We are exiting the surface so we need to flip the eta
		etaI = etaT;
		etaT = 1.0f;
	}

	// Eval fresnel term
	float f = fresnelForDielectric(etaI / etaT, iDotN);

	// Get halfway trasmission vector (equation 16)
	float3 h = normalize(-(etaI * inRayDir + etaT * outRayDir));

	// Calc all required dot products
	float iDotH = fabs(dot(inRayDir, h));
	float oDotH = fabs(dot(outRayDir, h));

	// Calc focus term (see equation 21)
	//float focusTermDenom = iDotN * oDotN * (etaI*iDotH + etaT*oDotH)*(etaI*iDotH + etaT*oDotH);
	float focusTermDenom = iDotN * oDotN * (etaI * iDotH + etaT * oDotH) * (etaI * iDotH + etaT * oDotH);
	if( focusTermDenom == 0.0f ){
		return (float3)(0.0f, 0.0f, 0.0f);
	}
	float focusTerm = fabs(etaT * etaT * iDotH * oDotH / focusTermDenom);
	
	// Calculate d and g for GGX
	float d = ggxGetD(roughness, surface->normal, h);
	float g = ggxGetG(roughness, inRayDir, outRayDir, surface->normal, h);

	// Eval sample (equation 21)
	float3 tf = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	return tf * (1.0f - f) * d * g * focusTerm;
}

#endif
