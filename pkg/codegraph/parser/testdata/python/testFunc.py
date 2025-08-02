def hello():
    pass

def greet(name):
    return "Hello, " + name

def add(a, b):
    return a + b
def greet(name="world"):
    print(f"Hello, {name}")

def connect(host,/, port=8080, timeout=30):
    pass

def greet1(name: str) -> str:
    return "Hello, " + name

def process(items: list[dict[str, int]],items5: dict[str, int],items6: str) -> None:
    pass
def log(*args):
    print(args)

def config(**kwargs):
    for k, v in kwargs.items():
        print(f"{k}={v}")

def func(a, *args, **kwargs):
    pass
def great_test(a,b:str,c,d:int)->int:
    pass

def add_status(items: list[dict[str, int]]) -> list[dict[str, int]]:
    return [
        {**item, "status": 1 if item["value"] > 5 else 0}
        for item in items
    ]
    
def f_test(a) -> test.utils.Foo[test.utils.Foo1]:
    ...