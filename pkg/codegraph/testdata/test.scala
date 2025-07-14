// 1. 包与导入
package com.example.scala

import scala.math.{Pi, sqrt}

// 2. 样例类 (自动生成 equals, hashCode, toString)
case class Point(x: Double, y: Double) {
  def move(dx: Double, dy: Double): Point = Point(x + dx, y + dy)
}

// 3. 特质 (Trait)
trait Shape {
  def area: Double
  def name: String
  def describe: String = s"This is a $name"
}

// 4. 枚举 (Scala 3)
enum Color:
  case Red, Green, Blue

// 5. 类继承与特质混入
class Circle(radius: Double) extends Shape with Serializable {
  override def area: Double = Pi * radius * radius
  override def name: String = "Circle"
}

// 6. 泛型与高阶函数
def processList[T](list: List[T], f: T => Unit): Unit = {
  list.foreach(f)
}

// 7. 集合操作
val numbers = List(1, 2, 3, 4, 5)
val squared = numbers.map(x => x * x)
val sum = numbers.reduce(_ + _)
val even = numbers.filter(_ % 2 == 0)

// 8. 模式匹配
def getColorName(color: Color): String = color match {
  case Color.Red => "Red"
  case Color.Green => "Green"
  case Color.Blue => "Blue"
}

// 9. 函数式错误处理
def divide(a: Double, b: Double): Either[String, Double] = {
  if (b == 0) Left("Division by zero")
  else Right(a / b)
}

// 10. 闭包
def multiplier(factor: Int): Int => Int = {
  x => x * factor
}

// 11. 元组
val person = ("Alice", 30)
val (name, age) = person

// 12. 隐式转换
implicit class StringUtils(s: String) {
  def quote: String = s"'$s'"
}

// 13. 选项类型
def findPerson(id: Int): Option[String] = {
  if (id > 0) Some(s"Person-$id")
  else None
}

// 14. 递归函数
def factorial(n: Int): Int = {
  if (n <= 1) 1
  else n * factorial(n - 1)
}

// 15. 并发与 Future
import scala.concurrent.{Future, ExecutionContext}
import scala.concurrent.ExecutionContext.Implicits.global

def asyncOperation: Future[String] = Future {
  Thread.sleep(1000)
  "Async result"
}

// 16. 流处理
val stream = Stream.from(1).take(5)
val streamSquared = stream.map(_ * _)

// 17. 类型边界
def max[T <: Ordered[T]](a: T, b: T): T = {
  if (a >= b) a else b
}

// 18. 偏函数
val dividePF: PartialFunction[(Int, Int), Int] = {
  case (a, b) if b != 0 => a / b
}

// 19. 模式匹配与提取器
object Even {
  def unapply(n: Int): Boolean = n % 2 == 0
}

def checkNumber(n: Int): String = n match {
  case Even() => "Even"
  case _ => "Odd"
}

// 20. 主程序
@main def runExample(): Unit = {
  // 测试 Point
  val p = Point(3, 4)
  println(s"Point: ${p.move(1, 1)}")

  // 测试 Circle
  val circle = new Circle(5)
  println(s"Circle area: ${circle.area}")

  // 测试集合
  println(s"Squared: ${squared}")
  println(s"Sum: ${sum}")

  // 测试模式匹配
  println(s"Color: ${getColorName(Color.Blue)}")

  // 测试错误处理
  divide(10, 2) match {
    case Right(result) => println(s"Result: ${result}")
    case Left(error) => println(s"Error: ${error}")
  }

  // 测试闭包
  val double = multiplier(2)
  println(s"Double of 5: ${double(5)}")

  // 测试隐式转换
  println("Hello".quote)

  // 测试选项类型
  findPerson(1).foreach(println)

  // 测试递归
  println(s"Factorial of 5: ${factorial(5)}")

  // 测试并发
  asyncOperation.foreach(println)

  // 测试流
  println(s"Stream: ${streamSquared.toList}")

  // 测试偏函数
  println(s"Divide: ${dividePF(10, 2)}")

  // 测试提取器
  println(s"Number check: ${checkNumber(4)}")

  // 等待异步操作完成
  Thread.sleep(2000)
}