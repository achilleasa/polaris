#ifndef INTERSECT_CL
#define INTERSECT_CL

#define INTERSECTION_EPSILON 0.001f

#define BVH_MAX_STACK_SIZE 32
#define BVH_IS_LEAF(node) (node->leftChild.w <= 0)
#define BVH_MESH_INSTANCE_ID(node) (-node->meshInstance.w)
#define BVH_MESH_INSTANCE(node) (meshInstances[-node->meshInstance.w])
#define BVH_LEFT_CHILD(node) (bvhNodes + node->leftChild.w)
#define BVH_RIGHT_CHILD(node) (bvhNodes + node->rightChild.w)
#define BVH_TRIANGLE_INDEX(node) (-node->firstTriIndex.w)
#define BVH_TRIANGLE_COUNT(node) (node->numTriangles.w)

float intersectRayNode(Ray* ray, __global BvhNode* node);
bool rayMeshIntersectionTest( Ray ray, MeshInstance meshInstance, __global BvhNode* bvhNodes, __global float4* vertexList);
bool rayMeshIntersectionQuery( Ray ray, MeshInstance meshInstance, __global BvhNode* bvhNodes, __global float4* vertexList, Intersection *intersection);

float intersectRayNode(Ray* ray, __global BvhNode* node){
	float3 invDir = native_recip(ray->dir.xyz);
	float3 tmin = (node->minExtent.xyz - ray->origin.xyz) * invDir;
	float3 tmax = (node->maxExtent.xyz - ray->origin.xyz) * invDir;

	// If tmin > tmax we need to swap them
	float3 rmin = fmin(tmin, tmax);
	float3 rmax = fmax(tmin, tmax);

	float minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
	float maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);

	return minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray->origin.w ? FLT_MAX : maxmin);
}

bool rayMeshIntersectionTest(
		Ray ray,
		MeshInstance meshInstance,
		__global BvhNode* bvhNodes,
		__global float4* vertexList
		){

	// Transform ray by mesh instance matrix. Note we do not apply translation
	// to the ray direction
	ray.origin.xyz = mul4x1(ray.origin.xyz, meshInstance.transformMat);
	ray.dir.xyz = mul3x1(ray.dir.xyz, meshInstance.transformMat);

	uchar stackIndex = 0;
	__global BvhNode *nodeStack[BVH_MAX_STACK_SIZE];
	__global BvhNode *curNode = bvhNodes + meshInstance.bvhRoot;

	for(;;){
		if(BVH_IS_LEAF(curNode)){
			// Intersect with all triangles using the Moller-Trumbore algorithm
			int triStartIndex = BVH_TRIANGLE_INDEX(curNode);
			int numTriangles = BVH_TRIANGLE_COUNT(curNode);
			for(int vIndex = triStartIndex * 3; vIndex < (triStartIndex + numTriangles)*3;vIndex+=3){
				float3 v0 = vertexList[vIndex].xyz;
				float3 edge01 = vertexList[vIndex+1].xyz - v0;
				float3 edge02 = vertexList[vIndex+2].xyz - v0;

				float3 pVec = cross(ray.dir.xyz, edge02);
				float det = dot(edge01, pVec);

				if (fabs(det) < INTERSECTION_EPSILON){
					continue;
				}

				float invDet = native_recip(det);

				// Calculate barycentric coords
				float3 tVec = ray.origin.xyz - v0;
				float u = dot(tVec, pVec) * invDet;
				if (u < 0.0f || u > 1.0f){
					continue;
				}

				float3 qVec = cross(tVec, edge01);
				float v = dot(ray.dir.xyz, qVec) * invDet;
				if (v < 0.0f || u + v > 1.0f){
					continue;
				}

				// Found hit; early exit
				return true;
			}

			// No hit; pop next pointer of the stack
			if( stackIndex == 0){
				return false;
			}
			curNode = nodeStack[--stackIndex];
		}else {
			// Load children
			__global BvhNode *lChild = BVH_LEFT_CHILD(curNode);
			__global BvhNode *rChild = BVH_RIGHT_CHILD(curNode);

			// Check for intersection with both children
			float lHitDist = intersectRayNode(&ray, lChild);
			float rHitDist = intersectRayNode(&ray, rChild);

			if( lHitDist < FLT_MAX && rHitDist == FLT_MAX ){
				curNode = lChild;
			} else if (lHitDist == FLT_MAX && rHitDist < FLT_MAX ){
				curNode = rChild;
			} else if (lHitDist < FLT_MAX && rHitDist < FLT_MAX ){
				// Follow near child and push far child to the stack
				curNode = lHitDist < rHitDist ? lChild : rChild;
				nodeStack[stackIndex++] = lHitDist < rHitDist ? rChild : lChild;
			} else {
				// Just pop the next pointer of the stack
				if(stackIndex == 0){
					return false;
				}
				curNode = nodeStack[--stackIndex];
			}
		}
	}
}

