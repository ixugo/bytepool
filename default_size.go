package bytepool

// SizePowerOfTwo returns a sequence of powers of 2 from 7 to 21
func SizePowerOfTwo() []int {
	return []int{
		128,     // 2^7
		256,     // 2^8
		512,     // 2^9
		1024,    // 2^10
		2048,    // 2^11
		4096,    // 2^12
		8192,    // 2^13
		16384,   // 2^14
		32768,   // 2^15
		65536,   // 2^16
		131072,  // 2^17
		262144,  // 2^18
		524288,  // 2^19
		1048576, // 2^20
		2097152, // 2^21
	}
}

// SizeStream returns a sequence of sizes optimized for streaming
func SizeStream() []int {
	return []int{
		1024,
		4096,
		16384,
		32768,
		65536,
		131072,
	}
}
