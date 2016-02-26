// Global parameters that will be set via a template during compilation time.
#ifndef M_PI
#define M_PI 3.14159265358979323846f
#endif
#define FRAME_W 512
#define FRAME_H 512
#define TEXEL_STEP_X 1.0f / 512.0f
#define TEXEL_STEP_Y 1.0f / 512.0f
#define NO_MATERIAL_HIT -1

// Limits
#define MAX_TRACE_STEPS 32
#define MAX_BOUNCES 4 // iris pro compilation runs out of memory if set to > 5
#define MIN_BOUNCES_TO_USE_RR 2

// Epsilon values
#define NUDGE_EPSILON 1.0e-2f
#define DIST_SNAP_EPSILON 1.0e-3f

// Vector helpers
#define VEC3_MAX_COMPONENT(v) max(v.x, max(v.y, v.z))
#define VEC1(x) (float4)(x, 0.0f, 0.0f, 0.0f)
#define VEC2(x,y) (float4)(x, y, 0.0f, 0.0f)
#define VEC3(x,y,z) (float4)(x,y,z,0.0f)
#define VEC4(x,y,z,w) (float4)(x,y,z,w)
#define ORIGIN(v) VEC3(v,v,v)

// Normal estimator generators
#define OBJECT_NORMAL_AT(estimatorFn, objIndex, point) \
	normalize( \
		(float4)( \
			estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point + VEC3(NUDGE_EPSILON, 0,0)) - estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point - VEC3(NUDGE_EPSILON, 0,0)), \
			estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point + VEC3(0, NUDGE_EPSILON,0)) - estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point - VEC3(0, NUDGE_EPSILON,0)), \
			estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point + VEC3(0, 0, NUDGE_EPSILON)) - estimatorFn(SceneObject[objIndex].origin,SceneObject[objIndex].params, point - VEC3(0, 0, NUDGE_EPSILON)), \
			0.0f \
		) \
	)

// Tracer structs
// ---------------------

typedef struct {
	float4 position;
	float4 normal;
	float distFromOrigin;
	int objIndex;
	int matIndex;
} Hit;


// Supported material types
#define MATERIAL_DIFFUSE 0
#define MATERIAL_SPECULAR 1
#define MATERIAL_REFRACTIVE 2
#define MATERIAL_LIGHT 65535

typedef struct {
	unsigned int type;
	float4 diffuse;
	float4 emissive;
} Material;

// Supported object types
#define OBJECT_PLANE 0
#define OBJECT_SPHERE 1
#define OBJECT_BOX 2
#define OBJECT_TORUS 3

typedef struct {
	unsigned int type;
	unsigned int material;
	float4 origin;
	float4 params;
} Object;

// -----------------------
// Object and material definitions
// The following code will be autogenerated by the scene processor
// ------------------
__constant float4 sceneBgColor = (float4)(0.0f);

__constant Material SceneMaterial[] = {
	{MATERIAL_LIGHT, VEC3(0.0f,0.0f,0.0f), VEC3(60.0f,60.0f,60.0f)},
	{MATERIAL_DIFFUSE, VEC3(0.75f, 0.25f, 0.25f)},
	{MATERIAL_DIFFUSE, VEC3(0.25f, 0.25f, 0.75f)},
	{MATERIAL_DIFFUSE, VEC3(0.75f, 0.75f, 0.75f)},
	{MATERIAL_SPECULAR, VEC3(0.999f, 0.999f, 0.999f)},
	{MATERIAL_REFRACTIVE, VEC3(0.999f, 0.999f, 0.999f)}
};

__constant Object SceneObject[] = {
	// Light
	{OBJECT_SPHERE, 0, VEC3(0.0f, 1.5f, 0.0f), VEC1(0.1f)},
	// Left wall
	{OBJECT_BOX, 1, VEC3(-2.0f,0.0f,0.0f), VEC3(0.1f, 2.0f, 2.0f)},
	// Right wall
	{OBJECT_BOX, 2, VEC3(2.0f,0.0f,0.0f), VEC3(0.1f, 2.0f, 2.0f)},
	// Top wall
	{OBJECT_BOX, 3, VEC3(0.0f,2.0f,0.0f), VEC3(2.0f, 0.1f, 2.0f)},
	// Bottom wall
	{OBJECT_BOX, 3, VEC3(0.0f,-2.0f,0.0f), VEC3(2.0f, 0.1f, 2.0f)},
	// Back wall
	{OBJECT_BOX, 3, VEC3(0,0,-2), VEC3(2.0f, 2.0f, 0.1f)},
	// Mirror ball
	{OBJECT_SPHERE, 4, VEC3(-1.0f, -1.4f, -1.0f), VEC1(0.5f)},
	// Glass ball
	{OBJECT_SPHERE, 5, VEC3(1.4f, -1.4f, -0.5f), VEC1(0.5f)}
};

