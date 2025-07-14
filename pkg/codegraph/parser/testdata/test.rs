// ============== 一、标准库模块导入 ==============
use std::fmt;                      // 格式化输出
use std::fs::{File, self};          // 文件操作与文件系统
use std::io::{self, Read, Write, BufReader, BufWriter}; // 输入输出
use std::sync::{mpsc, Arc, Mutex, RwLock}; // 并发编程
use std::thread;                    // 线程
use std::collections::{HashMap, VecDeque, BinaryHeap, HashSet}; // 集合类型
use std::string::String;            // 字符串
use std::vec::Vec;                  // 动态数组
use std::option::Option::{Some, None}; // 可选值
use std::result::Result::{Ok, Err}; // 结果类型
use std::panic;                     // 异常处理
use std::iter::{Iterator, IntoIterator}; // 迭代器
use std::path::{Path, PathBuf};     // 路径处理
use std::borrow::Cow;               // 借用语义
use std::ops::{Add, Sub, Mul, Div}; // 运算符重载
use std::future::Future;            // 异步编程
use std::time::{Duration, Instant}; // 时间处理
use std::sync::atomic::{AtomicUsize, Ordering}; // 原子操作

// 导入第三方库示例（需在Cargo.toml中声明）
// extern crate regex;
// use regex::Regex;

// ============== 二、常量与类型别名 ==============
const PI: f64 = 3.141592653589793;
const MAX_SIZE: u32 = 1024;

// 类型别名
type Point2D = (f64, f64);
type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

// ============== 三、所有权与借用 ==============
fn ownership_demo() {
    // 移动语义
    let s1 = String::from("hello");
    let s2 = s1; // s1的所有权转移给s2，s1不再可用
    // println!("s1: {}", s1); // 编译错误：s1已被移动

    // 克隆
    let s3 = String::from("world");
    let s4 = s3.clone(); // 深拷贝
    println!("s3: {}, s4: {}", s3, s4);

    // 借用
    let s5 = String::from("borrow");
    let len = calculate_length(&s5); // 借用s5的不可变引用
    println!("Length of s5: {}", len);

    // 可变借用
    let mut s6 = String::from("mutable");
    change(&mut s6); // 借用可变引用
    println!("s6: {}", s6);

    // 切片
    let s = String::from("hello world");
    let hello = &s[0..5];
    let world = &s[6..11];
    println!("Substrings: {} and {}", hello, world);
}

fn calculate_length(s: &String) -> usize {
    s.len()
}

fn change(s: &mut String) {
    s.push_str(", world!");
}

// ============== 四、结构体与枚举高级特性 ==============
// 元组结构体
struct ColorRGB(u8, u8, u8);
// 类结构体
struct Point {
    x: f64,
    y: f64,
}
// 单元结构体
struct Nil;

// 枚举带数据
enum Shape {
    Circle { radius: f64, center: Point },
    Rectangle { width: f64, height: f64, color: ColorRGB },
    Triangle(Vec<Point>),
}

// 结构体实现
impl Point {
    // 关联函数（无self）
    fn origin() -> Self {
        Point { x: 0.0, y: 0.0 }
    }

    // 方法（有self）
    fn distance_to(&self, other: &Point) -> f64 {
        let dx = self.x - other.x;
        let dy = self.y - other.y;
        (dx*dx + dy*dy).sqrt()
    }
}

// ============== 五、trait与泛型高级特性 ==============
// trait带默认实现
trait Draw {
    fn draw(&self) {
        println!("Drawing a generic shape");
    }
    fn area(&self) -> f64;
}

// 泛型trait
trait Container<T> {
    fn add(&mut self, item: T);
    fn get(&self, index: usize) -> Option<&T>;
}

// 为类型实现trait
impl Draw for Shape {
    fn area(&self) -> f64 {
        match self {
            Shape::Circle { radius, .. } => PI * radius * radius,
            Shape::Rectangle { width, height, .. } => width * height,
            Shape::Triangle(points) => {
                // 简化的三角形面积计算
                if points.len() >= 3 {
                    let p1 = &points[0];
                    let p2 = &points[1];
                    let p3 = &points[2];
                    0.5 * ((p2.x - p1.x) * (p3.y - p1.y) - (p3.x - p1.x) * (p2.y - p1.y)).abs()
                } else {
                    0.0
                }
            }
        }
    }
}

// 泛型结构体
struct Vector<T> {
    data: Vec<T>,
}

// 为泛型结构体实现trait
impl<T> Container<T> for Vector<T> {
    fn add(&mut self, item: T) {
        self.data.push(item);
    }
    fn get(&self, index: usize) -> Option<&T> {
        self.data.get(index)
    }
}

// 关联类型
trait IteratorExt {
    type Item;
    fn next(&mut self) -> Option<Self::Item>;
}

// ============== 六、错误处理高级特性 ==============
fn read_config() -> Result<String> {
    let path = Path::new("config.toml");
    if !path.exists() {
        return Err("Config file not found".into());
    }

    let file = File::open(path)?;
    let mut reader = BufReader::new(file);
    let mut content = String::new();
    reader.read_to_string(&mut content)?;

    Ok(content)
}

// 组合错误处理
fn process_data() -> Result<()> {
    let config = read_config()?;
    println!("Config loaded: {}", config.len());
    Ok(())
}

