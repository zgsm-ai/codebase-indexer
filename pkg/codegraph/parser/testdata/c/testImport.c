// 各种#include情况

// 1. 标准库头文件
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

// 2. 用户自定义头文件
#include "myheader.h"
#include "utils.h"
#include "project_config.h"
#include "utils.h"
#include "main_module.h"

// 3. 条件包含
#ifdef DEBUG
#include <assert.h>
#endif

// 4. 系统特定头文件
#include <unistd.h>
#include <sys/types.h>

// 5. 网络编程头文件
#include <sys/socket.h>
#include <netinet/in.h>

// 6. 第三方库头文件
#include <curl/curl.h>

// 7. 包含保护
#ifndef HEADER_H
#define HEADER_H
#include "config.h"
#endif

// 8. 条件编译包含
#if defined(WIN32)
#include <windows.h>
#elif defined(__linux__)
#include <pthread.h>
#endif

// 9. 错误处理头文件
#include <errno.h>
#include <signal.h>

// 10. 时间处理头文件
#include <time.h>