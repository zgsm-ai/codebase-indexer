class TestClass:
    # 1. 基本实例方法
    def simple_method(self):
        pass

    # 2. 带参数的实例方法
    def method_with_params(self, a, b, c):
        return a + b + c

    # 3. 带默认参数的实例方法
    def method_with_defaults(self, a, b=10, c="default"):
        return a + b

    # 4. 带类型注解的实例方法
    def typed_method(self, x: int, y: str) -> bool:
        return len(y) > x

    # 5. 带复杂类型注解的实例方法
    def complex_typed_method(
        self, 
        data: list[dict[str, int]], 
        callback: callable
    ) -> tuple[int, str]:
        return (len(data), "processed")

    # 6. 可变参数实例方法
    def variadic_args(self, *args):
        return sum(args)

    # 7. 关键字可变参数实例方法
    def variadic_kwargs(self, **kwargs):
        return kwargs

    # 8. 混合参数实例方法
    def mixed_params(self, a, b=5, *args, **kwargs):
        return a, b, args, kwargs

    # 9. 仅关键字参数实例方法
    def keyword_only(self, *, x, y=10):
        return x + y

    # 10. 仅位置参数实例方法
    def positional_only(self, a, b, /, c, d=4):
        return a + b + c + d

    # 11. 带文档字符串的实例方法
    def documented_method(self, x, y):
        """
        这是一个文档字符串
        Args:
            x: 第一个参数
            y: 第二个参数
        Returns:
            两数之和
        """
        return x + y

    # 12. 生成器实例方法
    def generator_method(self):
        for i in range(10):
            yield i

    # 13. 异步实例方法
    async def async_method(self):
        await some_async_operation()

    # 14. 带复杂表达式的实例方法
    def complex_body_method(self, x):
        if x > 0:
            result = x * 2
        else:
            result = x / 2
        return result

    # 15. 带异常处理的实例方法
    def method_with_exception(self):
        try:
            risky_operation()
        except ValueError as e:
            print(f"Value error: {e}")
        except Exception as e:
            print(f"Other error: {e}")
        finally:
            cleanup()

    # 16. 带上下文管理器的实例方法
    def method_with_context(self):
        with open("file.txt") as f:
            return f.read()

    # 17. 带列表推导式的实例方法
    def method_with_comprehension(self, items):
        return [x * 2 for x in items if x > 0]

    # 18. 带条件表达式的实例方法
    def method_with_ternary(self, x):
        return x if x > 0 else -x

    # 19. 带循环的实例方法
    def method_with_loops(self, items):
        result = []
        for item in items:
            if item > 0:
                result.append(item)
        return result

    # 20. 带断言的实例方法
    def method_with_assert(self, value):
        assert value > 0, "Value must be positive"
        return value * 2

    # 21. 带复杂返回语句的实例方法
    def method_with_complex_return(self):
        return {
            "list": [1, 2, 3],
            "dict": {"nested": True},
            "tuple": (1, 2, 3)
        }

    # 22. 带递归的实例方法
    def recursive_method(self, n):
        if n <= 1:
            return 1
        return n * self.recursive_method(n - 1)

    # 23. 带特殊方法名的实例方法
    def __special_method__(self):
        return "special"

    def _private_method(self):
        return "private"

    def method_with_underscore_end_(self):
        return "underscore_end"

    # 24. 带复杂控制流的实例方法
    def method_with_complex_control_flow(self, items):
        for i, item in enumerate(items):
            if i % 2 == 0:
                continue
            if item is None:
                break
            yield item
        else:
            return "completed normally"

# 25. 静态方法
class StaticMethodClass:
    @staticmethod
    def static_method():
        return "static"

    @staticmethod
    def static_method_with_params(a, b=10):
        return a + b

    @staticmethod
    def typed_static_method(x: int, y: str) -> bool:
        return len(y) > x

# 26. 类方法
class ClassMethodClass:
    @classmethod
    def class_method(cls):
        return cls

    @classmethod
    def class_method_with_params(cls, name):
        return f"{cls.__name__}: {name}"

