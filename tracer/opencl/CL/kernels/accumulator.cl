#ifndef ACCUMULATOR_KERNEL_CL
#define ACCUMULATOR_KERNEL_CL

// Clear accumulation buffer
__kernel void clearAccumulator(
		__global float3 *accumulator,
		const uint frameW
		){
	accumulator[(get_global_id(1) * frameW) + get_global_id(0)] = (float3)(0.0f, 0.0f, 0.0f);
}

#endif
