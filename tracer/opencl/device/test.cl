__kernel void square(__global int *in, __global int *out, const int count){
	int tid = get_global_id(0);
	if(tid < count){
		int val = in[tid];
		out[tid] = val * val;
	}
}

__kernel void mapBlock(__global int *in, __global int *out, const int count){
	int x = get_global_id(0);
	int y = get_global_id(1);
	int width = get_global_size(0);

	int tid = (y*width) + x;
	if(tid < count){
		out[tid] = in[tid];
	}
}
