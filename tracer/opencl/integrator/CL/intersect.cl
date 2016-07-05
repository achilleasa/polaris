#ifndef INTERSECT_CL
#define INTERSECT_CL

#define MAX_ITERATIONS 10000

#define INTERSECTION_EPSILON 0.001f

#define BVH_MAX_STACK_SIZE 32

#define BVH_IS_LEAF(node) (node.leftChild.w <= 0)
#define BVH_LEFT_CHILD(node) (node.leftChild.w)
#define BVH_RIGHT_CHILD(node) (node.rightChild.w)
#define BVH_TRIANGLE_INDEX(node) (-node.firstTriIndex.w)
#define BVH_TRIANGLE_COUNT(node) (node.numTriangles.w)
#define BVH_MESH_INSTANCE_ID(node) (-node.meshInstance.w)

#define RAY_PACKET_SIZE 64
#define HALF_RAY_PACKET_SIZE RAY_PACKET_SIZE / 2

#define RAY_VISIT_NO_NODE 0
#define RAY_VISIT_LEFT_NODE 1
#define RAY_VISIT_RIGHT_NODE 2
#define RAY_VISIT_BOTH_NODES 3

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

	Intersection intersection;
	intersection.wuvt.w = FLT_MAX;

	uint globalId = get_global_id(0);

	int stackIndex;
	int meshBvhStackStartIndex;
	uint nodeStack[BVH_MAX_STACK_SIZE];
	BvhNode curNode;
	BvhNode childNodes[2];
	int meshInstanceId;
	MeshInstance meshInstance;

	// triangle intersection vars
	float3 v0, edge01, edge02;

	// Node bbox and leaf primitive intersection vars
	float3 invDir, tmin, tmax, rmin, rmax;
	float minmax, maxmin;
	int triStartIndex, numTriangles;

	// Fetch ray
	Ray	ray = rays[globalId];
	float3 origRayOrigin = ray.origin.xyz;
	float3 origRayDir = ray.dir.xyz;
	
	// Setup stack
	stackIndex = 0;
	meshBvhStackStartIndex = -1;
	curNode = bvhNodes[0];

	for(int iteration=0;iteration<MAX_ITERATIONS;iteration++){
		if(BVH_IS_LEAF(curNode)){
			numTriangles = BVH_TRIANGLE_COUNT(curNode);

			// If this is a top BVH leaf we need to load the mesh instance
			// and transform all rays using its matrix.
			if( numTriangles == 0 ){
				meshInstanceId = BVH_MESH_INSTANCE_ID(curNode);
				meshInstance = meshInstances[meshInstanceId];

				// Push bottom BVH root to the stack and keep a record
				// of the current stack so that we know when we exit the 
				// bottom BVH
				meshBvhStackStartIndex = stackIndex;
				nodeStack[stackIndex++] = meshInstance.bvhRoot;

				// Transform rays without translating ray direction vector
				ray.origin.xyz = mul4x1(ray.origin.xyz, meshInstance.transformMat0, meshInstance.transformMat1, meshInstance.transformMat2, meshInstance.transformMat3);
				ray.dir.xyz = mul3x1(ray.dir.xyz, meshInstance.transformMat0.xyz, meshInstance.transformMat1.xyz, meshInstance.transformMat2.xyz);
			} else {
				// Intersect with all triangles using the Moller-Trumbore algorithm
				triStartIndex = BVH_TRIANGLE_INDEX(curNode);
				for(int vIndex = triStartIndex * 3; vIndex < (triStartIndex + numTriangles)*3;vIndex+=3){
					v0 = vertexList[vIndex].xyz;
					edge01 = vertexList[vIndex+1].xyz - v0;
					edge02 = vertexList[vIndex+2].xyz - v0;

					float3 pVec = cross(ray.dir.xyz, edge02);
					float det = dot(edge01, pVec);

					if (fabs(det) < INTERSECTION_EPSILON){
						continue;
					}
						
					float invDet = native_recip(det);

					// Calculate barycentric coords
					float3 tVec = ray.origin.xyz - v0;
					float u = dot(tVec, pVec) * invDet;
					if( u < 0.0f || u > 1.0f ){
						continue;
					}

					float3 qVec = cross(tVec, edge01);
					float v = dot(ray.dir.xyz, qVec) * invDet;
					if( v < 0.0f || u+v > 1.0f ){
						continue;
					}

					float t = dot(edge02, qVec) * invDet;
					if (t > INTERSECTION_EPSILON && t < intersection.wuvt.w){
						hitFlag[globalId] = 1;
						return;
					}
				}
			}
		} else {
			// Read children
			childNodes[0] = bvhNodes[BVH_LEFT_CHILD(curNode)];
			childNodes[1] = bvhNodes[BVH_RIGHT_CHILD(curNode)];

			// Check for intersection with first child
			invDir = native_recip(ray.dir.xyz);
			tmin = (childNodes[0].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[0].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float lHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);

			// Check for intersection with second child
			tmin = (childNodes[1].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[1].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float rHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);
			
			int wantLeft = lHitDist < FLT_MAX ? 1 : 0;
			int wantRight = rHitDist < FLT_MAX ? 1 : 0;

			if( wantLeft && wantRight ){
				nodeStack[stackIndex++] = wantLeft ? BVH_RIGHT_CHILD(curNode) : BVH_LEFT_CHILD(curNode);
				curNode = wantLeft ? childNodes[0] : childNodes[1];
				continue;
			} else if(wantLeft || wantRight){
				curNode = wantLeft ? childNodes[0] : childNodes[1];
				continue;
			} 
		} 
		
 		if(stackIndex == 0){
			// We are done
			hitFlag[globalId] = 0;
			return;
		} else if(stackIndex == meshBvhStackStartIndex){
			// If we exited from a bottom bvh tree we need to restore our ray
			ray.origin.xyz = origRayOrigin;
			ray.dir.xyz = origRayDir;
			meshBvhStackStartIndex = -1;
		}

		// Pop the next node off the stack
		curNode = bvhNodes[nodeStack[--stackIndex]];
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

	uint globalId = get_global_id(0);

	int stackIndex;
	int meshBvhStackStartIndex;
	uint nodeStack[BVH_MAX_STACK_SIZE];
	BvhNode curNode;
	BvhNode childNodes[2];
	int meshInstanceId;
	MeshInstance meshInstance;

	// triangle intersection vars
	 float3 v0, edge01, edge02;

	// Node bbox and leaf primitive intersection vars
	float3 invDir, tmin, tmax, rmin, rmax;
	float minmax, maxmin;
	int triStartIndex, numTriangles;

	// Fetch ray
	Ray	ray = rays[globalId];
	float3 origRayOrigin = ray.origin.xyz;
	float3 origRayDir = ray.dir.xyz;
	
	// Setup stack
	stackIndex = 0;
	meshBvhStackStartIndex = -1;
	curNode = bvhNodes[0];

	for(int iteration=0;iteration<MAX_ITERATIONS;iteration++){
		if(BVH_IS_LEAF(curNode)){
			numTriangles = BVH_TRIANGLE_COUNT(curNode);

			// If this is a top BVH leaf we need to load the mesh instance
			// and transform all rays using its matrix.
			if( numTriangles == 0 ){
				meshInstanceId = BVH_MESH_INSTANCE_ID(curNode);
				meshInstance = meshInstances[meshInstanceId];

				// Push bottom BVH root to the stack and keep a record
				// of the current stack so that we know when we exit the 
				// bottom BVH
				meshBvhStackStartIndex = stackIndex;
				nodeStack[stackIndex++] = meshInstance.bvhRoot;

				// Transform rays without translating ray direction vector
				ray.origin.xyz = mul4x1(ray.origin.xyz, meshInstance.transformMat0, meshInstance.transformMat1, meshInstance.transformMat2, meshInstance.transformMat3);
				ray.dir.xyz = mul3x1(ray.dir.xyz, meshInstance.transformMat0.xyz, meshInstance.transformMat1.xyz, meshInstance.transformMat2.xyz);
			} else {
				// Intersect with all triangles using the Moller-Trumbore algorithm
				triStartIndex = BVH_TRIANGLE_INDEX(curNode);
				for(int vIndex = triStartIndex * 3; vIndex < (triStartIndex + numTriangles)*3;vIndex+=3){
					v0 = vertexList[vIndex].xyz;
					edge01 = vertexList[vIndex+1].xyz - v0;
					edge02 = vertexList[vIndex+2].xyz - v0;

					float3 pVec = cross(ray.dir.xyz, edge02);
					float det = dot(edge01, pVec);

					if (fabs(det) < INTERSECTION_EPSILON){
						continue;
					}
						
					float invDet = native_recip(det);

					// Calculate barycentric coords
					float3 tVec = ray.origin.xyz - v0;
					float u = dot(tVec, pVec) * invDet;
					if( u < 0.0f || u > 1.0f ){
						continue;
					}

					float3 qVec = cross(tVec, edge01);
					float v = dot(ray.dir.xyz, qVec) * invDet;
					if( v < 0.0f || u+v > 1.0f ){
						continue;
					}

					float t = dot(edge02, qVec) * invDet;
					if (t > INTERSECTION_EPSILON && t < intersection.wuvt.w){
						intersection.wuvt = (float4)(
								1.0f - (u+v),
								u,
								v,
								t
						);
						intersection.triIndex = vIndex / 3;
						intersection.meshInstance = meshInstanceId;
					}
				}
			}
		} else {
			// Read children
			childNodes[0] = bvhNodes[BVH_LEFT_CHILD(curNode)];
			childNodes[1] = bvhNodes[BVH_RIGHT_CHILD(curNode)];

			// Check for intersection with first child
			invDir = native_recip(ray.dir.xyz);
			tmin = (childNodes[0].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[0].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float lHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);

			// Check for intersection with second child
			tmin = (childNodes[1].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[1].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float rHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);
			
			int wantLeft = lHitDist < FLT_MAX ? 1 : 0;
			int wantRight = rHitDist < FLT_MAX ? 1 : 0;
			
			if( wantLeft && wantRight ){
				nodeStack[stackIndex++] = wantLeft ? BVH_RIGHT_CHILD(curNode) : BVH_LEFT_CHILD(curNode);
				curNode = wantLeft ? childNodes[0] : childNodes[1];
				
				continue;
			} else if(wantLeft || wantRight){
				curNode = wantLeft ? childNodes[0] : childNodes[1];
				continue;
			} 
		} 
		
 		if(stackIndex == 0){
			// We are done
			hitFlag[globalId] = intersection.wuvt.w < FLT_MAX ? 1 : 0;
			intersections[globalId] = intersection;
			return;
		} else if(stackIndex == meshBvhStackStartIndex){
			// If we exited from a bottom bvh tree we need to restore our ray
			ray.origin.xyz = origRayOrigin;
			ray.dir.xyz = origRayDir;
			meshBvhStackStartIndex = -1;
		}

		// Pop the next node off the stack
		curNode = bvhNodes[nodeStack[--stackIndex]];
	}
}

