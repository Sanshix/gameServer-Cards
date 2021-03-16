package util

// IsStringArrayContain 判断一个字符串数组是否包含指定元素
func IsStringArrayContain(
	array []string,
	expected string,
) bool {
	for _, val := range array {
		if val == expected {
			return true
		}
	}
	return false
}

// RemoveElementFromStringArray 从一个字符串数组中移除元素
func RemoveElementFromStringArray(
	array []string,
	removeElement string,
) []string {
	ret := []string{}
	for _, val := range array {
		if val != removeElement {
			ret = append(ret, val)
		}
	}
	return ret
}
