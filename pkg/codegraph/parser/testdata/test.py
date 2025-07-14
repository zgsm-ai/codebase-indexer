# ============== 一、基础模块导入 ==============
import math                          # 数学计算
import cmath                         # 复数计算
import numbers                       # 数字抽象基类
import operator                      # 运算符重载
import functools                     # 高阶函数与装饰器工具
import itertools                     # 高效迭代工具
import collections                   # 扩展数据类型
from collections import (           # 具名元组/双端队列等
    namedtuple, deque, Counter, OrderedDict, defaultdict
)
import typing                        # 类型注解
from typing import (                # 类型注解扩展
    List, Dict, Tuple, Set, Optional, Callable, Generator, Any, Union
)
import sys                           # 系统交互
import os                            # 操作系统接口
import pathlib                       # 路径对象操作
import time                          # 时间处理
import datetime as dt                # 日期时间
from datetime import datetime, timedelta, date
import calendar                      # 日历处理
import random                        # 随机数生成
import hashlib                       # 哈希算法
import base64                        # 二进制编码
import zlib                          # 数据压缩
import pickle                        # 数据序列化
import shelve                        # 持久化字典
import json                          # JSON处理
import xml.etree.ElementTree as ET   # XML解析
import re                            # 正则表达式
import fileinput                     # 多行文件处理
import tempfile                      # 临时文件操作
import shutil                        # 高级文件操作
import glob                          # 文件路径匹配
import argparse                      # 命令行参数解析
import sysconfig                     # Python配置信息

# ============== 二、科学计算与数据处理 ==============
import numpy as np                   # 数值计算
import pandas as pd                  # 数据处理
import matplotlib.pyplot as plt      # 绘图
import scipy as sp                   # 科学计算库（需第三方安装）
import scikit_learn as skl           # 机器学习（需第三方安装）

# ============== 三、并发与网络 ==============
import threading                     # 多线程
import multiprocessing as mp         # 多进程
from multiprocessing import Pool, Queue
import concurrent.futures            # 线程/进程池
import asyncio                       # 异步编程
import socket                        # 网络编程
import http.server                   # HTTP服务器
import urllib.request                # URL请求
import urllib.parse                  # URL解析
import smtplib                       # 邮件发送
import email.message                 # 邮件构造
import ftplib                        # FTP客户端
import telnetlib                     # Telnet客户端

# ============== 四、图形界面与交互 ==============
import tkinter as tk                 # GUI基础库
from tkinter import ttk              # 增强控件
import pygame                        # 游戏开发（需第三方安装）
import wx                              # wxPython GUI（需第三方安装）

# ============== 五、测试与调试 ==============
import unittest                      # 单元测试框架
import pytest                        # 测试框架（需第三方安装）
import doctest                       # 文档测试
import pdb                           # 调试器
import traceback                     # 错误追踪
import sys                           # 异常处理
import logging                       # 日志系统
from logging import config, handlers

# ============== 六、系统与进程 ==============
import subprocess                    # 子进程管理
import psutil                        # 系统监控（需第三方安装）
import platform                      # 平台信息
import resource                      # 资源使用
import getpass                       # 密码输入
import getopt                        # 命令行选项解析

# ============== 七、元编程与高级特性 ==============
import types                         # 类型工具
import inspect                       # 反射机制
import importlib                     # 动态导入
from importlib import metadata       # 包元数据（Python 3.8+）
import __future__                    # 未来特性
import sys                          # 模块系统

# ============== 八、常量定义 ==============
PI = math.pi                        # 圆周率
E = math.e                          # 自然常数
MAX_INT = sys.maxsize               # 最大整数
MIN_INT = -sys.maxsize - 1          # 最小整数
TODAY = dt.date.today()             # 今日日期

# ============== 九、函数定义（含高阶函数与装饰器） ==============
def add(a: int, b: int) -> int:
    """基础加法函数"""
    return a + b

@functools.lru_cache(maxsize=128)    # 缓存装饰器
def fibonacci(n: int) -> int:
    """递归计算斐波那契数列（演示闭包与缓存）"""
    if n < 2:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

def timer(func: Callable) -> Callable:
    """计时装饰器（演示闭包）"""
    def wrapper(*args, **kwargs):
        start = time.perf_counter()
        result = func(*args, **kwargs)
        end = time.perf_counter()
        print(f"[TIMER] {func.__name__} took {end-start:.6f}s")
        return result
    return wrapper

# 偏函数示例
def power(base: float, exp: float) -> float:
    """幂运算"""
    return base ** exp
square = functools.partial(power, exp=2)

# ============== 十、类与面向对象编程 ==============
# 抽象基类
from abc import ABC, abstractmethod
class Shape(ABC):
    def __init__(self, name: str):
        self.name = name

    @abstractmethod
    def area(self) -> float:
        """计算面积（需子类实现）"""
        pass

    def describe(self) -> str:
        """对象描述"""
        return f"This is a {self.name}"

# 继承与多态
class Circle(Shape):
    def __init__(self, radius: float):
        super().__init__("Circle")
        self.radius = radius

    def area(self) -> float:
        return PI * self.radius ** 2

# 多重继承
class Movable:
    def move(self, dx: int, dy: int) -> None:
        print(f"Moved by ({dx}, {dy})")

class Colored:
    def __init__(self, color: str):
        self.color = color

class ColoredCircle(Circle, Movable, Colored):
    def __init__(self, radius: float, color: str):
        Circle.__init__(self, radius)
        Colored.__init__(self, color)

    def describe(self) -> str:
        return f"{super().describe()} with color {self.color}"