// Test for ray packet intersections with scene geometry. Sets an ouput flag to 
// indicate intersections and also emits intersection data for any found intersections.
// This kernel operates on a bundle of RAY_PACKET_SIZE rays in parallel. Stack
// operations are handled by the first thread in the local thread group.
__kernel void packetIntersectionQuery(
		__global Ray* rays,
		__global BvhNode* bvhNodes,
		__global MeshInstance* meshInstances,
		__global float4* vertexList,
		__global int* hitFlag,
		__global Intersection* intersections
		){

	Intersection intersection;
	intersection.wuvt.w = FLT_MAX;

	uint globalId = get_global_id(0);
	uint localId = get_local_id(0);

	// Shared data for all threads
	__local int stackIndex;
	__local int meshBvhStackStartIndex;
	__local uint nodeStack[BVH_MAX_STACK_SIZE];
	__local BvhNode curNode;
	__local  BvhNode childNodes[2];
	__local int scratchMemory[RAY_PACKET_SIZE];
	__local int meshInstanceId;
	__local MeshInstance meshInstance;

	// Shared triangle intersection vars
	__local float3 v0, edge01, edge02;

	// Node bbox and leaf primitive intersection vars
	float3 invDir, tmin, tmax, rmin, rmax;
	float minmax, maxmin;
	int triStartIndex, numTriangles;

	// Fetch ray
	Ray	ray = rays[globalId];
	float3 origRayOrigin = ray.origin.xyz;
	float3 origRayDir = ray.dir.xyz;
	
	// Thread 0 manages the stack; set initial values
	if(localId == 0){
		stackIndex = 0;
		meshBvhStackStartIndex = -1;
		curNode = bvhNodes[0];
	}

	for(int iteration=0;iteration<MAX_ITERATIONS;iteration++){
		if(BVH_IS_LEAF(curNode)){
			numTriangles = BVH_TRIANGLE_COUNT(curNode);

			// If this is a top BVH leaf we need to load the mesh instance
			// and transform all rays using its matrix.
			if( numTriangles == 0 ){
				if( localId == 0 ){
					meshInstanceId = BVH_MESH_INSTANCE_ID(curNode);
					meshInstance = meshInstances[meshInstanceId];

					// Push bottom BVH root to the stack and keep a record
					// of the current stack so that we know when we exit the 
					// bottom BVH
					meshBvhStackStartIndex = stackIndex;
					nodeStack[stackIndex++] = meshInstance.bvhRoot;
				}
				barrier(CLK_LOCAL_MEM_FENCE);

				// Transform rays without translating ray direction vector
				ray.origin.xyz = mul4x1(ray.origin.xyz, meshInstance.transformMat0, meshInstance.transformMat1, meshInstance.transformMat2, meshInstance.transformMat3);
				ray.dir.xyz = mul3x1(ray.dir.xyz, meshInstance.transformMat0.xyz, meshInstance.transformMat1.xyz, meshInstance.transformMat2.xyz);
			} else {
				// Intersect with all triangles using the Moller-Trumbore algorithm
				triStartIndex = BVH_TRIANGLE_INDEX(curNode);
				for(int vIndex = triStartIndex * 3; vIndex < (triStartIndex + numTriangles)*3;vIndex+=3){
					// Thread 0 fetches triangle data into local memory
					if(localId == 0 ){
						v0 = vertexList[vIndex].xyz;
						edge01 = vertexList[vIndex+1].xyz - v0;
						edge02 = vertexList[vIndex+2].xyz - v0;
					}
					barrier(CLK_LOCAL_MEM_FENCE);

					float3 pVec = cross(ray.dir.xyz, edge02);
					float det = dot(edge01, pVec);

					if (fabs(det) >= INTERSECTION_EPSILON){
						float invDet = native_recip(det);

						// Calculate barycentric coords
						float3 tVec = ray.origin.xyz - v0;
						float u = dot(tVec, pVec) * invDet;
						float3 qVec = cross(tVec, edge01);
						float v = dot(ray.dir.xyz, qVec) * invDet;
						float t = dot(edge02, qVec) * invDet;

						if (u >= 0.0f && 
							u <= 1.0f && 
							v >= 0.0f && 
							u+v <= 1.0f && 
							t > INTERSECTION_EPSILON && 
							t < intersection.wuvt.w){
							intersection.wuvt = (float4)(
									1.0f - (u+v),
									u,
									v,
									t
							);
							intersection.triIndex = vIndex / 3;
							intersection.meshInstance = meshInstanceId;
						}
					}
					barrier(CLK_LOCAL_MEM_FENCE);
				}
			}
		} else {
			// Read children in parallel
			if( localId < 2 ){
				childNodes[localId] = bvhNodes[localId == 0 ? BVH_LEFT_CHILD(curNode) : BVH_RIGHT_CHILD(curNode)];

				// Clear the first 4 scratchMemory locations
				scratchMemory[localId] = 0;
				scratchMemory[localId+2] = 0;
			}

			barrier(CLK_LOCAL_MEM_FENCE);

			// Check for intersection with first child
			invDir = native_recip(ray.dir.xyz);
			tmin = (childNodes[0].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[0].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float lHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);

			// Check for intersection with second child
			tmin = (childNodes[1].minExtent.xyz - ray.origin.xyz) * invDir;
			tmax = (childNodes[1].maxExtent.xyz - ray.origin.xyz) * invDir;
			rmin = fmin(tmin, tmax);
			rmax = fmax(tmin, tmax);
			minmax = fmin( fmin(rmax.x, rmax.y), rmax.z);
			maxmin = fmax( fmax(rmin.x, rmin.y), rmin.z);
			float rHitDist = minmax < 0 || maxmin > minmax ? FLT_MAX : (maxmin >= ray.origin.w ? FLT_MAX : maxmin);
			
			// If scratchMemory[i] is TRUE then at least one ray wants to:
			// [0] visit none of the nodes
			// [1] visit only left node
			// [2] visit only right node
			// [3] visit both nodes
			int wantLeft = lHitDist < FLT_MAX ? 1 : 0;
			int wantRight = rHitDist < FLT_MAX ? 1 : 0;
			//printf("[%d] wantLeft: %d, wantRight: %d\n", localId, wantLeft, wantRight);
			scratchMemory[2*wantRight + wantLeft] = 1;

			// Wait for all threads to encode their traversal preference
			barrier(CLK_LOCAL_MEM_FENCE);

			if( scratchMemory[RAY_VISIT_BOTH_NODES] || (scratchMemory[RAY_VISIT_LEFT_NODE] && scratchMemory[RAY_VISIT_RIGHT_NODE])){
				// For each thread, set its scratchMemory location to -1 if 
				// ray prefers the left node and to 1 if the ray prefers the right node.
				scratchMemory[localId] = wantLeft || lHitDist < rHitDist ? -1 : 1;
				//printf("[%2d] scratchMemory -> %d\n", localId, scratchMemory[localId]);
				barrier(CLK_LOCAL_MEM_FENCE);

				// run a parallel reduction on scratchMemory. We are using
				// sequential addressing to avoid bank conflicts
				// see: https://docs.nvidia.com/cuda/samples/6_Advanced/reduction/doc/reduction.pdf
				for (unsigned int s=HALF_RAY_PACKET_SIZE; s>0; s>>=1) {
					if (localId < s) {
						scratchMemory[localId] += scratchMemory[localId + s];
					}
					barrier(CLK_LOCAL_MEM_FENCE);
				}

				// scratchMemory[0] sign indicates which node should we visit first
				if( localId == 0 ){
					nodeStack[stackIndex++] = scratchMemory[0] < 1 ? BVH_RIGHT_CHILD(curNode) : BVH_LEFT_CHILD(curNode);
					curNode = scratchMemory[0] < 1 ? childNodes[0] : childNodes[1];
				}

				barrier(CLK_LOCAL_MEM_FENCE);
				continue;
			} else if(scratchMemory[RAY_VISIT_LEFT_NODE] || scratchMemory[RAY_VISIT_RIGHT_NODE]) {
				if( localId == 0 ){
					curNode = scratchMemory[1] ? childNodes[0] : childNodes[1];
				}

				barrier(CLK_LOCAL_MEM_FENCE);
				continue;
			} 
		} 
		
 		if(stackIndex == 0){
			// We are done
			hitFlag[globalId] = intersection.wuvt.w < FLT_MAX ? 1 : 0;
			intersections[globalId] = intersection;
			return;
		} else if(stackIndex == meshBvhStackStartIndex){
			// If we exited from a bottom bvh tree we need to restore our ray
			ray.origin.xyz = origRayOrigin;
			ray.dir.xyz = origRayDir;
			
			if(localId == 0 ){
				meshBvhStackStartIndex = -1;
			}
		}

		barrier(CLK_LOCAL_MEM_FENCE);

		// Pop the next node pointer off the stack and read it into local memory
		if( localId == 0 ){
			curNode = bvhNodes[nodeStack[--stackIndex]];
		}

		barrier(CLK_LOCAL_MEM_FENCE);
	}
}

#endif
