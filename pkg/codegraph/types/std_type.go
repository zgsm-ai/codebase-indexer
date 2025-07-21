package types

var javaStdTypes = map[string]struct{}{
	"Map": {}, "List": {}, "Set": {}, "String": {}, "Integer": {}, "Double": {}, "Boolean": {},
	"Short": {}, "Long": {}, "Float": {}, "Character": {}, "Byte": {}, "Object": {}, "Runnable": {},
	"ArrayList": {}, "LinkedList": {}, "HashSet": {}, "TreeSet": {}, "HashMap": {}, "TreeMap": {},
	"Hashtable": {}, "LinkedHashMap": {}, "Queue": {}, "Deque": {}, "PriorityQueue": {},
	"Collections": {}, "Arrays": {}, "Date": {}, "Calendar": {}, "Optional": {},
	"File": {}, "InputStream": {}, "OutputStream": {}, "Reader": {}, "Writer": {},
	"BufferedReader": {}, "BufferedWriter": {}, "FileInputStream": {}, "FileOutputStream": {},
	"Path": {}, "Paths": {}, "Files": {}, "ByteBuffer": {},
	"URL": {}, "HttpURLConnection": {}, "Socket": {},
	// java基本数据类型
	"int": {}, "boolean": {}, "void": {},
	"byte": {}, "short": {}, "char": {}, "long": {}, "float": {}, "double": {},
}

// FilterCustomTypes 过滤类型名切片，只保留用户自定义类型
func FilterCustomTypes(typeNames []string) []string {
	// 标准库类型集合
	var customTypes []string
	for _, t := range typeNames {
		if _, isStd := javaStdTypes[t]; !isStd {
			customTypes = append(customTypes, t)
		}
	}
	if len(customTypes) == 0 {
		return []string{PrimitiveType}
	}
	return customTypes
}
