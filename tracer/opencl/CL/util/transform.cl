#ifndef TRANSFORM_CL
#define TRANSFORM_CL 

float3 mul4x1(float3 vec, float4 mat0, float4 mat1, float4 mat2, float4 mat3);
float3 mul3x1(float3 vec, float3 mat0, float3 mat1, float3 mat2);
float2 rayToLatLongUV(float3 vec);

// Transform vector with a 4x4 matrix.
float3 mul4x1(float3 vec, float4 mat0, float4 mat1, float4 mat2, float4 mat3){
    float3 out;
	// Assume vec.w = 1 to save a multiplication
    out.x = mat0.x * vec.x + mat1.x * vec.y + mat2.x * vec.z + mat3.x;
    out.y = mat0.y * vec.x + mat1.y * vec.y + mat2.y * vec.z + mat3.y;
    out.z = mat0.z * vec.x + mat1.z * vec.y + mat2.z * vec.z + mat3.z;
    return out;
}

// Transform vector with a 3x3 rotation matrix. 
// This function ignores the 4 row/col of the matrix.
float3 mul3x1(float3 vec, float3 mat0, float3 mat1, float3 mat2){
    float3 out;
    out.x = mat0.x * vec.x + mat1.x * vec.y + mat2.x * vec.z;
    out.y = mat0.y * vec.x + mat1.y * vec.y + mat2.y * vec.z;
    out.z = mat0.z * vec.x + mat1.z * vec.y + mat2.z * vec.z;
    return out;
}

// Convert ray direction vector to normalized spherical UV coords
float2 rayToLatLongUV(float3 vec){
	float at2 = atan2(vec.x, vec.z);
    float r = length(vec);

	return (float2)(
			(at2 >= 0.0f ? at2 : (at2 + C_TWO_TIMES_PI)) / C_TWO_TIMES_PI,
			acos(vec.y / r) / C_PI
		);
}

#endif
