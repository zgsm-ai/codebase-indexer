// 导入模块
import defaultName from 'modules.js';
import { test } from 'modules';
import { export as ex1 } from 'modules';
import { export1, export2 } from 'modules.js';
import { export1 as ex1, export2 as ex2 } from 'moduls.js';
import defaultName, { expoprt } from 'modules';
import * as moduleName from 'modules.js';
import defaultName, * as moduleName from 'modules';
import 'modules';
// 变量声明
var globalVar = 'global';
let blockVar = 'block';
const constant = 42;

// 数据类型
const num = 123;
const str = "Hello";
const bool = true;
const nullVal = null;
const undefinedVal = undefined;
const sym = Symbol('key');

// 对象
const person = {
    name: 'Alice',
    age: 30,
    greet() { console.log(`Hi, I'm ${this.name}`); }
};

// 数组
const arr = [1, 'two', true];
arr.push(4);

// 函数
function add(a, b) {
    return a + b;
}

// 箭头函数
const multiply = (a, b) => a * b;

// 解构赋值
const { name } = person;
const [first] = arr;

// 展开语法
const clone = { ...person };
const combined = [...arr, 5, 6];

// 条件语句
if (num > 100) {
    console.log('Greater than 100');
} else {
    console.log('Less than or equal to 100');
}

// 循环
for (let i = 0; i < 3; i++) {
    console.log(i);
}

arr.forEach(item => console.log(item));

// Promise
const fetchData = () =>
    new Promise(resolve => setTimeout(() => resolve('Data'), 1000));

// Async/Await
async function getData() {
    const data = await fetchData();
    console.log(data);
}

// 类
class Animal {
    constructor(name) {
        this.name = name;
    }
    speak() {
        console.log(`${this.name} makes a sound`);
    }
}

class Dog extends Animal {
    speak() {
        super.speak();
        console.log('Woof!');
    }
}

// DOM 操作 (模拟)
document.addEventListener('DOMContentLoaded', () => {
    const element = document.createElement('div');
    element.textContent = 'Hello World';
    document.body.appendChild(element);
});

// 错误处理
try {
    throw new Error('Something went wrong');
} catch (error) {
    console.error(error.message);
} finally {
    console.log('Cleanup');
}

// 事件监听
window.addEventListener('click', () => console.log('Clicked'));

// 闭包
function outer() {
    const x = 10;
    return () => console.log(x);
}

// 立即执行函数
(() => console.log('IIFE executed'))();