// ============== 七、宏与元编程 ==============
// 使用内置宏
fn macro_demo() {
    // 向量宏
    let v = vec![1, 2, 3, 4, 5];
    println!("Vector: {:?}", v);

    // 格式化宏
    println!("{} + {} = {}", 2, 3, 5);

    // 条件编译宏
    #[cfg(debug_assertions)]
    println!("Debug mode enabled");

    // 自定义宏（简化版）
    my_macro!(10, 20, add); // 展开为 println!("30")
}

// 自定义宏定义（需放在模块顶部）
macro_rules! my_macro {
    ($a:expr, $b:expr, add) => {
        println!("{}", $a + $b);
    };
    ($a:expr, $b:expr, sub) => {
        println!("{}", $a - $b);
    };
}

// ============== 八、模块系统 ==============
// 模块定义（可分离为独立文件）
mod graphics {
    pub mod shapes {
        pub struct Circle {
            pub radius: f64,
        }

        impl Circle {
            pub fn area(&self) -> f64 {
                super::PI * self.radius * self.radius
            }
        }
    }

    pub const PI: f64 = 3.14159;
}

// 使用模块
fn module_demo() {
    // 导入模块成员
    use graphics::shapes::Circle;
    let circle = Circle { radius: 5.0 };
    println!("Circle area: {}", circle.area());

    // 访问模块常量
    println!("PI from module: {}", graphics::PI);
}

// ============== 九、并发编程高级特性 ==============
fn concurrency_advanced() {
    // 原子操作
    let counter = Arc::new(AtomicUsize::new(0));
    let mut handles = vec![];

    for _ in 0..10 {
        let counter = Arc::clone(&counter);
        let handle = thread::spawn(move || {
            let old = counter.fetch_add(1, Ordering::SeqCst);
            println!("Thread {} incremented to {}", old, counter.load(Ordering::SeqCst));
        });
        handles.push(handle);
    }

    for handle in handles {
        handle.join().unwrap();
    }

    // 读写锁
    let map = Arc::new(RwLock::new(HashMap::new()));
    let map_clone = Arc::clone(&map);
    let writer = thread::spawn(move || {
        let mut w = map_clone.write().unwrap();
        w.insert("key", "value");
    });

    let reader = thread::spawn(move || {
        let r = map.read().unwrap();
        if let Some(value) = r.get("key") {
            println!("Read value: {}", value);
        }
    });

    writer.join().unwrap();
    reader.join().unwrap();
}

// ============== 十、异步编程（需nightly版） ==============
#[cfg(feature = "async")]
async fn async_demo() -> Result<()> {
    // 模拟异步操作
    let future = async {
        println!("Async task started");
        tokio::time::sleep(Duration::from_secs(1)).await;
        println!("Async task completed");
        "Result"
    };

    let result = future.await;
    println!("Async result: {}", result);
    Ok(())
}

// ============== 十一、主函数 ==============
fn main() {
    // 调用补充的语法示例
    ownership_demo();

    // 结构体使用
    let p1 = Point { x: 3.0, y: 4.0 };
    let p2 = Point::origin();
    println!("Distance between points: {}", p1.distance_to(&p2));

    // 枚举使用
    let circle = Shape::Circle {
        radius: 5.0,
        center: Point { x: 0.0, y: 0.0 }
    };
    println!("Circle area: {}", circle.area());

    // 泛型容器
    let mut vec = Vector { data: vec![] };
    vec.add(10);
    vec.add(20);
    if let Some(item) = vec.get(0) {
        println!("First item: {}", item);
    }

    // 错误处理
    match process_data() {
        Ok(()) => println!("Data processed successfully"),
        Err(e) => println!("Error: {}", e),
    }

    // 宏调用
    macro_demo();

    // 模块调用
    module_demo();

    // 并发高级特性
    concurrency_advanced();

    // 异步编程（需启用特性）
    #[cfg(feature = "async")]
    {
        tokio::runtime::Runtime::new().unwrap().block_on(async_demo());
    }

    // 原代码示例
    let rect = Rectangle { width: 30, height: 50 };
    println!("Rectangle area: {}", rect.area());

    let numbers = vec![34, 50, 25, 100, 65];
    println!("Largest number: {}", largest(&numbers));

    match read_file() {
        Ok(contents) => println!("File contents: {}", contents),
        Err(e) => println!("Couldn't read file: {}", e),
    }

    closure_demo();

    let string1 = String::from("abcd");
    let string2 = "xyz";
    let result = longest(string1.as_str(), string2);
    println!("The longest string is {}", result);

    concurrency_demo();
}

// 原代码中的函数
struct Rectangle {
    width: u32,
    height: u32,
}

enum Color {
    Red,
    Green,
    Blue,
}

trait Area {
    fn area(&self) -> u32;
}

impl Area for Rectangle {
    fn area(&self) -> u32 {
        self.width * self.height
    }
}

fn largest<T: PartialOrd + Copy>(list: &[T]) -> T {
    let mut largest = list[0];
    for &item in list.iter() {
        if item > largest {
            largest = item;
        }
    }
    largest
}

fn closure_demo() {
    let list = vec![1, 2, 3];
    let only_borrows = || println!("From closure: {:?}", list);
    only_borrows();
}

fn read_file() -> io::Result<String> {
    let mut file = File::open("hello.txt")?;
    let mut contents = String::new();
    file.read_to_string(&mut contents)?;
    Ok(contents)
}

fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() {
        x
    } else {
        y
    }
}

fn concurrency_demo() {
    let (tx, rx) = mpsc::channel();

    thread::spawn(move || {
        let val = String::from("hi");
        tx.send(val).unwrap();
    });

    let received = rx.recv().unwrap();
    println!("Got: {}", received);
}