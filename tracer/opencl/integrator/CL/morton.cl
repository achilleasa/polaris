__constant uint mortonMasks2D[5] = { 0x0000FFFF, 0x00FF00FF, 0x0F0F0F0F, 0x33333333, 0x55555555 };

// Generate 2D morton code from coordinate pair. The implementation is 
// based on: https://github.com/Forceflow/libmorton
inline uint morton2d(uint2 coords){
	coords = (coords | coords << 16) & mortonMasks2D[0];
	coords = (coords | coords << 8) & mortonMasks2D[1];
	coords = (coords | coords << 4) & mortonMasks2D[2];
	coords = (coords | coords << 2) & mortonMasks2D[3];
	coords = (coords | coords << 1) & mortonMasks2D[4];
	return coords.x | (coords.y << 1);
}
