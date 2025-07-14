package com.example.kotlindemo

// 1. 包与导入
import kotlin.math.PI
import kotlin.random.Random

// 2. 枚举类
enum class Color {
    RED, GREEN, BLUE
}

// 3. 数据类 (自动生成 equals, hashCode, toString)
data class Point(val x: Int, val y: Int) {
    fun move(dx: Int, dy: Int) = Point(x + dx, y + dy)
}

// 4. 密封类 (限制继承)
sealed class Shape {
    abstract val name: String
    abstract fun area(): Double
}

// 5. 继承与接口
class Circle(
    val radius: Double
) : Shape(), Drawable {
    override val name = "Circle"
    override fun area() = PI * radius * radius
    override fun draw() = println("Drawing $name with radius $radius")
}

interface Drawable {
    fun draw()
}

// 6. 函数式编程
val sum: (Int, Int) -> Int = { a, b -> a + b }
val square: (Int) -> Int = { it * it }

// 7. 扩展函数
fun String.repeat(times: Int): String {
    return this.repeat(times)
}

// 8. 泛型类
class Box<T>(private val item: T) {
    fun get() = item
}

// 9. 伴生对象 (静态成员)
object MathUtils {
    fun calculateCircumference(radius: Double) = 2 * PI * radius
}

// 10. 主函数
fun main() {
    // 11. 变量与类型推断
    val number: Int = 42
    var message = "Hello, Kotlin!"

    // 12. 集合
    val numbers = listOf(1, 2, 3, 4, 5)
    val mutableNumbers = mutableListOf(1, 2, 3)

    // 13. 条件表达式
    val result = if (number > 50) "Large" else "Small"

    // 14. 循环
    for (num in numbers) {
        println(num)
    }

    numbers.forEach { println(it) }

    // 15. 高阶函数
    val doubled = numbers.map(square)
    val evenSum = numbers.filter { it % 2 == 0 }.sum()

    // 16. 字符串模板
    println("$message The sum is $evenSum")

    // 17. 对象实例化 (无需 new 关键字)
    val point = Point(10, 20)
    val movedPoint = point.move(5, 5)

    // 18. 多态
    val shape: Shape = Circle(5.0)
    println("Area: ${shape.area()}")
    (shape as Drawable).draw()

    // 19. 空安全
    var nullableStr: String? = null
    println(nullableStr?.length ?: "String is null")

    // 20. 异常处理
    try {
        val value = numbers[10] // 会抛出 IndexOutOfBoundsException
    } catch (e: Exception) {
        println("Error: ${e.message}")
    }

    // 21. 范围表达式
    for (i in 1..5) {
        println(i)
    }

    // 22. 数据类特性
    val point1 = Point(1, 1)
    val point2 = Point(1, 1)
    println(point1 == point2) // true (值相等)

    // 23. 解构声明
    val (x, y) = point
    println("Coordinates: $x, $y")

    // 24. 单例模式 (对象声明)
    val utils = MathUtils
    println("Circumference: ${utils.calculateCircumference(3.0)}")

    // 25. 委托属性
    class Example {
        var p: String by Delegates.observable("initial") {
                prop, old, new -> println("Property changed: $old -> $new")
        }
    }

    // 26. 扩展属性
    val String.firstChar: Char
    get() = this[0]

    println("Kotlin".firstChar) // 'K'

    // 27. 带接收者的函数字面量
    val greet: String.() -> String = { "Hello, $this!" }
    println("World".greet()) // "Hello, World!"

    // 28. 协程 (简化示例)
    GlobalScope.launch {
        delay(1000L)
        println("Coroutine executed")
    }

    // 保持主线程运行，以便协程有机会执行
    Thread.sleep(2000L)
}