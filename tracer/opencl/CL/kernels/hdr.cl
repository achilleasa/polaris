#ifndef HDR_KERNEL_CL
#define HDR_KERNEL_CL

// Simple Reinhard tone-mapping
__kernel void tonemapSimpleReinhard(
	__global float3 *accumulator,
	__global Path *paths,
	__global uchar4 *frameBuffer,
	const float sampleWeight,
	const float exposure
		){

			int globalId = get_global_id(0);

			// Apply tone-mapping
			float3 hdrColor = accumulator[globalId] * sampleWeight * exposure;
			float3 mapped = hdrColor / (hdrColor + 1.0f);

			// Apply gamma correction and scale
			float3 normalizedOutput = clamp(pow(mapped, 1.0f / 2.2f), 0.0f, 1.0f) * 255.0f;

			frameBuffer[paths[globalId].pixelIndex] = (uchar4)(
					(uchar)normalizedOutput.r,
					(uchar)normalizedOutput.g,
					(uchar)normalizedOutput.b,
					255 // alpha
					);
		}

#endif
