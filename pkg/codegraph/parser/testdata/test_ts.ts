// ========== 基础类型 ==========
var a = 2;
let isDone: boolean = false;
const count: number = 42;
let userName: string = "Alice";
let items: number[] = [1, 2, 3];
let tuple: [string, number] = ["hello", 10];
let anyValue: any = "could be anything";
const { a, b }: { a: string; b: number } = user;
const fetchData = (a: string, b: number): Promise<string> =>
  new Promise(resolve => setTimeout(() => resolve("Data"), 1000));

// ========== 枚举 ==========
enum Direction {
  Up = 1,
  Down,
  Left,
  Right,
}

// ========== 接口 ==========
interface Person {
  id: number;
  name: string;
  age?: number; // 可选属性
  greet(): void;
}

// ========== 类型别名 ==========
type ID = string | number;
type ReadOnlyPerson = Readonly<Person>;

// ========== 类 ==========
class Animal implements Person {
  public name: string;
  private age: number;
  protected species: string;

  constructor(name: string, age: number, species: string) {
      this.name = name;
      this.age = age;
      this.species = species;
  }

  private getAge() {
      return this.age;
  }
}


// ========== 泛型 ==========
function identity<T>(arg: T): T {
  return arg;
}

let output = identity<string>("myString");

class Box<T> {
  contents: T;
  constructor(value: T) {
    this.contents = value;
  }
}

// ========== 函数 ==========
function add(x: number, y: number = 10): number {
  return x + y;
}

const subtract = (a: number, b: number): number => a - b;

// ========== 类型断言 ==========
let someValue: any = "this is a string";
let strLength: number = (someValue as string).length;

// ========== 联合类型 & 交叉类型 ==========
type Action = "start" | "stop" | "pause";
type Shape = { color: string } & { radius: number };

// ========== 条件类型 ==========
type IsString<T> = T extends string ? true : false;

// ========== 映射类型 ==========
type PartialPerson = {
  [K in keyof Person]?: Person[K];
};

// ========== 装饰器 ==========
function Log(target: any, key: string) {
  console.log(`Called: ${key}`);
}

class Calculator {
  @Log
  add(a: number, b: number): number {
    return a + b;
  }
}

// ========== 异步函数 ==========
async function fetchData(url: string): Promise<string> {
  const response = await fetch(url);
  return await response.text();
}

// ========== 模块系统 ==========
import fs from "fs";
export const version = "1.0.0";

// ========== 命名空间 ==========
namespace Geometry {
  export function area(radius: number): number {
    return Math.PI * radius * radius;
  }
}
