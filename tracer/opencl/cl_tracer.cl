// Global parameters that will be set via a template during compilation time.
__constant unsigned int frameW = 512;
__constant unsigned int frameH = 512;
__constant float texelStepX = 1.0 / 512.0;
__constant float texelStepY = 1.0 / 512.0;


// Trace a ray and return the gathered color.
float4 traceRay(float4 rayOrigin, float4 rayDir){
	// For now, just return something so we can test the output
	return fabs(rayDir);
}

// Emit the color of a pixel by tracing a ray through the scene.
__kernel void tracePixel(
		__global float4 *frameBuffer,
		__global float4 *frustrumCorners,
		const float4 eyePos,
		const unsigned int blockY,
		const unsigned int samplesPerPixel,
		const float exposure
){
	// Get pixel coordinates
	unsigned int x = get_global_id(0);
	unsigned int y = get_global_id(1);
	if ( x > frameW || y > frameH ) {
		return;
	}

	// Calculate texel coordinates [0,1] range
	float tx = (float)(x) * texelStepX;
	float ty = (float)(y) * texelStepY;
	float accumScaler = 1.0f / (float)samplesPerPixel;
	float4 accum = (float4)(0,0,0,0);
	for( uint sample=0;sample<samplesPerPixel;sample++){
		// Get ray direction using trilinear interpolation; trace ray and
		// add result to accumulation buffer
		float4 lVec = frustrumCorners[0] * (1 - ty) + frustrumCorners[2] * ty;
		float4 rVec = frustrumCorners[1] * (1 - ty) + frustrumCorners[3] * ty;
		accum += traceRay(eyePos, normalize(lVec * (1 - tx) + rVec * tx));
	}

	//
	// Average samples
	accum *= accumScaler;

	// Apply tone-mapping and gamma correction using:
	// 1 - exp(-hdrColor * exposure)) [tone mapping HDR -> LDR]
	// pow(ldr, 0.45)) [gamma correction]
	frameBuffer[y * frameW + x] = pow(-expm1(-accum*exposure), 0.45f);
}
