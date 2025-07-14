package main

import (
	"fmt"
	"math"
	"os"
)
import "strings"

// 全局变量
var globalVar int = 42

// 结构体定义
type Person struct {
	Name string
	Age  int
}

// 接口定义
type Shape interface {
	Area() float64
}

// 函数定义
func add(a, b int) int {
	return a + b
}

// 方法定义
func (p Person) SayHello() {
	fmt.Printf("Hello, my name is %s\n", strings.TrimSpace(p.Name))
}

// 实现接口
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func main() {
	// 变量声明与初始化
	var x int
	x = 10
	y := 20

	// 常量
	const pi = 3.14159

	// 数组
	arr := [3]int{1, 2, 3}

	// 切片
	slice := []int{4, 5, 6}
	slice = append(slice, 7)

	// 映射
	m := make(map[string]int)
	m["apple"] = 1
	m["banana"] = 2

	// 条件语句
	if x > y {
		fmt.Println("x is greater than y")
	} else {
		fmt.Println("x is less than or equal to y")
	}

	// 循环
	for i := 0; i < 3; i++ {
		fmt.Println(arr[i])
	}

	// 范围遍历
	for k, v := range m {
		fmt.Printf("%s: %d\n", k, v)
	}

	// 函数调用
	sum := add(x, y)
	fmt.Printf("Sum: %d\n", sum)

	// 结构体实例
	p := Person{Name: "Alice", Age: 30}
	p.SayHello()

	// 接口使用
	var s Shape = Circle{Radius: 5}
	fmt.Printf("Circle area: %.2f\n", s.Area())

	// 指针
	ptr := &x
	fmt.Printf("Value at pointer: %d\n", *ptr)

	// 错误处理
	file, err := os.Open("nonexistent.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer file.Close()
	}

	// 类型断言
	var i interface{} = "hello"
	s2, ok := i.(string)
	if ok {
		fmt.Printf("Type assertion: %s\n", s2)
	}

	// 通道
	ch := make(chan int)
	go func() {
		ch <- 42
		close(ch)
	}()
	for val := range ch {
		fmt.Printf("Received from channel: %d\n", val)
	}
}
