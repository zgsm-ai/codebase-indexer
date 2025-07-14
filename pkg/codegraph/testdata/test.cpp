#include <iostream>
#include <string>
#include <vector>
#include <memory>
#include "a.h"

#define PI 3.14159
constexpr int MAX = 100;
using std;

int add(int a, int b) ;

// 1. 命名空间定义
namespace Geometry {

enum class Color { RED, GREEN, BLUE };

struct Point {
    int x = 0;
    int y = 0;
    void move(int dx, int dy) { x += dx; y += dy; }
};

class Shape {
protected:
    std::string name;
public:
    Shape(const std::string& n) : name(n) {}
    virtual double area() const = 0;
    virtual ~Shape() = default;
};

class Circle : public Shape {
private:
    double radius;
public:
    Circle(double r) : Shape("Circle"), radius(r) {}
    double area() const override { return PI * radius * radius; }
};

} // namespace Geometry

// 2. 另一个命名空间
namespace Util {

template<typename T>
T max(T a, T b) {
    return a > b ? a : b;
}

void swap(int& a, int& b) {
    int temp = a;
    a = b;
    b = temp;
}

} // namespace Util

int globalVar = 10;

int main() {
    using namespace Geometry; // 引入命名空间

    int num = 42;
    std::string message = "Hello, C++!";
    std::vector<int> numbers = {1, 2, 3, 4, 5};

    // 3. 使用命名空间成员
    Point p{10, 20};
    p.move(5, 5);

    std::unique_ptr<Shape> circle = std::make_unique<Circle>(5.0);

    if (num > 0) {
        std::cout << message << std::endl;
    }

    for (int i : numbers) {
        std::cout << i << " ";
    }
    std::cout << std::endl;

    std::cout << "Area: " << circle->area() << std::endl;

    // 4. 显式调用命名空间函数
    std::cout << "Max: " << Util::max(10, 20) << std::endl;

    int a = 5, b = 3;
    Util::swap(a, b);
    std::cout << "Swapped: a=" << a << ", b=" << b << std::endl;

    std::cout << "Point: (" << p.x << ", " << p.y << ")" << std::endl;

    return 0;
}

