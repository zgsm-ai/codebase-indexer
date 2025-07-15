// ============== 一、模块导入高级特性 ==============
// 1. 命名导入与重命名
import { format, parse } from "./utils"; // 导入具名导出
import { log as logger } from "./logger"; // 重命名导入

// 2. 默认导入
import httpClient from "./api/client"; // 导入默认导出

// 3. 导入整个模块
import * as fs from "fs"; // 导入Node.js模块

// 4. 动态导入（ES模块）
async function loadPlugin(name: string) {
    const plugin = await import(`./plugins/${name}`);
    return plugin.init();
}

// 5. 导入类型（仅用于类型注解）
import type { User, Role } from "./models"; // 类型导入

// 6. CommonJS模块导入
const moment = require("moment"); // CommonJS风格导入
import dayjs from "dayjs"; // ES模块风格导入


// ============== 二、类型系统高级特性 ==============
// 1. 条件类型进阶
type NonNullable<T> = T extends null | undefined ? never : T;
type DeepReadonly<T> = {
    readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

// 2. 映射类型与keyof
type PickByType<T, U> = {
    [P in keyof T as T[P] extends U ? P : never]: T[P];
};
type StringFields = PickByType<User, string>;

// 3. infer关键字（类型推断）
type ExtractPromise<T> = T extends Promise<infer U> ? U : T;
type Data = ExtractPromise<Promise<User[]>>; // User[]

// 4. 元组类型操作
type First<T extends unknown[]> = T[0];
type Last<T extends unknown[]> = T[T extends unknown[] ? T["length"] - 1 : never];
type UserTuple = [id: number, name: string, roles: string[]];
type FirstUserRole = Last<UserTuple>; // string[]

// 5. 混合类型（对象+函数）
interface Counter {
    (start: number): number;
    interval: number;
    reset(): void;
}

function createCounter(interval: number): Counter {
    const counter: Counter = function(start: number) {
        return start;
    };
    counter.interval = interval;
    counter.reset = function() {};
    return counter;
}

// 6. 模板字面量类型（TS 4.1+）
type Email = ` ${string}@${string}.${string} `;
type Color = `#${string}`;

// 7. 递归类型
type TreeNode = {
    name: string;
    children?: TreeNode[];
};


// ============== 三、泛型高级特性 ==============
// 1. 泛型类
class Queue<T> {
    private items: T[] = [];

    enqueue(item: T): void {
        this.items.push(item);
    }

    dequeue(): T | undefined {
        return this.items.shift();
    }

    peek(): T | undefined {
        return this.items[0];
    }

    isEmpty(): boolean {
        return this.items.length === 0;
    }
}

// 2. 泛型约束
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
    return obj[key];
}

// 3. 泛型接口
interface KeyValuePair<K, V> {
    key: K;
    value: V;
}

// 4. 泛型条件类型
type TypeName<T> = T extends string
    ? "string"
    : T extends number
        ? "number"
        : T extends boolean
            ? "boolean"
            : T extends undefined
                ? "undefined"
                : T extends Function
                    ? "function"
                    : "object";


// ============== 四、类与接口高级特性 ==============
// 1. 抽象类
abstract class DataService {
    protected data: any[] = [];

    abstract fetch(): Promise<void>;

    process(data: any[]): void {
        this.data = data;
    }

    get(): any[] {
        return this.data;
    }
}

// 2. 接口继承
interface Person {
    name: string;
    age: number;
}

interface Employee extends Person {
    id: string;
    department: string;
}

// 3. 类实现多个接口
interface Drawable {
    draw(): void;
}

interface Resizable {
    resize(width: number, height: number): void;
}

class UIElement implements Drawable, Resizable {
    draw(): void {
        console.log("Drawing UI element");
    }

    resize(width: number, height: number): void {
        console.log(`Resizing to ${width}x${height}`);
    }
}

// 4. 静态属性与方法
class MathUtils {
    static readonly PI = 3.14159;

    static add(a: number, b: number): number {
        return a + b;
    }

    static multiply(a: number, b: number): number {
        return a * b;
    }
}


// ============== 五、装饰器高级应用 ==============
// 1. 类装饰器
function LogClass(target: Function) {
    console.log(`Class decorated: ${target.name}`);
    const originalConstructor = target;

    const decoratedConstructor: any = function(...args: any[]) {
        console.log("Creating instance with args:", args);
        const instance = new originalConstructor(...args);
        return instance;
    };

    decoratedConstructor.prototype = originalConstructor.prototype;
    return decoratedConstructor;
}

// 2. 属性装饰器
function LogProperty(target: any, propertyKey: string) {
    let value: any;

    const getter = () => value;
    const setter = (newVal: any) => {
        console.log(`Setting ${propertyKey} to ${newVal}`);
        value = newVal;
    };

    if (delete target[propertyKey]) {
        Object.defineProperty(target, propertyKey, {
            get: getter,
            set: setter,
            enumerable: true,
            configurable: true
        });
    }
}