// Test for ray intersections with scene geometry and set an ouput flag to indicate
// intersections. This method does not calculate any intersection details so its
// cheaper to use for general intersection queries (e.g light occlusion)
__kernel void rayIntersectionTest(
		__global Ray* rays,
		__global BvhNode* bvhNodes,
		__global MeshInstance* meshInstances,
		__global float4* vertexList,
		__global int* hitFlag
		){

	// Load ray
	int threadId = get_global_id(0);
	Ray ray = rays[threadId];

	uchar stackIndex = 0;
	__global BvhNode *nodeStack[BVH_MAX_STACK_SIZE];
	__global BvhNode *curNode = bvhNodes;

	for(;;){
		if(BVH_IS_LEAF(curNode)){
			if( rayMeshIntersectionTest(ray, BVH_MESH_INSTANCE(curNode), bvhNodes, vertexList)){
				hitFlag[threadId] = 1;
				return;
			}

			// No hit; pop next pointer of the stack
			if( stackIndex == 0 ){
				return;
			}
			curNode = nodeStack[--stackIndex];
		}else {
			// Load children
			__global BvhNode *lChild = BVH_LEFT_CHILD(curNode);
			__global BvhNode *rChild = BVH_RIGHT_CHILD(curNode);

			// Check for intersection with both children
			float lHitDist = intersectRayNode(&ray, lChild);
			float rHitDist = intersectRayNode(&ray, rChild);

			if( lHitDist < FLT_MAX && rHitDist == FLT_MAX ){
				curNode = lChild;
			} else if (lHitDist == FLT_MAX && rHitDist < FLT_MAX ){
				curNode = rChild;
			} else if (lHitDist < FLT_MAX && rHitDist < FLT_MAX ){
				// Follow near child and push far child to the stack
				curNode = lHitDist < rHitDist ? lChild : rChild;
				nodeStack[stackIndex++] = lHitDist < rHitDist ? rChild : lChild;
			} else {
				// Just pop the next pointer of the stack
				if(stackIndex == 0){
					return;
				}
				curNode = nodeStack[--stackIndex];
			}
		}
	}
}

