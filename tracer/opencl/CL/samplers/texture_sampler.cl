#ifndef TEXTURE_SAMPLER_CL
#define TEXTURE_SAMPLER_CL

#define TEX_FMT_LUMINANCE8 0
#define TEX_FMT_LUMINANCE32F 1
#define TEX_FMT_RGBA8 2
#define TEX_FMT_RGBA32F 3

float3 texGetSample3f(float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* data);
float texGetSample1f(float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* data);

// Sample texture at given uv coordinates returning back a float3 vector
float3 texGetSample3f(float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* data) {
	uint2 texDims = (uint2)(
			metadata[texIndex].width,
			metadata[texIndex].height
	);

	// Handle repeating textures by keeping the fractional part of uv and
	// scale to [0, texDims) range
	float2 scaledUV = uv - floor(uv);
	scaledUV.x *= (float)texDims.x;
	scaledUV.y *= (float)texDims.y;

	// Calculate top-left and bottom-right corners for applying bilinear filtering
	uint tx = clamp((uint)scaledUV.x, uint(0), texDims.x - 1);
	uint ty = clamp((uint)scaledUV.y, uint(0), texDims.y - 1);
	uint bx = clamp(tx+1, uint(0), texDims.x - 1);
	uint by = clamp(ty+1, uint(0), texDims.y - 1);

	// Calculate coefficients
	float coeffX = scaledUV.x - (float)tx;
	float coeffY = scaledUV.y - (float)ty;

	__global uchar* basePtr = data + metadata[texIndex].dataOffset;

	switch(metadata[texIndex].format){
		case TEX_FMT_RGBA8:
		{
			const __global uchar4* vecPtr = (__global const uchar4*)basePtr;

			float4 rgbTL = convert_float4(vecPtr[(ty * texDims.x) + tx]);
			float4 rgbTR = convert_float4(vecPtr[(ty * texDims.x) + bx]);
			float4 rgbBL = convert_float4(vecPtr[(by * texDims.x) + tx]);
			float4 rgbBR = convert_float4(vecPtr[(by * texDims.x) + bx]);

			return mix(
					mix(rgbTL, rgbBL, coeffY),
					mix(rgbTR, rgbBR, coeffY),
					coeffX
			).xyz / 255.0f;
		}
		case TEX_FMT_RGBA32F:
		{
			const __global float4* vecPtr = (__global const float4*)basePtr;

			float4 rgbTL = vecPtr[(ty * texDims.x) + tx];
			float4 rgbTR = vecPtr[(ty * texDims.x) + bx];
			float4 rgbBL = vecPtr[(by * texDims.x) + tx];
			float4 rgbBR = vecPtr[(by * texDims.x) + bx];

			return mix(
					mix(rgbTL, rgbBL, coeffY),
					mix(rgbTR, rgbBR, coeffY),
					coeffX
			).xyz;
		}
		case TEX_FMT_LUMINANCE8:
		{
			float rTL = (float)basePtr[(ty * texDims.x) + tx];
			float rTR = (float)basePtr[(ty * texDims.x) + bx];
			float rBL = (float)basePtr[(by * texDims.x) + tx];
			float rBR = (float)basePtr[(by * texDims.x) + bx];
			float r = mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			) / 255.0f;
			
			return (float3)(r,r,r);
		}
		case TEX_FMT_LUMINANCE32F:
		{
			const __global float* floatPtr = (__global const float*)basePtr;
			float rTL = floatPtr[(ty * texDims.x) + tx];
			float rTR = floatPtr[(ty * texDims.x) + bx];
			float rBL = floatPtr[(by * texDims.x) + tx];
			float rBR = floatPtr[(by * texDims.x) + bx];
			float r = mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			);
			
			return (float3)(r,r,r);
		}
	}

	return (float3)(0.0f, 0.0f, 0.0f);
}

// Sample texture at given uv coordinates returning back a float. For multi-channel
// textures we only read from the red channel.
float texGetSample1f(float2 uv, int texIndex, __global TextureMetadata *metadata, __global uchar* data) {
	uint2 texDims = (uint2)(
			metadata[texIndex].width,
			metadata[texIndex].height
	);

	// Handle repeating textures by keeping the fractional part of uv and
	// scale to [0, texDims) range
	float2 scaledUV = uv - floor(uv);
	scaledUV.x *= (float)texDims.x;
	scaledUV.y *= (float)texDims.y;

	// Calculate top-left and bottom-right corners for applying bilinear filtering
	uint tx = clamp((uint)scaledUV.x, uint(0), texDims.x - 1);
	uint ty = clamp((uint)scaledUV.y, uint(0), texDims.y - 1);
	uint bx = clamp(tx+1, uint(0), texDims.x - 1);
	uint by = clamp(ty+1, uint(0), texDims.y - 1);

	// Calculate coefficients
	float coeffX = scaledUV.x - (float)tx;
	float coeffY = scaledUV.y - (float)ty;

	__global uchar* basePtr = data + metadata[texIndex].dataOffset;

	switch(metadata[texIndex].format){
		case TEX_FMT_RGBA8:
		{
			float rTL = (float)basePtr[(ty * texDims.x << 2) + (tx << 2)];
			float rTR = (float)basePtr[(ty * texDims.x << 2) + (bx << 2)];
			float rBL = (float)basePtr[(by * texDims.x << 2) + (tx << 2)];
			float rBR = (float)basePtr[(by * texDims.x << 2) + (bx << 2)];
			return mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			) / 255.0f;
		}
		case TEX_FMT_RGBA32F:
		{
			const __global float* floatPtr = (__global const float*)basePtr;

			float rTL = floatPtr[(ty * texDims.x << 2) + (tx << 2)];
			float rTR = floatPtr[(ty * texDims.x << 2) + (bx << 2)];
			float rBL = floatPtr[(by * texDims.x << 2) + (tx << 2)];
			float rBR = floatPtr[(by * texDims.x << 2) + (bx << 2)];
			return mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			);
		}
		case TEX_FMT_LUMINANCE8:
		{
			float rTL = (float)basePtr[(ty * texDims.x) + tx];
			float rTR = (float)basePtr[(ty * texDims.x) + bx];
			float rBL = (float)basePtr[(by * texDims.x) + tx];
			float rBR = (float)basePtr[(by * texDims.x) + bx];
			return mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			) / 255.0f;
		}
		case TEX_FMT_LUMINANCE32F:
		{
			const __global float* floatPtr = (__global const float*)basePtr;
			float rTL = floatPtr[(ty * texDims.x) + tx];
			float rTR = floatPtr[(ty * texDims.x) + bx];
			float rBL = floatPtr[(by * texDims.x) + tx];
			float rBR = floatPtr[(by * texDims.x) + bx];
			return mix(
					mix(rTL, rBL, coeffY),
					mix(rTR, rBR, coeffY),
					coeffX
			);
		}
	}

	return 0.0f;
}
#endif
