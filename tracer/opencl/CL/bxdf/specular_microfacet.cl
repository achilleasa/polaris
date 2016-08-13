#ifndef BXDF_MICROFACET_GGX_CL
#define BXDF_MICROFACET_GGX_CL

// GGX formulas explode if roughness is 0
#define MIN_ROUGHNESS 0.0001f

float3 microfacetSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf);
float microfacetPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);
float3 microfacetEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir);

// All formulas are taken from:
// https://graphicrants.blogspot.co.uk/2013/08/specular-brdf-reference.html
// https://cdn2.unrealengine.com/Resources/files/2013SiggraphPresentationsNotes-26915738.pd://cdn2.unrealengine.com/Resources/files/2013SiggraphPresentationsNotes-26915738.pdf
//
// The following notation is used:
// n = surface normall
// v = incoming ray (pointing outwards from the surface)
// l = outgoing ray
// h = halfway vector = normalize(v + n)
//
// We start with the generic microfacet equation:
//     D(h) * F(v,h) * G(l,v,h)
// f = ------------------------
//      4 * dot(n,l) * dot(n,v)
//
// And use:
//
// GGX normal distribution:
//                           a^2
// D_GGX(h) = --------------------------------
//            π * (dot(n,h)^2 * (a^2−1) + 1)^2
//
// 
// For the geometry term use the Schlick model with k = 0.5 * a to approximate
// Smith's model
//
// G_Schlick = G1(l) * G1(h) where:
//
//			       dot(n,v)
// G1(v) =  -----------------------
//             dot(n,v)*(1-k) + k
//
//                  dot(n,l) * dot(n, h)
// G_Schlick = --------------------------------------------
//                (dot(n,l)*(1-k) + k) * (dot(n,h)(1-k) + k
//
//  k = 0.5 * a
//
// For the Fresnel term we use Schlick's approximation:
// F = R0 + (1 - R0)(1 - dot(h, l))^5 where R0 is fetched from Kval

// Sample microfacet surface
float3 microfacetSample( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float2 randSample, float3 inRayDir, float3 *outRayDir, float *pdf){
	// Get roughness. We use a = roughness ^ 2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	float a = roughness * roughness;
	float aSquared = a * a;
	
	// Note: inRayDir points away from surface
	float3 h = normalize(inRayDir + surface->normal);

	// Sample ray around halfway vector using a PDF matching D_GGX, 
	// then mirror inRayDir around it to get the outgoing ray
	float3 ggxDir = rayGetGGXSample(h, aSquared, randSample);
	*outRayDir = 2.0f * fabs(dot(ggxDir, inRayDir)) * ggxDir - inRayDir;
	
	// Calc dot products
	float nDotV = clamp( dot( surface->normal, inRayDir ), 0.0f, 1.0f);
	float nDotL = clamp( dot( surface->normal, *outRayDir ), 0.0f, 1.0f);
	float nDotH = clamp( dot( surface->normal, h ), 0.0f, 1.0f);
	float vDotH = clamp( dot( inRayDir, h ), 0.0f, 1.0f );

	// Evaluate D_GGX
	float dDenomSqrt = nDotH * nDotH * (aSquared - 1.0f) + 1.0f;
	float d = aSquared * native_recip(C_PI * dDenomSqrt * dDenomSqrt);

	// Eval fresnel term
	float3 r0 = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	float3 f = r0 + (1.0f - r0)*native_powr(1.0f - vDotH, 5.0f);

	// Eval G
	float k = 0.5f * a;
	float gDenom = (nDotL*(1.0f - k) + k) * (nDotH*(1.0f - k) + k);
	float g = nDotL * nDotH * native_recip(gDenom);

	// Calculate PDF
	// pdf = D * dot(n,h) / (4 * dot(n,l) * dot(n,v))
	*pdf = vDotH > 0.0f ? (d * nDotH) / (4.0f * nDotL * nDotV) : 0.0f;

	// Eval sample
	return d * f * g * native_recip(4.0f * nDotL * nDotV);
}

// Get PDF given an outbound ray
float microfacetPdf( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Get roughness. We use a = roughness ^ 2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	float aSquared = native_powr(roughness, 4.0f);
	
	// Note: inRayDir points away from surface
	float3 h = normalize(inRayDir + surface->normal);

	// Calc dot products
	float nDotV = clamp( dot( surface->normal, inRayDir ), 0.0f, 1.0f);
	float nDotL = clamp( dot( surface->normal, outRayDir ), 0.0f, 1.0f);
	float nDotH = clamp( dot( surface->normal, h ), 0.0f, 1.0f);
	float vDotH = clamp( dot( inRayDir, h ), 0.0f, 1.0f );

	// Evaluate D_GGX
	float dDenomSqrt = nDotH * nDotH * (aSquared - 1.0f) + 1.0f;
	float d = aSquared * native_recip(C_PI * dDenomSqrt * dDenomSqrt);

	// Calculate PDF
	// pdf = D * dot(n,h) / (4 * dot(n,l) * dot(n,v))
	return vDotH > 0.0f ? (d * nDotH) / (4.0f * nDotL * nDotV) : 0.0f;
}

// Evaluate microfacet BXDF for the selected outgoing ray.
float3 microfacetEval( Surface *surface, MaterialNode *matNode, __global TextureMetadata *texMeta, __global uchar *texData, float3 inRayDir, float3 outRayDir){
	// Get roughness. We use a = roughness ^ 2
	float roughness = clamp(matGetSample1f(surface->uv, matNode->nval, matNode->nvalTex, texMeta, texData), MIN_ROUGHNESS, 1.0f);
	float a = roughness * roughness;
	float aSquared = a * a;
	
	// Note: inRayDir points away from surface
	float3 h = normalize(inRayDir + surface->normal);

	// Calc dot products
	float nDotV = clamp( dot( surface->normal, inRayDir ), 0.0f, 1.0f);
	float nDotL = clamp( dot( surface->normal, outRayDir ), 0.0f, 1.0f);
	float nDotH = clamp( dot( surface->normal, h ), 0.0f, 1.0f);
	float vDotH = clamp( dot( inRayDir, h ), 0.0f, 1.0f );

	// Evaluate D_GGX
	float dDenomSqrt = nDotH * nDotH * (aSquared - 1.0f) + 1.0f;
	float d = aSquared * native_recip(C_PI * dDenomSqrt * dDenomSqrt);

	// Eval fresnel term
	float3 r0 = matGetSample3f(surface->uv, matNode->kval, matNode->kvalTex, texMeta, texData);
	float3 f = r0 + (1.0f - r0)*native_powr(1.0f - vDotH, 5.0f);

	// Eval G
	float k = 0.5f * a;
	float gDenom = (nDotL*(1.0f - k) + k) * (nDotH*(1.0f - k) + k);
	float g = nDotL * nDotH * native_recip(gDenom);

	// Eval sample
	return d * f * g * native_recip(4.0f * nDotL * nDotV);
}

#endif
