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


    int* p_int = new int(10);
    p_obj = new MyClass();

    struct Point1 { int x, y; };
    Point1 p = {10, 20};
    int x_coord = p.x;

    Point* ptr_p = &p;
    int y_coord = ptr_p->y;

    auto my_lambda = [](int x) { return x * 2; };


    int a = (int)3.14;
    double d = 3.14;
    int i = static_cast<int>(d);                    // init_declarator: value = static_cast_expression

    float f = static_cast<float>(42);               // init_declarator
    result = static_cast<long>(i) * 1000;           // assignment_expression: right = binary_expression 包含 static_cast

    class Base {};
    class Derived : public Base {};
    Derived* dp = new Derived();
    Base* bp = static_cast<Base*>(dp);              // 向上转型


    class Base {
    public:
        virtual ~Base() = default;
    };
    class Derived : public Base {};

    struct Point {
        int x = 0, y = 0;
    };

    Point* pt = new Point;           // 默认构造
    Base* bp = new Derived();
    Derived* dp = dynamic_cast<Derived*>(bp);       // init_declarator: value = dynamic_cast_expression

    Base* another_bp = getBasePtr();
    Derived* safe_dp = dynamic_cast<Derived*>(another_bp);
    result_ptr = dynamic_cast<Derived*>(safe_dp);   // assignment_expression: right = dynamic_cast_expression

    const int* cp = &value;
    int* mp = const_cast<int*>(cp);                 // 移除 const，init_declarator

    const std::string& str_ref = getString();
    std::string& mutable_ref = const_cast<std::string&>(str_ref);
    buffer = const_cast<char*>(str_ref.c_str());    // assignment_expression

    int x = 42;
    int* p = &x;
    uintptr_t addr = reinterpret_cast<uintptr_t>(p);  // 指针转整数，init_declarator

    void* vp = reinterpret_cast<void*>(&x);           // int* 转 void*
    func_ptr = reinterpret_cast<FuncType*>(code_ptr); // 函数指针重解释

    char* raw = reinterpret_cast<char*>(&float_val);  // 查看 float 的字节表示

    MyType* mt = reinterpret_cast<MyType*>(p);

    auto x = foo<int, double>(42);
    auto y = Bar<std::vector<int>, MyClass*>();
    auto z = create_map<std::string, std::vector<int>>();

    foo<int>(42);
    
    A* a1 = new ns::A();
    // 带参数构造
    B* b1 = new ns::B(42);


    return 0;
}