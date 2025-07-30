# 简单类声明
class Person:
    pass

class Animal:
    pass

class Car:
    pass
# 单继承
class Dog(Animal):
    pass

class Cat(Animal):
    pass

class Manager(Employee):
    pass

class Rectangle(Shape):
    pass

# 多继承
class FlyingCar(Car, Aircraft):
    pass

class StudentTeacher(Student, Teacher):
    pass

class WalkerSwimmer(Walker, Swimmer):
    pass

# 带关键字参数的继承
class Database(metaclass=SingletonMeta):
    pass

class Model2(BaseModel, metaclass=ModelMeta):
    pass

# 多个关键字参数
class APIRouter(BaseRouter, metaclass=RouterMeta, prefix="/api"):
    pass

# 仅关键字参数
class Config(metaclass=ConfigMeta):
    pass
# 使用metaclass参数
class User2(metaclass=UserMeta):
    pass

class Product2(metaclass=ModelMeta):
    pass

# 继承并使用元类
class Order2(BaseModel, metaclass=OrderMeta):
    pass

class Payment(BaseModel, metaclass=PaymentMeta):
    pass
from typing import TypeVar, Generic

T = TypeVar('T')
K = TypeVar('K')
V = TypeVar('V')

# 泛型类
class Container(Generic[T]):
    def __init__(self, value: T):
        self.value = value

class Repository(Generic[T]):
    def save(self, item: T) -> None:
        pass

class Map(Generic[K, V]):
    def get(self, key: K) -> V:
        pass

# 继承泛型类
class UserContainer(Container[User]):
    pass

class ProductRepository(Repository[Product]):
    pass

from typing import List, Dict, Union, Optional, Generic, TypeVar

T = TypeVar('T')
K = TypeVar('K')
V = TypeVar('V')

# 继承包含List的类型
class UserList(List[User]):
    pass

class ProductList(List[Product]):
    pass

# 继承包含Dict的类型
class UserDict(Dict[str, User]):
    pass

class ConfigDict(Dict[str, Union[str, int, bool]]):
    pass

# 继承包含Union的类型
class FlexibleContainer(Union[List[int], List[str]],BaseClass,metaclass=PaymentMeta,prefix="/api"):
    pass

class NumberOrString(Union[int, float, str]):
    pass
# 多个复杂类型继承
class ComplexClass(List[User], Dict[str, Product], metaclass=MetaClass):
    pass

class DataProcessor(List[Dict[str, Union[int, str]]], 
                   Optional[Cache], 
                   Logger):
    pass

# 包含关键字参数的复杂继承
class AdvancedManager(List[User], 
                     Dict[str, Permission], 
                     metaclass=ManagerMeta,
                     thread_safe=True):
    pass
# 使用泛型作为 metaclass
class User(metaclass=Dict[str, User]):
    pass

class Product(metaclass=List[Product]):
    pass

class Order(metaclass=Union[TypeA, TypeB]):
    pass
class Model(metaclass=django.db.models.Model):
    pass
class Atest(metaclass=mylib.utils.Foo[mylib.utils.Foo1]):
	pass
class Btest(mylib.utils.Foo[mylib.utils.Foo1],User):
	pass
class Ctest(mylib.utils.Foo,User):
	pass
