with database_transaction("users_table") as transaction:
    # transaction是自定义类型
    pass
# 解析函数返回自定义类型
json_obj = json.loads(data, object_hook=CustomObject.from_dict)
xml_doc = XMLParser.parse(xml_string)
config = YAMLConfig.load("config.yaml")
# ORM查询返回自定义模型类型
user = User.objects.get(id=1)
queryset = User.objects.filter(age__gte=18)
session = DatabaseSession.create(engine_url)

# 配置构建器创建自定义配置类型
config = AppConfig.Builder().set_debug(True).set_database_url("...").build()
settings = SettingsLoader.load_from_file("settings.ini")
options = CommandLineParser.parse_args(sys.argv[1:])
processor = DataProcessor([a(), 2, 3]).filter().transform()

# 泛型类型的实例化
int_list = List[int]()
str_dict = Dict[str, int]()
optional_value = Optional[str]()