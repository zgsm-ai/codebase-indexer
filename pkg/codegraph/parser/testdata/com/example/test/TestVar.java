package com.example.test3;



import javax.swing.*;
import java.util.*;
class MyClass {

}
class Person {
    String name;
    int age;
    Person(String name, int age) {
        this.name = name;
        this.age = age;
    }
}
public class TypeDemo {
    // void 类型（用于方法）
    public void doSomething() {
        System.out.println("Doing something...");
        // 基本类型
        int number = 10;
        double price = 99.99;
        boolean isActive = true;
        char initial = 'A';
        MyClass[] array = new MyClass[10];
        // 标准库类型
        String name = "Alice";
        Integer age = 30;
        List<String> tags = new ArrayList<>();
        Map<String, Integer> scoreMap = new HashMap<>();

        // 自定义类型
        Person person = new Person("Bob", 25);

        // 数组类型
        int[] numbers = {1, 2, 3};
        String[] names = {"Tom", "Jerry"};
        Person[] people = new Person[5];

        // 泛型类型（标准库 + 自定义）
        List<Person> personList = new ArrayList<>();
        Map<String, Person> personMap = new HashMap<>();


        // 匿名类
        Runnable task = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running...");
            }
        };
        // 通配符类型
        List<? extends Person> wildcardList = new ArrayList<>();
    }
}
