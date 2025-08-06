#include <stdio.h>

// 枚举声明
enum Color {
    RED, GREEN, BLUE
};

enum Status {
    ACTIVE = 1,
    INACTIVE = 2,
    PENDING = 3
};

// 联合体声明
union Data {
    int i;
    float f;
    char str[20];
};

// 结构体声明
struct Point {
    int x;
    int y;
};

struct Person {
    // 基本数据类型
    int age;
    float height;
    double weight;
    char gender;
    unsigned int id;
    
    // 数组类型
    char name[50];
    int scores[5];
    float grades[3];
    
    // 指针类型
    char *email;
    int *data_ptr;
    
    // 结构体和枚举类型
    struct Point location;
    enum Color favorite_color;
    enum Status status;
    
    // 联合体类型
    union Data extra_info;
};

// 变量声明 - 基本类型
int a, b, c;
float x, y = 3.14f;
double d1, d2 = 2.718;
char ch1, ch2 = 'A';
unsigned int uid1, uid2 = 1000U;

// 数组声明
int arr[10];
float matrix[3][3];
char str[100];
double values[5] = {1.1, 2.2, 3.3, 4.4, 5.5};

// 指针声明
int *ptr1, *ptr2;
char *str_ptr;
float *float_ptr = NULL;
void *void_ptr;

// 结构体变量声明
struct Person person1;
struct Person person2 = {25, 5.9f, 70.5, 'M', 12345};
struct Person person_array[10];
struct Person *person_ptr;

// 联合体变量声明
union Data data1;
union Data data2 = {.f = 3.14f};

// 枚举变量声明
enum Color color1;
enum Status status1 = ACTIVE;

// 静态变量声明
static int static_var = 42;
static struct Point static_point = {100, 200};

// const变量声明
const int const_int = 100;
const float const_float = 1.414f;

int main() {
    // 局部变量声明
    int local_int, local_int2 = 10;
    char local_arr[50];
    struct Person local_person = {30, 6.2f, 80.0, 'F', 54321};
    int values[] = {10, 20, 30};
    int *ptrs[3] = { &values[0], &values[1], &values[2] };
    int *ptrs2[3];
    return 0;
}