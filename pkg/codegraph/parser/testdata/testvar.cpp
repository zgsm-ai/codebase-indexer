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

// 基本枚举
enum Color {
    RED,
    GREEN,
    BLUE
};

// 带值的枚举
enum Status {
    PENDING = 0,
    RUNNING = 1,
    COMPLETED = 2
};

// 指定底层类型的枚举
enum Direction : int {
    NORTH = 1,
    SOUTH = 2,
    EAST = 3,
    WEST = 4
};

// 作用域枚举 (enum class)
enum class Priority {
    LOW,
    MEDIUM,
    HIGH
};

// 带底层类型的作用域枚举
enum class ErrorCode : unsigned int {
    SUCCESS = 0,
    FAILURE = 1,
    TIMEOUT = 2
};

// 匿名枚举
enum {
    MAX_SIZE = 100,
    MIN_SIZE = 1
};

// 嵌套在类中的枚举
class NetworkManager {
public:
    enum Protocol {
        HTTP,
        HTTPS,
        FTP
    };

    enum class State : char {
        DISCONNECTED = 'D',
        CONNECTING = 'C',
        CONNECTED = 'N'
    };
};

// 带位字段的枚举
enum Flags {
    READ = 1 << 0,
    WRITE = 1 << 1,
    EXECUTE = 1 << 2
};

// 复杂的枚举定义
enum class LogLevel : short {
    DEBUG = -1,
    INFO = 0,
    WARNING = 1,
    ERROR = 2,
    CRITICAL = 3
};

// 跨行枚举定义
enum DatabaseType {
    MYSQL,
    POSTGRESQL,
    SQLITE,
    ORACLE,
    MSSQL
};

// 带注释的枚举
enum FilePermission {
    NONE1 = 0,        // 无权限
    READ1 = 1,        // 读权限
    WRITE1 = 2,       // 写权限
    EXECUTE1 = 4      // 执行权限
};