// 方式1：先声明后定义
struct Student {
    int id;
    char name[50];
    float score;
};

struct Student stu1;  // 定义变量

// 方式2：声明时同时定义变量
struct Student1 {
    int id;
    char name[50];
    float score;
} stu2, stu3;

// 方式3：匿名结构体（不指定标签名）
struct {
    int x;
    int y;
} point;


// 基本结构体
struct Person {
    char name[50];
    int age;
    float height;
    double weight;
    char gender;
    long phone;
};

// 嵌套结构体
struct Address {
    char street[100];
    char city[30];
    char state[20];
    int zip_code;
};

struct Employee {
    int id;
    char name[50];
    struct Address addr;
    float salary;
    short department;
    unsigned int hire_date;
};

// 位域结构体
struct Permission {
    unsigned int read : 1;
    unsigned int write : 1;
    unsigned int execute : 1;
    unsigned int admin : 1;
    unsigned int reserved : 4;
};

// 自引用结构体
struct ListNode {
    int data;
    float value;
    struct ListNode *next;
    struct ListNode *prev;
};

// 复杂嵌套结构体
struct Date {
    int year;
    short month;
    short day;
    char weekday[10];
};

struct Time {
    short hour;
    short minute;
    short second;
    long microsecond;
};

struct DateTime {
    struct Date date;
    struct Time time;
    char timezone[10];
};

// 联合体
union Data {
    int integer;
    float floating;
    double double_precision;
    char character;
    char string[20];
};

struct MixedData {
    int type;
    union Data data;
    char description[100];
};

// 函数指针结构体
struct MathOps {
    int (*add)(int, int);
    int (*subtract)(int, int);
    float (*multiply)(float, float);
    double (*divide)(double, double);
    void (*print_result)(const char*);
};

// 数组成员结构体
struct Student {
    int id;
    char name[50];
    float scores[10];
    double average;
    int grades[5][3];
    char subjects[10][20];
};

// 指针数组结构体
struct Database {
    char **table_names;
    int *table_sizes;
    void ***table_data;
    int table_count;
    long long total_records;
};

// 长整型和无符号类型
struct FileHeader {
    unsigned char signature[4];
    unsigned int version;
    unsigned long file_size;
    long long timestamp;
    short flags;
    unsigned short checksum;
};

// 枚举成员
enum Status {ACTIVE, INACTIVE, PENDING, SUSPENDED};
enum Priority {LOW = 1, MEDIUM = 5, HIGH = 10, CRITICAL = 15};

struct Task {
    int id;
    char title[100];
    enum Status status;
    enum Priority priority;
    struct DateTime created_time;
    struct DateTime deadline;
    char *description;
    const char *category;
};

// 柔性数组成员（C99）
struct Packet {
    int header;
    short length;
    char data[];  // 柔性数组
};

// 复杂的数据结构
struct TreeNode {
    int key;
    char value[50];
    double weight;
    struct TreeNode *left;
    struct TreeNode *right;
    struct TreeNode *parent;
    unsigned int height : 8;
    unsigned int color : 1;  // 红黑树颜色位
};

// 多层嵌套
struct University {
    char name[100];
    struct {
        char building[50];
        int room_number;
        struct {
            double latitude;
            double longitude;
        } coordinates;
    } location;
    struct {
        int faculty_count;
        int student_count;
        long long budget;
        float rating;
    } statistics;
};

// 匿名结构体成员
struct Config {
    int version;
    struct {
        char host[50];
        int port;
        unsigned short timeout;
    };  // 匿名结构体
    union {
        int debug_level;
        char log_file[100];
    };  // 匿名联合体
};

// 复杂指针结构体
struct Callback {
    void (*function)(void*);
    void *data;
    int (*validator)(const void*);
    void (*cleanup)(void*);
    char name[30];
};