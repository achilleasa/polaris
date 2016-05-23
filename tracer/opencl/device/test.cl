__kernel void square(__global int *in, __global int *out, const int count){
	int tid = get_global_id(0);
	if(tid < count){
		int val = in[tid];
		out[tid] = val * val;
	}
}
