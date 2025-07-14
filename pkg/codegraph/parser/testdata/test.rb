# 1. 模块与常量
module MathUtils
  PI = 3.14159
end

# 2. 类定义
class Shape
  attr_reader :name

  def initialize(name)
    @name = name
  end

  def area
    raise NotImplementedError, "Subclasses should implement this method"
  end

  def describe
    "This is a #{name}"
  end
end

# 3. 继承与方法重写
class Circle < Shape
  def initialize(radius)
    super("Circle")
    @radius = radius
  end

  def area
    MathUtils::PI * @radius**2
  end
end

# 4. 模块混入
module Drawable
  def draw
    "Drawing #{name} with radius #{@radius}"
  end
end

class Circle
  include Drawable
end

# 5. 枚举类 (使用模块)
module Color
  RED = :red
  GREEN = :green
  BLUE = :blue
end

# 6. 函数式编程
numbers = [1, 2, 3, 4, 5]
squared = numbers.map { |n| n**2 }
sum = numbers.reduce(0) { |acc, n| acc + n }

# 7. 块与闭包
def repeat(times)
  times.times { yield }
end

repeat(3) { puts "Hello Ruby!" }

# 8. 迭代器
class Fibonacci
  def initialize(limit)
    @limit = limit
  end

  def each
    a, b = 0, 1
    @limit.times do
      yield a
      a, b = b, a + b
    end
  end
end

# 9. 异常处理
begin
  result = 1 / 0
rescue ZeroDivisionError => e
  puts "Error: #{e.message}"
ensure
  puts "Cleanup code here"
end

# 10. 元编程
class DynamicClass
  def self.create_method(name)
    define_method(name) do
      "This is a dynamically created method: #{name}"
    end
  end
end

DynamicClass.create_method(:hello)
obj = DynamicClass.new
puts obj.hello

# 11. 哈希与符号
person = { name: "Alice", age: 30, hobbies: [:reading, :coding] }
puts person[:name]

# 12. 字符串插值
message = "Hello, #{person[:name]}! You are #{person[:age]} years old."
puts message

# 13. 条件表达式
status = person[:age] >= 18 ? "Adult" : "Minor"
puts status

# 14. 循环结构
1.upto(5) { |i| puts i }

# 15. 类方法与类变量
class Counter
  @@count = 0

  def self.increment
    @@count += 1
  end

  def self.count
    @@count
  end
end

Counter.increment
puts Counter.count

# 16. 单例方法
obj = Object.new
def obj.special_method
  "This is a special method for this instance only"
end

puts obj.special_method

# 17. 访问控制
class Secret
  def public_method
    "This is public"
  end

  protected

  def protected_method
    "This is protected"
  end

  private

  def private_method
    "This is private"
  end
end

# 18. 运算符重载
class Vector
  attr_reader :x, :y

  def initialize(x, y)
    @x = x
    @y = y
  end

  def +(other)
    Vector.new(@x + other.x, @y + other.y)
  end

  def to_s
    "(#{@x}, #{@y})"
  end
end

v1 = Vector.new(1, 2)
v2 = Vector.new(3, 4)
puts v1 + v2

# 19. 文件操作
File.open("test.txt", "w") do |f|
  f.write("Hello from Ruby!\n")
end

content = File.read("test.txt")
puts content

# 20. 正则表达式
text = "Hello, Ruby is amazing!"
match = text.match(/Ruby/)
puts match.to_s

# 21. 线程
thread = Thread.new do
  3.times do
    puts "Thread running..."
    sleep(0.5)
  end
end

thread.join

# 22. 类作为对象
class MyClass; end
puts MyClass.ancestors.inspect

# 23. 块参数
def execute(&block)
  block.call
end

execute { puts "Executing block..." }

# 24. 模块作为命名空间
module MyApp
  class User
    def initialize(name)
      @name = name
    end
  end
end

user = MyApp::User.new("Bob")

# 25. 可选参数
def greet(name = "World")
  "Hello, #{name}!"
end

puts greet
puts greet("Alice")

# 26. 多重赋值
a, b = 10, 20
puts "#{a}, #{b}"

# 27. 猴子补丁
class String
  def reverse_and_upcase
    self.reverse.upcase
  end
end

puts "hello".reverse_and_upcase

# 28. 链式方法调用
result = [1, 2, 3].map { |n| n * 2 }.select { |n| n > 3 }
puts result.inspect

# 29. 单例类
obj = Object.new
class << obj
  def special_method
    "This is a singleton method"
  end
end

puts obj.special_method

# 30. 主程序入口
if __FILE__ == $PROGRAM_NAME
  circle = Circle.new(5)
  puts circle.describe
  puts circle.area
  puts circle.draw

  Fibonacci.new(5).each { |n| puts n }
end