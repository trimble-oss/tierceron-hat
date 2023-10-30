package tap

var penseEyeMap map[string]string = map[string]string{}
var penseCodeMap map[string]string = map[string]string{}

func TapEyeRemember(penseIndex, memory string) {
	penseEyeMap[penseIndex] = memory
}

func PenseCode(penseCode string) (string, bool) {
	if _, penseCodeOk := penseCodeMap[penseCode]; penseCodeOk {
		delete(penseCodeMap, penseCode)
		return penseCode, penseCodeOk
	} else {
		// Might be a feather
		return "", false
	}
}
