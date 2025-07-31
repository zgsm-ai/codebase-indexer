// 调用函数
const greeting = greet("Alice");         // Hello, Alice!
const sum = add(3, 5);                   // 8
const joinedStr = join("A", "B");        // "AB"
const joinedNum = join(1, 2);            // 3

// 类方法调用
const calc = new Calculator(2);
const result1 = calc.multiply(6);         // 12
const result2 = calc.divide(10);          // 5

// 静态方法调用
const desc = Calculator.description();    // "This is a calculator."
// -------------------------------------------------------------------------------------------------
// 方法签名接口
// interface Logger {
//   log(message: string): void;
//   error?(msg: string): void;  // 可选方法
// }

// class Animal {
//   myPerson: Person
//   Dog: Animal
//   // 普通方法
//   speak(sound: string): string {
//     return `Animal says: ${sound}`;
//   }

//   // 静态方法
//   static category(): string {
//     return "Mammal";
//   }

//   // 可选方法
//   move?(distance: number): void;

//   // 访问器：getter 和 setter
//   private _age = 0;

//   get age(): number {
//     return this._age;
//   }

//   set age(val: number) {
//     if (val >= 0) this._age = val;
//   }

//   // 异步方法
//   async fetchFood(): Promise<string> {
//     return new Promise(resolve => setTimeout(() => resolve("🍖 Food fetched"), 1000));
//   }

//   // 方法重载
//   describe(name: string): string;
//   describe(name: string, age: number): string;
//   describe(name: string, age?: number): string {
//     return age ? `${name} is ${age} years old` : `${name} is a mysterious being`;
//   }

//   // 泛型方法
//   echo<T>(input: T): T {
//     return input;
//   }
// }

// // 实现接口
// class ConsoleLogger implements Logger {
//   log(message: string): void {
//     console.log("LOG:", message);
//   }

//   error(msg: string): void {
//     console.error("ERROR:", msg);
//   }
// }

//------------------------------------------------------------------------------------------------------------------

// // 1. 命名函数（具名函数）
// function add(a: number, b: number): number {
//     return a + b;
//   }
  
//   // 2. 匿名函数（赋值给变量）
//   const subtract = function(a: number, b: number): number {
//     return a - b;
//   };
  
//   // 3. 箭头函数（常用于回调或函数变量）
//   const multiply = (a: number, b: number): number => a * b;
  
//   // 4. 函数类型注解
//   let divide: (a: number, b: number) => number;
//   divide = (a, b) => a / b;
  
//   // 5. 可选参数 & 默认参数
//   function greet(name?: string, greeting: string = "Hello") {
//     return `${greeting}, ${name ?? "stranger"}!`;
//   }
  
//   // 6. 剩余参数（rest parameters）
//   function sumAll(...nums: number[]): number {
  
//   }
  
//   // 7. 函数重载
//   function reverse(x: string | number): string | number {
  
//   }
  
//   // 8. 泛型函数
//   function identity<T>(value: T): T {
//     return value;
//   }