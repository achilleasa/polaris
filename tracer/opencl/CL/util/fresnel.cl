#ifndef FRESNEL_CL
#define FRESNEL_CL

float3 fresnelForDielectricFromSpecular(float3 ks, float iDotN);
float fresnelForDielectric(float eta, float iDotN);
float fresnelForConductor(float eta, float etaK, float iDotN);

// Calculate fresnel given a specular color and cosTheta using Schlick's approximation.
inline float3 fresnelForDielectricFromSpecular(float3 ks, float iDotN){
	float c = 1.0f - fabs(iDotN);
	float c1 = c * c;
	return ks + (1.0f - ks) * c1 * c1 *c;
}

// Calculate fresnel given the eta and cosTheta using Schlick's approximation.
inline float fresnelForDielectric(float eta, float iDotN){
	// Calculate r0 from eta
	float r0 = ((1.0f - eta) * (1.0f - eta)) / ((1.0f + eta) * (1.0f + eta));
	float c = 1.0f - fabs(iDotN);
	float c1 = c * c;
	return r0 + (1.0f - r0) * c1 * c1 *c;
}

// Calculate fresnel for a conductor using etaK as the imaginary part of the eta
inline float fresnelForConductor(float eta, float etaK, float iDotN){
    float  iDotNSq = iDotN * iDotN;
    float twoEtaIDotN = 2.0f * eta * iDotN;

    float t0 = eta * eta + etaK * etaK;
    float t1 = t0 * iDotNSq;
    float Rs = (t0 - twoEtaIDotN + iDotNSq) / (t0 + twoEtaIDotN + iDotNSq);
    float Rp = (t1 - twoEtaIDotN + 1) / (t1 + twoEtaIDotN + 1);

    return 0.5f * (Rp + Rs);
}
#endif
