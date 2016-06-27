#ifndef MATRIX_CL
#define MATRIX_CL 

float3 mul4x1(float3 vec, float4 mat[4]);
float3 mul3x1(float3 vec, float4 mat[4]);

// Transform vector with a 4x4 matrix.
float3 mul4x1(float3 vec, float4 mat[4]){
    float3 out;
	// Assume vec.w = 1 to save a multiplication
    out.x = mat[0].x * vec.x + mat[1].x * vec.y + mat[2].x * vec.z + mat[3].x;
    out.y = mat[0].y * vec.x + mat[1].y * vec.y + mat[2].y * vec.z + mat[3].y;
    out.z = mat[0].z * vec.x + mat[1].z * vec.y + mat[2].z * vec.z + mat[3].z;
    return out;
}

// Transform vector with a 3x3 rotation matrix. 
// This function ignores the 4 row/col of the matrix.
float3 mul3x1(float3 vec, float4 mat[4]){
    float3 out;
    out.x = mat[0].x * vec.x + mat[1].x * vec.y + mat[2].x * vec.z;
    out.y = mat[0].y * vec.x + mat[1].y * vec.y + mat[2].y * vec.z;
    out.z = mat[0].z * vec.x + mat[1].z * vec.y + mat[2].z * vec.z;
    return out;
}

#endif
