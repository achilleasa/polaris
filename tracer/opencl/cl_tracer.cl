// Global parameters that will be set via a template during compilation time.
__constant unsigned int frameW = 512;
__constant unsigned int frameH = 512;
__constant float texelStepX = 1.0 / 512.0;
__constant float texelStepY = 1.0 / 512.0;

// Emit the color of a pixel by tracing a ray through the scene.
__kernel void tracePixel(
		__global float4 *frameBuffer,
		__global float4 *frustrumCorners,
		const unsigned int blockY,
		const unsigned int samplesPerPixel,
		const float exposure
){
	// Get pixel coordinates
	unsigned int x = get_global_id(0);
	unsigned int y = blockY + get_global_id(1);
	if ( x > frameW || y > frameH ) {
		return;
	}

	// Just set target to white
	frameBuffer[y * frameW + x] = 1.0;
}