# 27. 属性方法
class PropertyClass:
    def __init__(self):
        self._value = 0

    @property
    def computed_property1(self):
        return self._value

    @computed_property.setter
    def computed_property2(self, value):
        self._value = value * 2

    @computed_property.deleter
    def computed_property3(self):
        del self._value

# 28. 带装饰器的实例方法
class DecoratedMethodClass:
    @my_decorator
    def decorated_method(self):
        pass

    @another_decorator(10)
    def decorated_method_with_args(self):
        pass

    @property
    @cache
    def cached_property(self):
        return expensive_computation()

# 29. 抽象方法
from abc import ABC, abstractmethod

class AbstractClass(ABC):
    @abstractmethod
    def abstract_method(self):
        pass

    @abstractmethod
    def abstract_method_with_params(self, x: int) -> str:
        pass

    def concrete_method(self):
        return "concrete"

# 30. 类继承中的方法重写
class ParentClass:
    def parent_method(self):
        return "parent"

    def overridden_method(self):
        return "parent version"

class ChildClass(ParentClass):
    def overridden_method(self):
        return "child version"

    def extended_method(self):
        parent_result = super().overridden_method()
        return f"child extends {parent_result}"

# 31. 带复杂类型注解的类方法
from typing import List, Dict, Optional, Union, Callable, TypeVar, Generic

T = TypeVar('T')

class TypedClass(Generic[T]):
    def generic_method(self, item: T) -> T:
        return item

    def method_with_typing_annotations(
        self,
        items: List[int],
        mapping: Dict[str, int],
        optional_value: Optional[str] = None,
        union_value: Union[int, str] = 0,
        callback: Callable[[int], str] = None
    ) -> List[str]:
        return [str(item) for item in items]

# 32. 带特殊方法（魔术方法）
class MagicMethodClass:
    def __init__(self, value):
        self.value = value

    def __str__(self):
        return f"MagicMethodClass({self.value})"

    def __repr__(self):
        return f"MagicMethodClass(value={self.value})"

    def __len__(self):
        return len(str(self.value))

    def __getitem__(self, key):
        return str(self.value)[key]

    def __setitem__(self, key, value):
        self.value = str(self.value)[:key] + str(value) + str(self.value)[key+1:]

    def __call__(self, *args, **kwargs):
        return f"Called with {args}, {kwargs}"

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    def __iter__(self):
        return iter(str(self.value))

    def __next__(self):
        # 这里只是示例，实际需要维护迭代状态
        pass

    def __add__(self, other):
        return MagicMethodClass(self.value + other.value)

    def __eq__(self, other):
        return self.value == other.value

# 33. 嵌套类中的方法
class OuterClass:
    class InnerClass:
        def inner_method(self):
            return "inner"

        @staticmethod
        def inner_static_method():
            return "inner static"

    def outer_method(self):
        inner = self.InnerClass()
        return inner.inner_method()

# 34. 带复杂参数解包的实例方法
class UnpackingClass:
    def method_with_unpacking(self):
        def inner(a, b, c):
            return a + b + c
        
        args = [1, 2, 3]
        kwargs = {"a": 1, "b": 2, "c": 3}
        
        return inner(*args) + inner(**kwargs)

# 35. 带 walrus 操作符的实例方法（Python 3.8+）
class WalrusClass:
    def method_with_walrus(self, items):
        results = []
        for item in items:
            if (length := len(item)) > 5:
                results.append(length)
        return results

# 36. 多重继承中的方法
class Base1:
    def method_a(self):
        return "Base1"

class Base2:
    def method_b(self):
        return "Base2"

class MultiInheritanceClass(Base1, Base2):
    def combined_method(self):
        return self.method_a() + self.method_b()

# 37. 带类变量访问的实例方法
class ClassVariableClass:
    class_var = "shared"

    def access_class_var(self):
        return self.class_var

    def modify_class_var(self, new_value):
        ClassVariableClass.class_var = new_value

# 38. 带实例变量的方法
class InstanceVariableClass:
    def __init__(self):
        self.instance_var = "instance"

    def access_instance_var(self):
        return self.instance_var

    def modify_instance_var(self, new_value):
        self.instance_var = new_value