// -----------------------
// Random number and direction generators
// -----------------------
float2 clRand(uint2 *state)
{
    const float2 invMaxInt = (float2) (1.0f/4294967296.0f, 1.0f/4294967296.0f);
    uint x = (*state).x * 17 + (*state).y * 13123;
    (*state).x = (x<<13) ^ x;
    (*state).y ^= (x<<7);

    uint2 tmp = (uint2)
    (
		(x * (x * x * 15731 + 74323) + 871483),
		(x * (x * x * 13734 + 37828) + 234234)
	);

    return convert_float2(tmp) * invMaxInt;
}

// Return random direction on hemiphere around normal.
// This function has: PDF = cos(theta) / pi
float4 rndCosWeightedHemisphereDir(float4 normal, uint2 *seed) {
	float2 rnd  = clRand(seed);

	// Generate point on disk
	float rd = sqrt(rnd.x);
	float phi = 2.0f*M_PI*rnd.y;

	// Generate tangent, bi-tangent vectors
	float4 u = normalize(cross((fabs(normal.x) > .1f ? (float4)(0.0f, 1.0f, 0.0f, 0.0f) : (float4)(1.0f, 0.0f, 0.0f, 0.0f)), normal));
	float4 v = cross(normal,u);

	// Project disk point to unit hemisphere and rotate so that the normal points up
	return normalize(u * rd * cos(phi) + v * rd * sin(phi) + normal * sqrt(1 - rnd.x));
}

// -----------------------
// Distance estimators for supported primitives
// -----------------------
float planeSD(float4 origin, float4 dims, float4 point){
	dims = normalize(dims);
	return dot((point - origin).xyz, dims.xyz) + dims.w;
}
float sphereSD(float4 origin, float4 radius, float4 point){
	return length(point - origin) - radius.x;
}
float boxSD(float4 origin, float4 dims, float4 point){
	float3 d = fabs(point.xyz - origin.xyz) - dims.xyz;
	return fmin(VEC3_MAX_COMPONENT(d), 0.0f) + length(fmax(d, 0.0f));
}
float torusSD(float4 origin, float4 dims, float4 point){
	point -= origin;
	return length((float2)(length(point.xz) - dims.x, point.y)) - dims.y;
}

// -----------------------
// Tracer implementation
// -----------------------

// Estimate normal at intersection point
void getNormal(Hit *hit){
	switch(SceneObject[hit->objIndex].type) {
		case OBJECT_PLANE:
			hit->normal = OBJECT_NORMAL_AT(planeSD, hit->objIndex, hit->position);
		break;
		case OBJECT_SPHERE:
			hit->normal = OBJECT_NORMAL_AT(sphereSD, hit->objIndex, hit->position);
		break;
		case OBJECT_BOX:
			hit->normal = OBJECT_NORMAL_AT(boxSD, hit->objIndex, hit->position);
		break;
		case OBJECT_TORUS:
			hit->normal = OBJECT_NORMAL_AT(torusSD, hit->objIndex, hit->position);
		break;
	}
}

// Estimate distance to nearest object
void intersectWorld(const float4 rayOrigin, const float4 rayDir, const float minDist, const float maxDist, Hit *hit){
	hit->objIndex = NO_MATERIAL_HIT;
	hit->matIndex = NO_MATERIAL_HIT;

	float4 point;
	float curDist = minDist;
	float objDist, nearestDist;
	int nearestObj = -1;
	int nearestMat = -1;
	unsigned int objIndex;
	unsigned int numObjects = (uint)(sizeof(SceneObject) / sizeof(Object));
	for(unsigned int step=0;step<MAX_TRACE_STEPS;step++){
		point = rayOrigin + rayDir * curDist;

		// Find nearest object/dist to point
		nearestDist = FLT_MAX;
		for(objIndex=0;objIndex<numObjects; objIndex++){
			switch(SceneObject[objIndex].type) {
				case OBJECT_PLANE:
					objDist = planeSD(SceneObject[objIndex].origin, SceneObject[objIndex].params, point);
				break;
				case OBJECT_SPHERE:
					objDist = sphereSD(SceneObject[objIndex].origin, SceneObject[objIndex].params, point);
				break;
				case OBJECT_BOX:
					objDist = boxSD(SceneObject[objIndex].origin, SceneObject[objIndex].params, point);
				break;
				case OBJECT_TORUS:
					objDist = torusSD(SceneObject[objIndex].origin, SceneObject[objIndex].params, point);
				break;
			}

			if(objDist < nearestDist){
				nearestDist = objDist;
				nearestObj = objIndex;
				nearestMat = SceneObject[objIndex].material;
			}
		}

		// If we are inside an object nearestDist will be < 0
		nearestDist = fabs(nearestDist);

		curDist += nearestDist;
		if(curDist > maxDist){
			return;
		}

		// We are close enough to the object to register a hit
		if(nearestDist < DIST_SNAP_EPSILON){
			hit->position = point;
			hit->distFromOrigin = curDist;
			hit->objIndex = nearestObj;
			hit->matIndex = nearestMat;
			return;
		}
	}
}

