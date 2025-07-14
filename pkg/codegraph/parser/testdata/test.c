#include <stdio.h>      // 标准输入输出库
#include <stdlib.h>     // 标准库（包含动态内存分配）
#include "b.h"
// 1. 宏定义与常量
#define PI 3.14159
const int MAX_VALUE = 100;

// 2. 全局变量与外部声明
int global_var = 10;
extern int external_var; // 假设在其他文件中定义

// 3. 结构体与枚举
struct Point {
    int x, y;
};

// 1. 新增：联合(Union)定义
union Data {
    int i;
    float f;
    char str[20];
};


enum Color { RED, GREEN, BLUE };

// 4. 函数声明与定义
int add(int a, int b);               // 函数原型声明
void swap(int *a, int *b);           // 指针参数
struct Point move(struct Point p);   // 返回结构体

// 5. 主函数
int main(int argc, char *argv[]) {
    // 6. 局部变量与类型
    int a = 5, b = 3;
    float result;
    char str[] = "Hello, C!";
    enum Color my_color = GREEN;

    // 7. 控制流：if-else
    if (a > b) {
        printf("a 大于 b\n");
    } else {
        printf("a 小于等于 b\n");
    }

    // 8. 循环：for
    for (int i = 0; i < 5; i++) {
        printf("循环迭代: %d\n", i);
    }

    // 9. 函数调用
    result = add(a, b) * PI;
    printf("计算结果: %.2f\n", result);

    // 10. 指针与内存操作
    int *ptr = &a;
    printf("指针值: %d\n", *ptr);

    swap(&a, &b);
    printf("交换后: a=%d, b=%d\n", a, b);

    // 11. 结构体操作
    struct Point p = {10, 20};
    struct Point new_p = move(p);
    printf("移动后坐标: (%d, %d)\n", new_p.x, new_p.y);

    // 12. 动态内存分配
    int *array = (int *)malloc(5 * sizeof(int));
    if (array == NULL) {
        fprintf(stderr, "内存分配失败\n");
        return 1;
    }

    for (int i = 0; i < 5; i++) {
        array[i] = i * 10;
    }

    // 13. 数组与指针遍历
    printf("动态数组: ");
    for (int i = 0; i < 5; i++) {
        printf("%d ", *(array + i));
    }
    printf("\n");

    // 14. 释放内存
    free(array);

    // 15. 命令行参数处理
    if (argc > 1) {
        printf("第一个参数: %s\n", argv[1]);
    }

    // 16. 返回值
    return 0;
}

// 函数定义
int add(int a, int b) {
    return a + b;
}

void swap(int *a, int *b) {
    int temp = *a;
    *a = *b;
    *b = temp;
}

struct Point move(struct Point p) {
    p.x += 5;
    p.y += 10;
    return p;
}

// 17. 注释：说明代码功能