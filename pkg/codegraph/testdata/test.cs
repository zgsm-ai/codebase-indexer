using System;
using System.Collections.Generic;


namespace CSharpDemo {

	// 重载一元负号运算符
public static Point operator -(Point p)
{
    return new Point { X = -p.X, Y = -p.Y };
}


    public enum Color { Red, Green, Blue }

    public struct Point {
        public int X { get; set; }
        public int Y { get; set; }
    }

    public abstract class Shape {
        public string Name { get; }
        public Shape(string name) => Name = name;
        public abstract double Area();
    }

    public class Circle : Shape {
        private double Radius;
        public Circle(double radius) : base("Circle") => Radius = radius;
        public override double Area() => Math.PI * Radius * Radius;
    }

    // 1. 新增：带索引器的集合类
    public class ShapeCollection {
        private List<Shape> shapes = new List<Shape>();

        // 索引器定义
        public Shape this[int index] {
            get => shapes[index];
            set => shapes[index] = value;
        }

        public void Add(Shape shape) => shapes.Add(shape);
        public int Count => shapes.Count;
    }

    class Program {
        static void Main(string[] args) {
            // 2. 使用索引器
            var collection = new ShapeCollection();
            collection.Add(new Circle(5));
            collection.Add(new Circle(10));

            Console.WriteLine($"Collection size: {collection.Count}");
            Console.WriteLine($"First area: {collection[0].Area():F2}");

            // 3. 其他核心元素
            var point = new Point { X = 10, Y = 20 };
            Console.WriteLine($"Point: ({point.X}, {point.Y})");

            var numbers = new List<int> { 1, 2, 3 };
            numbers.ForEach(n => Console.Write($"{n} "));

            try {
                if (args.Length == 0) {
                    throw new ArgumentException("No args");
                }
            } catch (Exception ex) {
                Console.WriteLine($"\nError: {ex.Message}");
            }

            // 4. 匿名类型与 var
            var person = new { Name = "Alice", Age = 30 };
            Console.WriteLine($"Person: {person.Name}, {person.Age}");
        }
    }
}