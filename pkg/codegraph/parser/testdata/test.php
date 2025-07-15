<?php
// 1. 声明严格类型
declare(strict_types=1);

// 2. 常量与命名空间
namespace App\Demo;
const MAX_ITEMS = 100;
define('BASE_URL', 'https://example.com');

// 3. 数据类型
$int = 42;
$float = 3.14;
$string = 'PHP';
$bool = true;
$null = null;
$array = [1, 'a', true];
$assoc = ['name' => 'Alice', 'age' => 30];

// 4. 类与继承
abstract class Model {
    protected string $table;

    public function __construct(string $table) {
        $this->table = $table;
    }

    abstract public function save(): bool;
}

// 5. 接口与实现
interface Cacheable {
    public function cacheKey(): string;
}

class User extends Model implements Cacheable {
    use Logger;
    public string $name;           // 公共属性，类型为字符串
    protected static int $count;   // 受保护的静态属性，类型为整数
    private ?array $data = [];     // 私有属性，可为 null，默认值为空数组
    public function __construct(
        private string $name,
        private int $age
    ) {
        parent::__construct('users');
    }

    public function save(): bool {
        $this->log("Saving user {$this->name}");
        return true;
    }

    public function cacheKey(): string {
        return "user:{$this->name}";
    }
}

// 6. Trait
trait Logger {
    public function log(string $message): void {
        echo "[$this->table] $message\n";
    }
}

// 7. 函数与类型声明
function add(int $a, int $b): int {
    return $a + $b;
}

$multiply = fn($a, $b) => $a * $b;

// 8. 控制结构
if ($int > 0) {
    echo "$string is great!\n";
} else {
    echo "Something's wrong\n";
}

// 9. 循环
foreach ($assoc as $key => $value) {
    echo "$key: $value\n";
}

// 10. 异常处理
try {
    if ($float < 3) {
        throw new InvalidArgumentException("Value too small");
    }
} catch (Exception $e) {
    echo "Error: {$e->getMessage()}\n";
}

// 11. 数组操作
$filtered = array_filter($array, 'is_string');
$mapped = array_map($multiply, [2, 4, 6], [3, 5, 7]);

// 12. 生成器
function counter(int $limit): Generator {
    for ($i = 0; $i < $limit; $i++) {
        yield $i;
    }
}

// 13. 匿名类
$calculator = new class {
    public function sum(int ...$numbers): int {
        return array_sum($numbers);
    }
};

// 14. 对象操作
$user = new User('Bob', 25);
$user->save();
echo "Cache key: {$user->cacheKey()}\n";

// 15. 闭包与作用域
$factor = 10;
$scaled = array_map(function($n) use ($factor) {
    return $n * $factor;
}, [1, 2, 3]);

// 16. 类型检查
var_dump(is_array($array));
var_dump(gettype($string));

// 17. 生成器使用
foreach (counter(3) as $num) {
    echo "Count: $num\n";
}

// 18. 静态方法
class Utils {
    public static function formatName(string $name): string {
        return ucwords(strtolower($name));
    }
}

echo Utils::formatName('jOHN DOE') . "\n";