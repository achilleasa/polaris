#ifndef DISTRIBUTION_SAMPLER_CL
#define DISTRIBUTION_SAMPLER_CL

float _ggxGetG1(float roughness, float3 v, float3 n, float3 m);
float ggxGetG(float roughness, float3 inRayDir, float3 outRayDir, float3 n, float3 m);
float ggxGetD(float roughness, float3 n, float3 m);
float3 ggxGetSample(float roughness, float3 inRayDir, float3 n, float2 randSample);
float ggxGetReflectionPdf(float roughness, float3 inRayDir, float3 outRayDir, float3 n, float3 h);
float ggxGetRefractionPdf(float roughness, float etaI, float etaT, float3 inRayDir, float3 outRayDir, float3 n, float3 h);
float3 cosWeightedHemisphereGetSample(float3 normal, float2 randSample);

// See https://www.cs.cornell.edu/~srm/publications/EGSR07-btdf.pdf
// for GGX distribution formulas

// G1(v, m) = 2 / 1 + sqrt( 1 + a^2 * tanv^2 )  (formula 34)
float _ggxGetG1(float roughness, float3 v, float3 n, float3 m){
	float nDotV = dot(n,v);
	float mDotV = dot(m,v);
	if( nDotV * mDotV <= 0.0f ){
		return 0.0f;
	}
	float nDotVSq = nDotV * nDotV;

	// Calc tanV^2
	float tanSq = nDotVSq > 0.0f ? (1.0f - nDotVSq) / nDotVSq : 0.0f;

	float aSq = roughness * roughness;
    return 2.0f / (1.0f + sqrt(1.0f + aSq * tanSq));
}

// Use smith approximation for G:
// G(l, v, h) = G1(l,h) * G1(v,h)
float ggxGetG(float roughness, float3 inRayDir, float3 outRayDir, float3 n, float3 m){
	return _ggxGetG1(roughness, inRayDir, n, m) * _ggxGetG1(roughness, outRayDir, n, m);
}

// D(m) = a^2 / PI * cosT^4 * (a^2 + tanT^2)^2  (formula 33)
float ggxGetD(float roughness, float3 n, float3 m){
	float nDotM = dot(n, m);
	if( nDotM <= 0.0f ){
		return 0.0f;
	}
	float nDotMSq = nDotM * nDotM;

	// Calc tanT^2
	float tanSq = nDotM != 0.0f ? ((1.0f - nDotMSq) / nDotMSq) : 0.0f;

	// Calc denominator
	float aSq = roughness * roughness;
	float denom = C_PI * nDotMSq * nDotMSq * (aSq + tanSq) * (aSq + tanSq);
	return denom > 0.0f ? (aSq / denom) : 0.0f;
}

// Sample GGX distribution to generate a normal that will be used for microfacet calculations.
float3 ggxGetSample(float roughness, float3 inRayDir, float3 n, float2 randSample){
	// Generate tangent, bi-tangent vectors
	float3 u,v;
	TANGENT_VECTORS(n, u, v);

	// According to equations (35, 36) for sampling GGX:
	// theta = atan( a * sqrt(randSample.x / 1 - randSample.x) )
	// phi = 2 * pi * randSample.y
	float theta = atan( roughness * sqrt(randSample.x / (1.0f - randSample.x)));
	theta = theta >= 0.0f ? theta : (theta + C_TWO_TIMES_PI);

	float cosTheta = native_cos(theta);
	float sinTheta = sqrt(1.0f - cosTheta * cosTheta );

	float cosPhi = native_cos(C_TWO_TIMES_PI * randSample.y);
	float sinPhi = sqrt(1.0f - cosPhi * cosPhi);

    // Project and rotate to get the halfway vector
    return normalize(u * sinTheta * cosPhi + v * sinTheta * sinPhi + n * cosTheta);
}

float ggxGetReflectionPdf(float roughness, float3 inRayDir, float3 outRayDir, float3 n, float3 h) {
	float nDotH = fabs(dot(n, h));
	float oDotH = fabs(dot(outRayDir, h));

	// pdf = D * hDotN / 4 * oDotH
	// the nominator comes from equation 24 and
	// the denominator comes form the half-dir Jacobian (equation 14)
	float denom = 4.0f * oDotH;
	return denom == 0.0f ? 0.0f : ggxGetD(roughness, n, h) * nDotH / denom; 
}

float ggxGetRefractionPdf(float roughness, float etaI, float etaT, float3 inRayDir, float3 outRayDir, float3 n, float3 h) {
	float iDotH = fabs(dot(inRayDir, h));
	float oDotH = fabs(dot(outRayDir, h));
	float hDotN = fabs(dot(h, n));

	// pdf = D * hDotN * focusTerm where
	// focusTerm = etaT * etaT * oDotH / (etaI * iDotH + etaT * oDotH)^2
	float denom = (etaI * iDotH + etaT * oDotH) * (etaI * iDotH + etaT * oDotH);
	return denom > 0.0f ? ggxGetD(roughness, n, h) * hDotN * oDotH * etaT * etaT / denom : 0.0f; 
}

// Sample hemisphere direction using a cosine weighted distribution
// 
// PDF = cos(theta) / pi
float3 cosWeightedHemisphereGetSample(float3 normal, float2 randSample) {
	// Generate point on disk
	float rd = sqrt(randSample.x);
	float phi = C_TWO_TIMES_PI*randSample.y;

	// Generate tangent, bi-tangent vectors
	float3 u,v;
	TANGENT_VECTORS(normal, u, v);

	// Project disk point to unit hemisphere and rotate so that the normal points up
	return normalize(u * rd * native_cos(phi) + v * rd * native_sin(phi) + normal * native_sqrt(1 - randSample.x));
}

#endif
