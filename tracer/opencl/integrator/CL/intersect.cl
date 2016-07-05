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

#endif
