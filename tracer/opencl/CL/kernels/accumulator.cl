#ifndef ACCUMULATOR_KERNEL_CL
#define ACCUMULATOR_KERNEL_CL

// Clear accumulation buffer
__kernel void clearAccumulator(
		__global float3 *accumulator
		){
	accumulator[get_global_id(0)] = (float3)(0.0f, 0.0f, 0.0f);
}


// Aggregate trace accumulator to the primary tracer's frame accumulator 
__kernel void aggregateAccumulator(
		__global float3 *srcAccumulator,
		__global float3 *dstAccumulator
		){
	int globalId = get_global_id(0);
	dstAccumulator[globalId] += srcAccumulator[globalId];
}

#endif
