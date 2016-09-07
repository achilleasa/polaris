#ifndef FRESNEL_CL
#define FRESNEL_CL

float fresnelForDielectric(float etaI, float etaT, float iDotN);
float fresnelForConductor(float eta, float etaK, float iDotN);

// Calculate fresnel given the eta and cosTheta using Schlick's approximation.
inline float fresnelForDielectric(float etaI, float etaT, float iDotN){
	float eta = etaI / etaT;

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