# 魔术方法
class Vector:
    def __init__(self, x: float, y: float):
        self.x = x
        self.y = y

    def __add__(self, other: 'Vector') -> 'Vector':
        return Vector(self.x + other.x, self.y + other.y)

    def __sub__(self, other: 'Vector') -> 'Vector':
        return Vector(self.x - other.x, self.y - other.y)

    def __mul__(self, scalar: float) -> 'Vector':
        return Vector(self.x * scalar, self.y * scalar)

    def __str__(self) -> str:
        return f"Vector({self.x}, {self.y})"

    def __len__(self) -> int:
        return 2  # 模拟向量维度

# 元类
class MetaClass(type):
    def __new__(mcs, name, bases, namespace):
        namespace['created_at'] = datetime.now()
        return super().__new__(mcs, name, bases, namespace)

class MetaDemo(metaclass=MetaClass):
    pass

# ============== 十一、数据结构与生成器 ==============
# 生成器函数
def fib_generator(n: int) -> Generator[int, None, None]:
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b

# 生成器表达式
even_gen = (x for x in range(10) if x % 2 == 0)

# 具名元组
Point = namedtuple('Point', ['x', 'y', 'z'])
p = Point(1, 2, 3)

# 双端队列
dq = deque([1, 2, 3])
dq.append(4)
dq.appendleft(0)

# ============== 十二、异常处理与上下文管理器 ==============
# 自定义异常
class CustomError(Exception):
    pass

# 异常处理
try:
    result = 10 / 0
except ZeroDivisionError as e:
    print(f"Error: {e}")
except Exception as e:
    print(f"Unexpected error: {e}")
else:
    print("No error occurred")
finally:
    print("Cleanup completed")

# 上下文管理器（类实现）
class FileHandler:
    def __init__(self, path: str, mode: str):
        self.path = path
        self.mode = mode
        self.file = None

    def __enter__(self):
        self.file = open(self.path, self.mode)
        return self.file

    def __exit__(self, exc_type, exc_val, exc_tb):
        if self.file:
            self.file.close()

# 上下文管理器（函数实现）
from contextlib import contextmanager
@contextmanager
def timer_context(name: str):
    start = time.perf_counter()
    try:
        yield
    finally:
        end = time.perf_counter()
        print(f"[{name}] Execution time: {end-start:.6f}s")

# ============== 十三、函数式编程与高阶函数 ==============
my_list = [1, 2, 3, 4, 5]

# 映射
doubled = list(map(lambda x: x * 2, my_list))

# 过滤
evens = list(filter(lambda x: x % 2 == 0, my_list))

# 归约
from functools import reduce
sum_total = reduce(lambda a, b: a + b, my_list)

# 生成器表达式
squared = (x**2 for x in my_list)

# ============== 十四、并发编程 ==============
# 多线程
def thread_task(name: str, delay: int):
    for i in range(3):
        time.sleep(delay)
        print(f"Thread {name}: {i}")

thread = threading.Thread(target=thread_task, args=("A", 1))
thread.start()
thread.join()

# 多进程
def process_task(n: int):
    print(f"Process {n}: {os.getpid()}")

process = mp.Process(target=process_task, args=(1,))
process.start()
process.join()

# 异步编程
async def async_task(name: str, delay: int):
    await asyncio.sleep(delay)
    print(f"Async {name} completed")

async def main():
    await asyncio.gather(
        async_task("A", 1),
        async_task("B", 0.5)
    )

# ============== 十五、文件与系统操作 ==============
# 文件读写
with open("data.txt", "w") as f:
    f.write("Hello, Python!\n")
with open("data.txt", "r") as f:
    content = f.read()

# 目录操作
current_dir = os.getcwd()
files = os.listdir(current_dir)
pathlib.Path("new_dir").mkdir(exist_ok=True)

# 命令行参数
parser = argparse.ArgumentParser(description="Example program")
parser.add_argument("-v", "--verbose", action="store_true", help="Enable verbose mode")
parser.add_argument("input", type=str, help="Input file path")
args = parser.parse_args()

# ============== 十六、网络编程 ==============
# 简单HTTP请求
import urllib.request
try:
    with urllib.request.urlopen("https://www.python.org") as response:
        html = response.read().decode("utf-8")
        print(f"Page length: {len(html)} characters")
except urllib.error.URLError as e:
    print(f"Network error: {e}")

# ============== 十七、测试与日志 ==============
# 单元测试
class TestMath(unittest.TestCase):
    def test_add(self):
        self.assertEqual(add(2, 3), 5)
    def test_fibonacci(self):
        self.assertEqual(fibonacci(5), 5)

# 日志配置
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    handlers=[
        logging.FileHandler("app.log"),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)
logger.info("Application started")

# ============== 十八、主程序入口 ==============
if __name__ == "__main__":
    # 类型注解示例
    def greet(name: Optional[str] = None) -> str:
        return f"Hello, {name or 'World'}!"
    print(greet("Python"))

    # 类实例化
    circle = Circle(5.0)
    print(circle.describe())
    print(f"Area: {circle.area():.2f}")

    # 生成器使用
    print("Fibonacci sequence:")
    for num in fib_generator(10):
        print(num, end=" ")
    print()

    # 向量运算
    v1 = Vector(1, 2)
    v2 = Vector(3, 4)
    print(v1 + v2)
    print(v1 * 2)

    # 元类演示
    obj = MetaDemo()
    print(f"Created at: {obj.created_at}")

    # 异步编程调用
    asyncio.run(main())

    # 测试运行
    unittest.main(argv=[''], verbosity=2, exit=False)