package com.example.test;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.function.Function;
// 定义一个复杂的接口，包含泛型、默认方法、静态方法、抽象方法和常量
interface ComplexInterface<T, R> extends java.io.Serializable {

    // 常量
    int DEFAULT_SIZE = 10;
    public static final String INTERFACE_NAME = "ComplexInterface";

    // 抽象方法
    R process(T input, int count, String description);

    // 泛型方法
    <E> List<E> convertToList(E[] array, int start, int end);

    // 默认方法
    default void printInfo(T input, boolean verbose, double value) {
        System.out.println("Input: " + input + ", verbose: " + verbose + ", value: " + value);
    }

    // 静态方法
    public static void showInterfaceName() {
        System.out.println("Interface: " + INTERFACE_NAME);
    }

    // 带有可变参数的抽象方法
    void logMessages(String... messages);

    // 嵌套接口
    // interface InnerInterface {
    //     void doSomething();
    // }
}

class TestClass {


    private int a;
    private int b;
    private int c;
    private int d;
    private int e;
    private int f;
    private int g;
    private int h;
    public int add(int e, int f) {
        return e + f;
    }
    public void test(int a, Function<String, Integer> func, Runnable r, List<String[]> arrs, int[] anums, int... nums) {
        int x = 1;
        double y = 3.14;
        String s = "hello";
        int[] arr = {1, 2, 3};
        List<String> list = new ArrayList<>();
        Map<String, Integer> map = new HashMap<>();
        Set<Double> set = new HashSet<>();
        Runnable localRunnable = new Runnable() {
            @Override
            public void run() {
                System.out.println("Inner Runnable");
            }
        };
        Function<Integer, String> intToString = (Integer i) -> String.valueOf(i);

        list.add("test");
        map.put("key", 100);
        set.add(2.71);

        for (int n : arr) {
            System.out.println(n);
        }

        if (a > 0) {
            System.out.println("a is positive");
        } else {
            System.out.println("a is not positive");
        }

        localRunnable.run();
        String result = intToString.apply(x);
        System.out.println(result);

        return;
    }

}

public class Example {
    public static void main(String[] args) {
        // 字符串字面量赋值
        String greeting = "Hi there!";
        // 新增变量定义
        int count = 10;
        double price = 99.99;
        boolean flag = true;
        char letter = 'A';
        List<String> names = new ArrayList<>();
        Map<String, Integer> scores = new HashMap<>();
        // 类的赋值（实例化）
        Person person = new com.example.test.Person("Alice", 30);
    }
}

class Person {
    String name;
    int age;

    public Person(String name, int age) {
        this.name = name;
        this.age = age;
    }
}