// 3. 参数装饰器
function LogParameter(target: any, methodKey: string, parameterIndex: number) {
    console.log(`Parameter ${parameterIndex} in ${methodKey} is decorated`);
}

// 使用装饰器
@LogClass
class UserService {
    @LogProperty
    private apiKey: string;

    constructor(apiKey: string) {
        this.apiKey = apiKey;
    }

    fetchUsers(@LogParameter page: number, @LogParameter limit: number) {
        console.log(`Fetching users: page ${page}, limit ${limit}`);
    }
}


// ============== 六、异步编程与生成器 ==============
// 1. 异步函数
async function fetchUserData() {
    try {
        const response = await httpClient.get("/api/users");
        return response.data as User[];
    } catch (error) {
        console.error("Error fetching users:", error);
        throw error;
    }
}

// 2. 生成器函数
function* iterateUsers(users: User[]) {
    for (const user of users) {
        yield user;
    }
}

// 3. 可迭代协议
class UserIterator implements Iterable<User> {
    private users: User[];
    private index: number = 0;

    constructor(users: User[]) {
        this.users = users;
    }

    [Symbol.iterator](): Iterator<User> {
        return this;
    }

    next(): IteratorResult<User> {
        if (this.index < this.users.length) {
            return {
                value: this.users[this.index++],
                done: false
            };
        }
        return { value: undefined, done: true };
    }
}


// ============== 七、Node.js特定语法 ==============
// 1. CommonJS模块导出
module.exports = {
    UserManager,
    createUser,
    Status
};

// 2. require动态导入
function loadConfig(env: string) {
    return require(`./config/${env}.json`);
}

// 3. Node.js类型声明
import { ReadStream, WriteStream } from "fs";
import { Server, IncomingMessage, ServerResponse } from "http";


// ============== 八、实用工具类型 ==============
// 1. 内置工具类型
type UserPartial = Partial<User>; // 所有属性可选
type UserRequired = Required<User>; // 所有属性必选
type UserPick = Pick<User, "id" | "name">; // 选择特定属性
type UserOmit = Omit<User, "createdAt">; // 省略特定属性
type UserExclude = Exclude<Status, Status.Active>; // 排除类型
type UserExtract = Extract<Status, Status.Active | Status.Pending>; // 提取类型

// 2. 自定义工具类型
type Merge<T, U> = Omit<T, keyof U> & U;
type UserWithAudit = Merge<User, { createdBy: string; updatedAt: Date }>;


// ============== 九、主程序应用示例 ==============
// 1. 使用泛型类
const userQueue = new Queue<User>();
userQueue.enqueue(admin);
userQueue.enqueue(createUser({ name: "Guest" }));
const firstUser = userQueue.dequeue();

// 2. 使用装饰器
const service = new UserService("api-key-123");
service.fetchUsers(1, 10);

// 3. 使用生成器
const users = [admin, createUser({ name: "User1" })];
const userGenerator = iterateUsers(users);
for (const user of userGenerator) {
    console.log(user.name);
}

// 4. 异步函数调用
fetchUserData()
    .then(users => console.log("Fetched users:", users.length))
    .catch(error => console.error("Error:", error));

// 5. 类型守卫应用
function processActiveUsers(users: User[]): void {
    const activeUsers = users.filter(isActive);
    activeUsers.forEach(user => {
        console.log(`${user.name} is active`);
    });
}

// 原代码中的定义
enum Status {
    Active = "ACTIVE",
    Inactive = "INACTIVE",
    Pending = "PENDING"
}

interface User {
    id: number;
    name: string;
    age: number;
    status: Status;
    createdAt: Date;
}

type ReadonlyUser = {
    readonly [P in keyof User]: User[P];
};

type PartialUser = {
    [P in keyof User]?: User[P];
};

function createUser(user: PartialUser): User {
    return {
        id: Math.random(),
        name: user.name || "Anonymous",
        age: user.age || 18,
        status: user.status || Status.Pending,
        createdAt: new Date()
    };
}

class UserManager {
    private users: User[] = [];

    addUser(user: User): void {
        this.users.push(user);
    }

    getUserById(id: number): User | undefined {
        return this.users.find(u => u.id === id);
    }

    updateUser(id: number, changes: PartialUser): void {
        const index = this.users.findIndex(u => u.id === id);
        if (index !== -1) {
            this.users[index] = { ...this.users[index], ...changes };
        }
    }

    getStatusCounts(): Record<Status, number> {
        return Object.values(Status).reduce((acc, status) => {
            acc[status] = this.users.filter(u => u.status === status).length;
            return acc;
        }, {} as Record<Status, number>);
    }
}

function isActive(user: User): user is User & { status: Status.Active } {
    return user.status === Status.Active;
}

const admin: User = {
    ...createUser({ name: "Admin", status: Status.Active }),
    id: 1,
    age: 30,
    createdAt: new Date()
};