// Trace a ray and return the gathered color.
float4 traceRay(float4 rayOrigin, float4 rayDir, uint2 *rndSeed){
	Hit hit;
	__constant Material *material;
	float4 rCol = (float4)(0.0f);
	float4 mask = (float4)(1.0f);
	unsigned int bounce = 0;
	for(bounce=0;bounce<MAX_BOUNCES;bounce++){
		intersectWorld(rayOrigin, rayDir, NUDGE_EPSILON, FLT_MAX, &hit);

		// No hit
		if(hit.objIndex == NO_MATERIAL_HIT){
			if(bounce == 0){
				rCol = sceneBgColor;
			}
			break;
		}

		// Get material at hit point
		material = &SceneMaterial[hit.matIndex];

		// If we hit a light just add its emissive property and stop
		if( material->type == MATERIAL_LIGHT ){
			rCol += mask * material->emissive;
			break;
		}

		// Get normal at hit point
		getNormal(&hit);

		// After some bounces apply a russian roulette to terminate long paths
		if( bounce > MIN_BOUNCES_TO_USE_RR ) {
			float maxComponent = VEC3_MAX_COMPONENT(material->diffuse);
			if( maxComponent == 0 || clRand(rndSeed).x > maxComponent ){
				break;
			}

			// Boost surviving paths by 1/rr_prob
			mask /= maxComponent;
		}

		// Mask outgoing reflectance by material diffuse property
		mask *= material->diffuse;

		// Next bounce starts at hit point
		rayOrigin = hit.position;

		if( material->type == MATERIAL_DIFFUSE ) {
			// We do importance sampling for diffuse rays
			rayDir = rndCosWeightedHemisphereDir(hit.normal, rndSeed);
		} else{
			// Reflect ray around normal
			rayDir = normalize(-2.0f * dot(hit.normal, rayDir) * hit.normal + rayDir);
		}

		++bounce;
	}

	return rCol;
}

// Emit the color of a pixel by tracing a ray through the scene.
__kernel void tracePixel(
		__global float4 *frameBuffer,
		__global float4 *frustrumCorners,
		const float4 eyePos,
		const unsigned int blockY,
		const unsigned int samplesPerPixel,
		const float exposure,
		const int seed
		){
	// Get pixel coordinates
	unsigned int x = get_global_id(0);
	unsigned int y = get_global_id(1);
	if ( x > FRAME_W || y > FRAME_H ) {
		return;
	}

	// Setup seed for random numbers
	uint2 rndSeed = (uint2)(x * seed, y * seed);

	// Calculate texel coordinates [0,1] range
	float accumScaler = 1.0f / (float)samplesPerPixel;
	float4 accum = (float4)(0.0f,0.0f,0.0f,0.0f);

	for( uint sample=0;sample<samplesPerPixel;sample++){
		// Apply stratified sampling using a tent filter. This will wrap our
		// random numbers in the [-1, 1] range. X and Y point to the top corner
		// of the current texel so we need to add a bit of offset to get the coords
		// into the [-0.5, 1.5] range.
		float2 random = clRand(&rndSeed);
		float offX = random.x < 0.5f ? sqrt(2.0f * random.x) - 0.5f : 1.5f - sqrt(2.0f - 2.0f * random.x);
		float offY = random.y < 0.5f ? sqrt(2.0f * random.y) - 0.5f : 1.5f - sqrt(2.0f - 2.0f * random.y);
		float tx = ((float)x + offX) * TEXEL_STEP_X;
		float ty = ((float)y + offY) * TEXEL_STEP_Y;

		// Get ray direction using trilinear interpolation; trace ray and
		// add result to accumulation buffer
		float4 lVec = frustrumCorners[0] * (1 - ty) + frustrumCorners[2] * ty;
		float4 rVec = frustrumCorners[1] * (1 - ty) + frustrumCorners[3] * ty;
		accum += traceRay(eyePos, normalize(lVec * (1 - tx) + rVec * tx), &rndSeed);
	}

	// Average samples
	accum *= accumScaler;

	// Apply tone-mapping and gamma correction using:
	// 1 - exp(-hdrColor * exposure)) [tone mapping HDR -> LDR]
	// pow(ldr, 0.45)) [gamma correction]
	frameBuffer[y*FRAME_W + x] = pow(-expm1(-accum*exposure), 0.45f);
}
