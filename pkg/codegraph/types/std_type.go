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

	// c++的
	"vector": {}, "map": {}, "set": {}, "list": {}, "queue": {}, "stack": {}, "deque": {}, "priority_queue": {},
	"pair": {}, "tuple": {}, "auto": {}, "decltype": {}, "function": {}, "lambda": {}, "bind": {}, "reference": {},
	"pointer": {}, "array": {}, "string": {}, "wstring": {}, "wchar_t": {}, "bool": {},
	// C++标准库类型，包含带std::和不带std::的写法
	"std::vector":         {},
	"std::map":            {},
	"std::set":            {},
	"std::list":           {},
	"std::queue":          {},
	"std::stack":          {},
	"std::deque":          {},
	"std::priority_queue": {},
	"std::pair":           {},
	"std::tuple":          {},
	"std::function":       {},
	"std::string":         {},
	"std::wstring":        {},
	"std::u16string":      {}, "u16string": {},
	"std::u32string": {}, "u32string": {},
	"std::array":      {},
	"std::unique_ptr": {}, "unique_ptr": {},
	"std::shared_ptr": {}, "shared_ptr": {},
	"std::weak_ptr": {}, "weak_ptr": {},
	"std::optional": {}, "optional": {},
	"std::any": {}, "any": {},
	"std::variant": {}, "variant": {},
	"std::nullptr_t": {}, "nullptr_t": {},
	"std::size_t": {}, "size_t": {},
	"std::initializer_list": {}, "initializer_list": {},
	"std::unordered_map": {}, "unordered_map": {},
	"std::unordered_set": {}, "unordered_set": {},
	"std::multimap": {}, "multimap": {},
	"std::multiset": {}, "multiset": {},
	"std::bitset": {}, "bitset": {},
	"std::istringstream": {}, "istringstream": {},
	"std::ostringstream": {}, "ostringstream": {},
	"std::stringstream": {}, "stringstream": {},
	"std::ifstream": {}, "ifstream": {},
	"std::ofstream": {}, "ofstream": {},
	"std::fstream": {}, "fstream": {},
	"std::istream": {}, "istream": {},
	"std::ostream": {}, "ostream": {},
	"std::iostream": {}, "iostream": {},
	"std::cin": {}, "cin": {},
	"std::cout": {}, "cout": {},
	"std::cerr": {}, "cerr": {},
	"std::clog": {}, "clog": {},
	"std::move": {}, "move": {},
	"std::forward": {}, "forward": {},
	"std::enable_if": {}, "enable_if": {},
	"std::remove_reference": {}, "remove_reference": {},
	"std::add_const": {}, "add_const": {},
	"std::add_lvalue_reference": {}, "add_lvalue_reference": {},
	"std::add_rvalue_reference": {}, "add_rvalue_reference": {},
	"std::is_same": {}, "is_same": {},
	"std::integral_constant": {}, "integral_constant": {},
	// C++基础类型
	"char16_t": {}, "char32_t": {},
	"long long": {},
	"unsigned":  {}, "unsigned int": {}, "unsigned short": {}, "unsigned long": {}, "unsigned long long": {},
	"long double": {},"T":{},
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
