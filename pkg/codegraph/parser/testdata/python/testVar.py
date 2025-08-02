user_dict: Dict[str, User] = {
    "admin": User("Admin", 25),
    "guest": User("Guest", 20)
}




# 直接类名
user: User = User("Alice", 25)
config: Config = Config()
name: str = user.name  # 右值是属性访问

# 可选类型
optional_user: Optional[User] = None
# 列表中的全限定名
items: List[test.utils.Item] = None
items = []

# 字典中的全限定名
mapping: Dict[str, test.models.User] = None
mapping = {}

# 嵌套容器
nested: List[Dict[str, test.api.models.Response]] = None
complex_nested: Dict[
    test.models.Category, 
    List[test.data.models.Item]
] = None


# 简单全限定名
name1: test.utils.Foo = None
name2: test.utils.Foo = test.utils.Foo()

# 全限定名泛型
name3: test.utils.Foo[test.utils.Foo1] = None
name4: test.utils.Foo[test.utils.Foo1] = test.utils.Foo[test.utils.Foo1]()

# 嵌套全限定名
name5: test.utils.Container[test.models.User] = None
name6: test.utils.Container[test.models.User] = test.utils.Container()

# 复杂嵌套全限定名
name7: test.utils.Container[List[test.models.User]] = None
name8: Dict[str, test.api.models.Response[test.data.models.Item]] = None

# 混合限定名
name9: test.utils.Foo[models.User] = None  # 部分限定
name10: utils.Foo[test.models.User] = None  # 部分限定

# 简单泛型
container: Container[str] = Container("hello")
container: Container[int] = Container(42)

# 嵌套泛型
nested_container: Container[List[str]] = Container(["a", "b"])
complex_container: Container[Dict[str, User]] = Container({})

# 多参数泛型
pair: Pair[User, Product] = Pair(User("Alice", 25), Product("Laptop", 999))

# 多层嵌套全限定名
complex_var: test.api.handlers.Processor[
    test.models.requests.UserRequest,
    test.models.responses.UserResponse[test.models.User]
] = None

# 混合标准类型和全限定名
mixed_var: Optional[Dict[
    str, 
    List[test.data.models.Item[test.config.Settings]]
]] = None

# 多个全限定名的组合
multi_qualified: Union[
    test.auth.models.User,
    test.api.models.Admin,
    test.data.models.Guest
] = None

