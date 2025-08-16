package utils

// UniqueIntSlice 删除重复的整数
func UniqueIntSlice(slice []int) []int {
	uniqueSlice := make([]int, 0, len(slice))
	uniqueMap := make(map[int]struct{})
	for _, str := range slice {
		if _, ok := uniqueMap[str]; !ok {
			uniqueMap[str] = struct{}{}
			uniqueSlice = append(uniqueSlice, str)
		}
	}
	return uniqueSlice
}

// UniqueInt64Slice 删除重复的int64
func UniqueInt64Slice(slice []int64) []int64 {
	uniqueSlice := make([]int64, 0, len(slice))
	uniqueMap := make(map[int64]struct{})
	for _, str := range slice {
		if _, ok := uniqueMap[str]; !ok {
			uniqueMap[str] = struct{}{}
			uniqueSlice = append(uniqueSlice, str)
		}
	}
	return uniqueSlice
}
