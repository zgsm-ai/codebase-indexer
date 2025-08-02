#include <vector>
#include <optional>
#include <memory>
#include <iostream>
#include <math.h>
class Animal {
private:
    std::string name;
    int age;

public:
    Animal(const std::string& name, int age) : name(name), age(age) {}
    
    std::string getName() const { return name; }
    virtual void speak() const { std::cout << "Animal sound\n"; }
};
class Shape {
public:
    virtual double area() const = 0;      // 纯虚函数
    virtual double perimeter() const = 0;
    virtual ~Shape() {}
};

class Circle : public Shape {
private:
    double radius;

public:
    Circle(double r) : radius(r) {}

    double area() const override {
        return 3.14159 * radius * radius;
    }

    double perimeter() const override {
        return 2 * 3.14159 * radius;
    }
};
class Flyable {
public:
    virtual void fly() const = 0;
};

class Swimmable {
public:
    virtual void swim() const = 0;
};

class Duck : public Animal, Flyable, private Swimmable {
public:
    Duck(const std::string& name, int age) : Animal(name, age) {}

    void speak() const override {
        std::cout << "Quack!\n";
    }

    void fly() const override {
        std::cout << "Duck is flying\n";
    }

    void swim() const override {
        std::cout << "Duck is swimming\n";
    }
};
class Outer {
public:
    int outerValue;

    class Inner {
    public:
        int innerValue;
        void show() {
            std::cout << "Inner value\n";
        }
    };
};
template <typename T>
class Box {
protected:
    T value;

public:
    Box(T val) : value(val) {}
    virtual T getValue() const { return value; }
};

template <typename T>
class LabeledBox : public Box<T> {
private:
    std::string label;

public:
    LabeledBox(T val, const std::string& lbl) : Box<T>(val), label(lbl) {}

    std::string getLabel() const { return label; }
};
struct Point {
    double x, y;

    Point(double x = 0, double y = 0) : x(x), y(y) {}

    virtual double distanceFromOrigin() const {
        return std::sqrt(x * x + y * y);
    }
};

struct ColoredPoint : public Point {
    std::string color;

    ColoredPoint(double x, double y, const std::string& c) : Point(x, y), color(c) {}

    void print() const {
        std::cout << "Point (" << x << ", " << y << ") Color: " << color << "\n";
    }
};


class Config {
private:
    std::string filePath;
    std::vector<std::pair<std::string, int>> settings;
    std::unique_ptr<int> version;

public:

    void setFilePath(const std::string& path) {
        filePath = path;
    }

    void addSetting(const std::string& key, int value) {
        settings.emplace_back(key, value);
    }
};
class MathUtil {
public:
    static constexpr double PI = 3.1415926;

    static double square(double x) {
        return x * x;
    }

    static double cube(double x) {
        return x * x * x;
    }
};
#include <iostream>

class Logger {
public:
    void log(const std::string& message) const {
        std::cout << "[LOG] " << message << std::endl;
    }
};

class Serializable {
public:
    virtual std::string serialize() const = 0;
};

class User : public Logger, public Serializable {
private:
    std::string name;
    int age;

public:
    User(const std::string& name, int age) : name(name), age(age) {}

    std::string serialize() const override {
        return "{ \"name\": \"" + name + "\", \"age\": " + std::to_string(age) + " }";
    }

    void printInfo() {
        log("Serializing user...");
        std::cout << serialize() << std::endl;
    }
};

int main() {
    User user("Alice", 30);
    user.printInfo();
}
#include <iostream>

struct Position {
    double x = 0;
    double y = 0;

    void move(double dx, double dy) {
        x += dx;
        y += dy;
    }
};

struct Drawable {
    virtual void draw() const = 0;
};

struct Circle1 : public Position, public Drawable {
    double radius = 1.0;
    Circle1(double x, double y, double r) {
        this->x = x;
        this->y = y;
        radius = r;
    }

    void draw() const override {
        std::cout << "Drawing circle at (" << x << ", " << y << ") with radius " << radius << std::endl;
    }
};



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

class Derived1 : public Outer<Base<Inner<int>>> {};
class Derived2 : virtual public Base1, public Base2 {};

typedef int MyInt,B,C,D;
typedef char* String, A;
typedef struct Person PersonAlias;

typedef struct {
    void** items;
    size_t size;
    size_t capacity;
    size_t item_size;
} GenericArray;

typedef struct TagNode {
    char* tag;
    struct TagNode* parent;
    struct TagNode* first_child;
    struct TagNode* next_sibling;
    void* data;
} TagNode;