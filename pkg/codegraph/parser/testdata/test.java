package com.example.demo;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;

// 1. 枚举类型
enum Color {
    RED, GREEN, BLUE
}

// 2. 抽象类与继承
abstract class Shape {
    protected String name;

    public Shape(String name) {
        this.name = name;
    }

    public abstract double area();

    public String getName() {
        return name;
    }
}

// 3. 接口定义
interface Drawable {
    void draw();
}
public @interface MyAnnotation {
    String value() default "default value"; // 定义一个名为value的元素，带有默认值
}

// 4. 具体类实现接口
class Circle extends Shape implements Drawable {
    private double radius;

    public Circle(double radius) {
        super("Circle");
        this.radius = radius;
    }

    @Override
    public double area() {
        return Math.PI * radius * radius;
    }

    @Override
    public void draw() {
        System.out.println("Drawing " + name + " with radius " + radius);
    }

    // 5. 重写 equals 和 hashCode
    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Circle circle = (Circle) o;
        return Double.compare(circle.radius, radius) == 0;
    }

    @Override
    public int hashCode() {
        return Objects.hash(radius);
    }
}

// 6. 泛型类
class Container<T> {
    private T item;

    public Container(T item) {
        this.item = item;
    }

    public T getItem() {
        return item;
    }
}

// 7. 主类
@Data
public class Main {
    // 8. 静态变量
    private static int globalCounter = 0;

    // 9. 静态方法
    public static <T> void printItem(T item) {
        System.out.println("Item: " + item);
    }

    // 10. 主方法
    public static void main(String[] args) {
        // 11. 基本变量
        int number = 42;
        String message = "Hello, Java!";
        Color favoriteColor = Color.BLUE;

        // 12. 数组和集合
        int[] numbersArray = {1, 2, 3, 4, 5};
        List<Integer> numbersList = new ArrayList<>();
        for (int i = 1; i <= 5; i++) {
            numbersList.add(i);
        }

        // 13. 对象创建
        Circle circle = new Circle(5.0);
        Container<String> container = new Container<>("Hello");

        // 14. 控制流
        if (number > 0) {
            System.out.println(message);
        }

        // 15. 循环
        for (int num : numbersList) {
            System.out.print(num + " ");
        }
        System.out.println();

        // 16. 多态调用
        Shape shape = circle;
        System.out.println("Area: " + shape.area());

        // 17. 接口调用
        Drawable drawable = circle;
        drawable.draw();

        // 18. 泛型方法调用
        printItem(container.getItem());

        // 19. 异常处理
        try {
            if (args.length == 0) {
                throw new IllegalArgumentException("No arguments provided");
            }
        } catch (IllegalArgumentException e) {
            System.out.println("Error: " + e.getMessage());
        } finally {
            globalCounter++;
        }

        // 20. 字符串操作
        String result = "Result: " + (globalCounter > 0 ? "Positive" : "Zero");
        System.out.println(result);

        // 21. 匿名内部类
        Runnable runnable = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running from anonymous class");
            }
        };
        new Thread(runnable).start();
    }
}