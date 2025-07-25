// demo.cpp —— 11 种常用调用，参数稍微多一点
#include <string>
#include <vector>

// 1. 自由函数
void freeFunction(int a, double b, char c) { (void)a; (void)b; (void)c; }

// 2. 命名空间函数
namespace MyNamespace {
void nsFunction(int x, int y, int z) { (void)x; (void)y; (void)z; }
}  // namespace MyNamespace

// 3. 类定义
struct MyClass {
    void memberFunction(int a, double b) { (void)a; (void)b; }
    void memberFunction1(int a, double b,char c) { (void)a; (void)b; (void)c; }
    static void staticFunction(int, int) {}
    int operator()(int, int, int, int) { return 0; }
};

int main() {
    MyClass obj;
    MyClass* ptr = &obj;


    // 1. 自由函数
    freeFunction(1, 2.5, 'A');

    // 2. 命名空间函数
    MyNamespace::nsFunction(1, 2, 3);

    // 3. 对象成员函数
    obj.memberFunction(10, 3.14);

    // 4. 指针成员函数
    ptr->memberFunction1(20, 2.718,'A');

    // 5. 静态成员函数
    MyClass::staticFunction(7, 8);

    // 6. 模板函数
    auto templatedFunction = [](auto a, auto b, auto c, auto d) { (void)a; (void)b; (void)c; (void)d; };
    templatedFunction(1, 2L, 3.0, 'x');

    // 7. lambda
    auto lambda = [](int a, int b, int c) { (void)a; (void)b; (void)c; };
    lambda(4, 5, 6);

    // 8. 函数指针
    void (*fp)(int, double, char) = freeFunction;
    fp(9, 1.2, 'Z');

    // 9. 函数对象
    obj(1, 2, 3, 4);

    // 10. append 链式首调
    // 11. at   链式次调
    std::string str;
    str.append("hello", 3).at(1);

    return 0;
}