# 39. 带全局和非局部变量的实例方法
global_var = 100

class ScopeClass:
    def method_with_global(self):
        global global_var
        global_var += 1
        return global_var

    def outer_with_nonlocal(self):
        x = 10
        def inner():
            nonlocal x
            x += 1
            return x
        return inner()

# 40. 带复杂的默认值的实例方法
class DefaultValuesClass:
    def method_with_complex_defaults(
        self,
        data=None,
        callback=lambda x: x,
        config={"debug": True, "timeout": 30}
    ):
        if data is None:
            data = []
        return callback(len(data))

# 41. 带 match-case 的实例方法（Python 3.10+）
class MatchClass:
    def method_with_match(self, value)->str:
        match value:
            case 1:
                return "one"
            case 2 | 3:
                return "two or three"
            case x if x > 10:
                return "greater than ten"
            case _:
                return "other"

# 42. 数据类中的方法
from dataclasses import dataclass

@dataclass
class DataClass:
    name: str
    age: int

    def greet(self):
        return f"Hello, I'm {self.name}"

    def is_adult(self) -> bool:
        return self.age >= 18

# 43. 带缓存装饰器的方法
class CacheClass:
    from functools import lru_cache

    @lru_cache(maxsize=128)
    def expensive_method(self, n):
        if n <= 1:
            return 1
        return n * self.expensive_method(n - 1)

# 44. 带多个装饰器的方法
class MultiDecoratorClass:
    @property
    @lru_cache()
    @staticmethod
    def multi_decorated_method():
        return "multi decorated"

# 45. 带复杂控制流的方法
class ControlFlowClass:
    def complex_control_flow(self, items):
        try:
            for i, item in enumerate(items):
                if i % 2 == 0:
                    continue
                if item is None:
                    raise ValueError("None item found")
                yield item.upper()
        except ValueError as e:
            print(f"Error: {e}")
        finally:
            print("Cleanup")

# 46. 带异步生成器的方法
class AsyncGeneratorClass:
    async def async_generator_method(self):
        for i in range(10):
            await asyncio.sleep(0.1)
            yield i

# 47. 带上下文管理器协议的方法
class ContextManagerClass:
    def __init__(self):
        self.resource = None

    def __enter__(self):
        self.resource = "acquired"
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.resource = None

    def method_with_context_manager(self):
        with self as cm:
            return cm.resource

# 48. 带自定义描述符的方法
class DescriptorClass:
    def __init__(self):
        self._value = 0

    class CustomDescriptor:
        def __get__(self, obj, objtype=None):
            return obj._value if obj else None

        def __set__(self, obj, value):
            obj._value = value * 2

        def __delete__(self, obj):
            obj._value = 0

    custom_attr = CustomDescriptor()

    def use_descriptor(self):
        self.custom_attr = 5
        return self.custom_attr

# 49. 带元类的方法
class MetaClass(type):
    def __new__(cls, name, bases, attrs):
        attrs['added_by_meta'] = 'meta'
        return super().__new__(cls, name, bases, attrs)

class MetaClassUser(metaclass=MetaClass):
    def method_using_meta_attr(self):
        return self.added_by_meta

# 50. 各种特殊情况的方法组合
class ComplexClass:
    def __init__(self, *args, **kwargs):
        self.args = args
        self.kwargs = kwargs

    @classmethod
    @property
    def class_property(cls):
        return "class property"

    @staticmethod
    @lru_cache(maxsize=32)
    def cached_static_method(x: int) -> int:
        return x ** 2

    def complex_method(
        self, 
        required_param: str,
        optional_param: int = 10,
        optional_param1: Foo = test.utils.Foo[test.utils.Foo1],
        *args: int,
        keyword_only: bool = True,
        **kwargs: dict
    ) -> Union[str, int, None]:
        """复杂方法示例"""
        try:
            if keyword_only:
                result = len(required_param) + optional_param
                for arg in args:
                    result += arg
                return result
            else:
                return None
        except Exception as e:
            raise RuntimeError(f"Method failed: {e}") from e