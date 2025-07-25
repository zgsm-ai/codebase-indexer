#include <string>
#include <vector>

namespace shapes_ns {
    struct ShapeData {};
}

class Widget {
public:
    int value = 10;
};

struct IShape {
    virtual double area() const = 0;
    virtual ~IShape() = default;
};

struct Circle : IShape {
    double radius;
    explicit Circle(double r) : radius(r) {}
    double area() const override { return 3.14159 * radius * radius; }
};

struct Point {
    int x = 1;
    int dummy = 0;
    int y = 2;
};

class Holder {
public:
    int a;
    int b = 5;

    const int c = 10;

    int* raw_ptr;
    int* raw_ptr2 = nullptr;

    int& ref_a = a;
    const std::string& name_ref;

    std::string text;
    std::string greeting = "hello";

    Point pt;
    Point pt_init = {1, 2};

    static int counter;

    mutable bool dirty_flag = false;

    constexpr static int version = 1;

    int nums[5];
    int nums_init[3] = {1, 2, 3};

    std::vector<int> vec;
    std::vector<int> vec_init = {1, 2, 3};

    bool flag = true;
};

int Holder::counter = 0;

int main() {
    int local_a = 5;
    float local_b(3.14);
    double local_c{2.718};
    char local_d;

    const int local_const = 42;
    volatile bool local_volatile_flag = true;

    int* local_ptr = &local_a;
    const char* local_cstr = "hello";
    float* local_float_ptr;

    int& local_ref = local_a;
    const std::string& local_str_ref = std::string("hello");

    int local_arr[5], *local_ptr2 = nullptr, *local_ptr3, &local_ref2 = local_a;
    int local_arr_init[] = {1, 2, 3};
    char local_chars[3] = {'A', 'B', 'C'};

    std::string local_name = "ChatGPT";
    std::vector<int> local_vec = {1, 2, 3};

    shapes_ns::ShapeData data;
    shapes_ns::ShapeData* data_ptr = &data;

    Widget w;
    Widget* w_ptr = new Widget();
    IShape* shape = new Circle{1.0};

    auto auto_int = 10;
    auto auto_str = local_name;
    auto& auto_vec_ref = local_vec;

    int loop_i = 0, loop_j = 1, loop_k;
    float loop_u = 1.0f, loop_v;

    struct TempPoint {
        int tx, ty;
    } temp_pt = {10, 20};

    return 0;
}