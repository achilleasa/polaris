#ifndef RANDOM_SAMPLER_CL
#define RAND_SAMPLER_CL

float2 randomGetSample2f(uint2 *state);

// Generate 2 random numbers in the [0, 1) range and update RNG state
float2 randomGetSample2f(uint2 *state)
{
	const float2 invMaxInt = (float2) (1.0f/4294967296.0f, 1.0f/4294967296.0f);
	uint x = (*state).x * 17 + (*state).y * 13123;
	(*state).x = (x<<13) ^ x;
	(*state).y ^= (x<<7);

	uint2 tmp = (uint2)((x * (x * x * 15731 + 74323) + 871483),(x * (x * x * 13734 + 37828) + 234234));
	return convert_float2(tmp) * invMaxInt;
}

#endif
