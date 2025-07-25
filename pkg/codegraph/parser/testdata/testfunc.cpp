#include <vector>
#include <map>
#include <string>
#include <tuple>
#include <utility>
#include <iostream>

class MyClass {
public:
    int id;
    std::string name;
};

struct MyStruct {
    double x;
    double y;
};
template <typename T>
class Box {
public:
    T value;
};

// 基本类型
int getInt() {
    return 42;
}
void doNothing() {
    std::cout << "doNothing called" << std::endl;
}
float getFloat() {
    return 3.14f;
}

// 指针和引用
char* getBuffer(int count) {
    static char buffer[128] = {0};
    return buffer;
}
const std::string& getNameRef1() {
    static std::string name = "hello";
    return name;
}
const std::string* getNameRef2() {
    static std::string name = "world";
    return &name;
}

// 标准模板容器
std::vector<int> getVector() {
    return {1, 2, 3};
}
std::map<std::string, float> getMap() {
    return {{"a", 1.0f}, {"b", 2.0f}};
}

// 嵌套模板类型
std::map<std::string, std::vector<int>> getComplexMap() {
    return {{"x", {1, 2}}, {"y", {3, 4}}};
}

// 自定义模板类型
Box<double> getBox() {
    Box<double> box;
    box.value = 1.23;
    return box;
}
vector<int> getBoxOfVector() {
    return vector<int>{4, 5, 6};
}
std::map<std::string, std::vector<int>> getComplexMap1(
    const std::map<std::string, int>& simpleMap,
    std::vector<std::string> names,
    const std::string& key,
    int count
) {
    return {{"z", {count}}};
}

// pair 和 tuple 类型
std::pair<int, std::string> getPair() {
    return {7, "pair"};
}
std::tuple<int, std::string, float> getTuple(int count /*= 10*/) {
    return std::make_tuple(count, "tuple", 1.0f);
}

// auto 和 decltype
auto getAutoValue() -> std::vector<std::string> {
    return {"auto", "value"};
}
// decltype(getInt()) getAnotherInt() {
//     return 100;
// }

// 带默认参数和命名空间返回值
std::vector<std::map<std::string, int>> getNames(int count /*= 10*/) {
    return std::vector<std::map<std::string, int>>(count);
}

// 带 const 和 noexcept 的返回值
const std::vector<int> & getConstVector() noexcept {
    static std::vector<int> v = {9, 8, 7};
    return v;
}

int* func0() {
    static int x = 0;
    return &x;
}
MyClass& func1(MyStruct* arg1, int arg2) {
    static MyClass obj;
    obj.id = arg2;
    return obj;
}
template<typename T>
T func2(T arg1) {
    return arg1;
}
template<typename T>
T* func3(T* arg1, const std::vector<T>& arg2) {
    if (!arg2.empty()) return const_cast<T*>(&arg2[0]);
    return arg1;
}
std::string func4() {
    return "func4";
}
std::vector<int> func5(const std::string& arg1) {
    return {static_cast<int>(arg1.size())};
}
MyStruct* func6() {
    static MyStruct s{1.0, 2.0};
    return &s;
}
const MyClass& func7(const MyClass& arg1, int* arg2) {
    return arg1;
}
template<typename T>
std::vector<T*> func8(const std::vector<T*>& arg1) {
    return arg1;
}
std::map<int, MyClass*> func9(int arg1, MyClass* arg2) {
    return {{arg1, arg2}};
}
int func10() {
    return 10;
}
MyClass func11(MyStruct arg1) {
    MyClass c;
    c.id = static_cast<int>(arg1.x + arg1.y);
    return c;
}
template<typename T>
T& func12(const T& arg1, int arg2) {
    static T t = arg1;
    return t;
}
std::vector<std::string> func13() {
    return {"a", "b"};
}
const std::vector<MyClass*>& func14(const std::vector<MyClass*>& arg1, int arg2) {
    return arg1;
}
template<typename T>
std::vector<T> func15() {
    return {};
}
template<typename T>
std::vector<T*> func16(T* arg1, int arg2) {
    return {arg1};
}
MyStruct& func17() {
    static MyStruct s{3.0, 4.0};
    return s;
}
const int& func18(const int& arg1) {
    return arg1;
}
std::vector<std::map<int, MyClass*>> func19() {
    return {};
}