// Test for intersection with bottom BVH. If a better intersection is found, the
// passed intersection pointer will get updated. Returns true if a better intersection
// wad found
bool rayMeshIntersectionQuery(
		Ray ray,
		MeshInstance meshInstance,
		__global BvhNode* bvhNodes,
		__global float4* vertexList,
		Intersection *intersection
		){

	// Transform ray by mesh instance matrix. Note we do not apply translation
	// to the ray direction
	ray.origin.xyz = mul4x1(ray.origin.xyz, meshInstance.transformMat);
	ray.dir.xyz = mul3x1(ray.dir.xyz, meshInstance.transformMat);

	uchar stackIndex = 0;
	__global BvhNode *nodeStack[BVH_MAX_STACK_SIZE];
	__global BvhNode *curNode = bvhNodes + meshInstance.bvhRoot;

	bool foundIntersection = false;

	for(;;){
		if(BVH_IS_LEAF(curNode)){
			// Intersect with all triangles using the Moller-Trumbore algorithm
			int triStartIndex = BVH_TRIANGLE_INDEX(curNode);
			int numTriangles = BVH_TRIANGLE_COUNT(curNode);
			for(int vIndex = triStartIndex * 3; vIndex < (triStartIndex + numTriangles)*3;vIndex+=3){
				float3 v0 = vertexList[vIndex].xyz;
				float3 edge01 = vertexList[vIndex+1].xyz - v0;
				float3 edge02 = vertexList[vIndex+2].xyz - v0;

				float3 pVec = cross(ray.dir.xyz, edge02);
				float det = dot(edge01, pVec);

				if (fabs(det) < INTERSECTION_EPSILON){
					continue;
				}

				float invDet = native_recip(det);

				// Calculate barycentric coords
				float3 tVec = ray.origin.xyz - v0;
				float u = dot(tVec, pVec) * invDet;
				if (u < 0.0f || u > 1.0f){
					continue;
				}

				float3 qVec = cross(tVec, edge01);
				float v = dot(ray.dir.xyz, qVec) * invDet;
				if (v < 0.0f || u + v > 1.0f){
					continue;
				}

				float t = dot(edge02, qVec) * invDet;
				if( t < intersection->wuvt.w ){
					intersection->wuvt = (float4)(
							1.0f - (u+v),
							u,
							v,
							t
					);
					intersection->triIndex = vIndex / 3;
					foundIntersection = true;
				}
			}

			// pop next pointer of the stack
			if( stackIndex == 0 ){
				return foundIntersection;
			}
			curNode = nodeStack[--stackIndex];
		}else {
			// Load children
			__global BvhNode *lChild = BVH_LEFT_CHILD(curNode);
			__global BvhNode *rChild = BVH_RIGHT_CHILD(curNode);

			// Check for intersection with both children
			float lHitDist = intersectRayNode(&ray, lChild);
			float rHitDist = intersectRayNode(&ray, rChild);

			if( lHitDist < FLT_MAX && rHitDist == FLT_MAX ){
				curNode = lChild;
			} else if (lHitDist == FLT_MAX && rHitDist < FLT_MAX ){
				curNode = rChild;
			} else if (lHitDist < FLT_MAX && rHitDist < FLT_MAX ){
				// Follow near child and push far child to the stack
				curNode = lHitDist < rHitDist ? lChild : rChild;
				nodeStack[stackIndex++] = lHitDist < rHitDist ? rChild : lChild;
			} else {
				// Just pop the next pointer of the stack
				if(stackIndex == 0){
					return foundIntersection;
				}
				curNode = nodeStack[--stackIndex];
			}
		}
	}
}

// Test for ray intersections with scene geometry. Sets an ouput flag to indicate
// intersections and also emits intersection data for any found intersections.
__kernel void rayIntersectionQuery(
		__global Ray* rays,
		__global BvhNode* bvhNodes,
		__global MeshInstance* meshInstances,
		__global float4* vertexList,
		__global int* hitFlag,
		__global Intersection* intersections
		){

	Intersection intersection;
	intersection.wuvt.w = FLT_MAX;

	// Load ray
	int threadId = get_global_id(0);
	Ray ray = rays[threadId];

	uchar stackIndex = 0;
	__global BvhNode *nodeStack[BVH_MAX_STACK_SIZE];
	__global BvhNode *curNode = bvhNodes;

	for(;;){
		if(BVH_IS_LEAF(curNode)){
			if( rayMeshIntersectionQuery(ray, BVH_MESH_INSTANCE(curNode), bvhNodes, vertexList, &intersection)){
				hitFlag[threadId] = 1;
				intersection.meshInstance = BVH_MESH_INSTANCE_ID(curNode);
			}

			// No hit; pop next pointer of the stack
			if( stackIndex == 0 ){
				break;
			}
			curNode = nodeStack[--stackIndex];
		}else {
			// Load children
			__global BvhNode *lChild = BVH_LEFT_CHILD(curNode);
			__global BvhNode *rChild = BVH_RIGHT_CHILD(curNode);

			// Check for intersection with both children
			float lHitDist = intersectRayNode(&ray, lChild);
			float rHitDist = intersectRayNode(&ray, rChild);

			if( lHitDist < FLT_MAX && rHitDist == FLT_MAX ){
				curNode = lChild;
			} else if (lHitDist == FLT_MAX && rHitDist < FLT_MAX ){
				curNode = rChild;
			} else if (lHitDist < FLT_MAX && rHitDist < FLT_MAX ){
				// Follow near child and push far child to the stack
				curNode = lHitDist < rHitDist ? lChild : rChild;
				nodeStack[stackIndex++] = lHitDist < rHitDist ? rChild : lChild;
			} else {
				// Just pop the next pointer of the stack
				if(stackIndex == 0){
					break;
				}
				curNode = nodeStack[--stackIndex];
			}
		}
	}

	if( intersection.wuvt.w < FLT_MAX ){
		intersections[threadId] = intersection;
	}
}
#endif
