#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

// ==================== 类型定义 ====================

typedef struct {
    int id;
    char name[32];
    double score;
} Student;

typedef union {
    int as_int;
    float as_float;
    char as_char[4];
} DataUnion;

typedef void (*LoggerFunc)(const char*);

// ==================== 函数声明 ====================

// 1. 无参数 + 无返回值
void initialize_system();

// 2. 无参数 + 有返回值（返回指针）
char* get_default_config();

// 3. 多参数 + 无返回值（结构体 + 基本类型）
void log_student(const Student *s, int year, double adjustment);

// 4. 多参数 + 有返回值（混合类型 + 表达式）
double compute_weighted_average(double m1, double m2, double final, int bonus_active);

// 5. 指针参数 + 修改内容（模拟输出参数）
void get_timestamp_and_status(char *buffer, size_t buf_size, int *status_code);

// 6. 返回结构体（构造对象）
Student create_student(int id, const char *name, double score);

// 7. 函数指针作为参数（回调机制）
void run_with_logger(LoggerFunc logger, const char *msg);

// 8. 可变参数包装函数（调用 printf 风格）
void custom_log(const char *prefix, ...);

// 9. 返回联合体（高级用法）
DataUnion parse_raw_data(unsigned int raw_value);

// 10. 嵌套调用 + 表达式作为参数
int process_and_validate(int (*validator)(int), int input_level);

// 辅助函数：用于函数指针
int validate_level(int level) {
    return level >= 1 && level <= 10;
}

// ==================== main：每个函数调用一次（复杂表达式） ====================

int main() {

    // ✅ 1. 无参无返回值：初始化系统
    initialize_system();

    // ✅ 2. 无参有返回值：获取配置字符串（返回堆内存）
    char *config = get_default_config();
    free(config);  // 注意：malloc 出来要释放

    // ✅ 3. 多参数无返回值：日志记录（结构体 + 变量 + 表达式）
    Student s = {101, "Alice", 88.5};
    int current_year = 2025;
    double curve = 1.1 * (0.05);  // 动态调整系数
    log_student(&s, current_year + 1, curve);  // 表达式作为参数

    // ✅ 4. 多参数有返回值：计算加权平均（混合常量、变量、条件表达式）
    double midterm1 = 76.0, midterm2 = 82.0, final = 90.0;
    int is_honors = 1;
    double final_avg = compute_weighted_average(
        midterm1,
        midterm2,
        final + (is_honors ? 5.0 : 0.0),  // 表达式
        is_honors
    );

    // ✅ 5. 指针参数（模拟输出参数）
    char timestamp[64];
    int status;
    get_timestamp_and_status(timestamp, sizeof(timestamp), &status);

    // ✅ 6. 返回结构体：构造新学生（参数含字符串字面量）
    Student new_student = create_student(102, "Bob\0\0\0", 75.0 + fmax(0.0, final_avg - 80.0));
    // ✅ 7. 函数指针调用：传递函数作为回调
    run_with_logger(print_log, "System running...");  // 传本地函数

    // ✅ 8. 可变参数函数调用（模拟日志）
    custom_log("DEBUG", "User %s logged in from IP %s", "Alice", "192.168.1.100");


    // ✅ 9. 返回联合体：解析原始数据
    DataUnion data = parse_raw_data(0x41C80000);  // IEEE 754 float: 25.5

    // ✅ 10. 嵌套调用 + 函数指针参数（表达式中调用函数）
    int level = 7;
    int is_valid = process_and_validate(validate_level, level * 2 - 5);  // 表达式: 7*2-5 = 9

    struct MyStruct a = (struct MyStruct){.x = 1, .y = 2};
    return 0;
}

// ==================== 函数定义 ====================

void initialize_system() {
    printf("[SYS] Initializing...\n");
}

char* get_default_config() {
    char *cfg = malloc(64);
    if (cfg) strcpy(cfg, "theme=dark;lang=en;auto_save=1");
    return cfg;
}

void log_student(const Student *s, int year, double adjustment) {
    printf("[LOG] Student %s (ID:%d) - Year: %d, Adjustment: %.3f\n",
           s->name, s->id, year, adjustment);
}

double compute_weighted_average(double m1, double m2, double final, int bonus_active) {
    double bonus = bonus_active ? 3.0 : 0.0;
    return 0.2*m1 + 0.2*m2 + 0.5*final + bonus;
}

void get_timestamp_and_status(char *buffer, size_t buf_size, int *status_code) {
    snprintf(buffer, buf_size, "2025-04-05 10:30:45");
    *status_code = 200;  // 模拟成功状态
}

Student create_student(int id, const char *name, double score) {
    Student s;
    s.id = id;
    strncpy(s.name, name, sizeof(s.name) - 1);
    s.name[sizeof(s.name) - 1] = '\0';
    s.score = score > 100.0 ? 100.0 : (score < 0.0 ? 0.0 : score);
    return s;
}

void run_with_logger(LoggerFunc logger, const char *msg) {
    if (logger && msg) logger(msg);
}

void custom_log(const char *prefix, ...) {
    va_list args;
    va_start(args, prefix);
    printf("[%s] ", prefix);
    vprintf(va_arg(args, char*), &args);  // 简化：假设第一个可变参数是格式串
    printf("\n");
    va_end(args);
}

DataUnion parse_raw_data(unsigned int raw_value) {
    DataUnion u;
    u.as_int = raw_value;
    // 注意：这依赖于字节序，仅用于演示
    return u;
}

int process_and_validate(int (*validator)(int), int input_level) {
    return validator(input_level);
}