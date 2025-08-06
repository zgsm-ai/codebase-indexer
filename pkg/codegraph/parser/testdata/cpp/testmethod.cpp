#include <iostream>
#include <string>
#include <map>
#include <vector>
#include <list>

// 用户自定义类型1
struct Address {
public:
    std::string city;
    int zipCode;

    Address() : city("Unknown"), zipCode(0) {}
    Address(const std::string& c, int z) : city(c), zipCode(z) {}
};

// 用户自定义类型2
class Job {
private:
    std::string title;
    double salary;

protected:
    void setSalary(double s) { salary = s; }

public:
    Job() : title("None"), salary(0) {}
    Job(const std::string& t, double s) : title(t), salary(s) {}

    std::string getTitle() const { return title; }
    double getSalary() const { return salary; }
};

// 定义结构体
struct PersonStruct {
public:
    std::string name;
    int age;

    // 返回map嵌套vector<UserDefinedType>
    std::map<std::string, std::vector<Address>> getAddressMap() const {
        return {{"Home", {Address("New York", 10001), Address("Boston", 2100)}}};
    }

protected:
    // 返回list嵌套map<string, UserDefinedType>
    std::list<std::map<std::string, Job>> getJobList() const {
        return {{{"Developer", Job("Developer", 80000)}, {"Manager", Job("Manager", 95000)}}};
    }

private:
    // 返回vector嵌套list<int>
    std::vector<std::list<int>> getNestedInts() const {
        return {{1,2,3}, {4,5,6}};
    }

public:
    // 无参数
    void sayHello() const {
        std::cout << "Hello from PersonStruct!" << std::endl;
    }

    // 一个基础类型参数，带默认值
    void setAge(int newAge = 30) {
        age = newAge;
    }

    // 两个参数，一个基础类型，一个用户自定义类型
    void setNameAndAddress(const std::string& newName, const Address& addr) {
        name = newName;
        std::cout << "Lives in " << addr.city << std::endl;
    }

    // 三个参数，包含vector和基础类型，带默认值
    void updateInfo(const std::vector<int>& scores, int rank = 1, double bonus = 0.0) {
        std::cout << "Rank: " << rank << ", Bonus: " << bonus << std::endl;
        std::cout << "Scores count: " << scores.size() << std::endl;
    }
};

// 定义类
class PersonClass {
public:
    std::string name;
    double height;

    // 返回vector嵌套list<UserDefinedType>
    std::vector<std::list<Address>> getAddresses() const {
        return {{{Address("LA", 90001), Address("SF", 94101)}}};
    }

protected:
    // 返回map嵌套map<string, UserDefinedType>
    std::map<std::string, std::map<std::string, Job>> getJobMap() {
        return {
                {"IT", { {"Dev", Job("Dev", 70000)}, {"QA", Job("QA", 65000)} }},
                {"HR", { {"Recruiter", Job("Recruiter", 60000)} }}
        };
    }


private:
    // 返回list嵌套vector<double>
    std::list<std::vector<double>> getNestedDoubles() const {
        return {{3.14, 2.71}, {1.41, 1.73}};
    }

public:
    // 无参数
    void greet() const {
        std::cout << "Hello from PersonClass!" << std::endl;
    }

    // 一个基础类型参数，带默认值
    void setHeight(double newHeight = 170.5) {
        height = newHeight;
    }

    // 两个参数，一个用户自定义类型，一个基础类型，带默认值
    void setJobAndAge(const Job& job, int age = 25) {
        std::cout << "Job: " << job.getTitle() << ", Age: " << age << std::endl;
    }

    // 三个参数，包含list和基础类型
    void updateStats(const std::list<int>& scores, int rank, double factor) {
        std::cout << "Rank: " << rank << ", Factor: " << factor << std::endl;
        std::cout << "Scores count: " << scores.size() << std::endl;
    }
};

int main() {
    PersonStruct ps;
    ps.sayHello();
    ps.setAge();
    ps.setNameAndAddress("Alice", Address("Seattle", 98101));
    ps.updateInfo({90, 85, 88});

    PersonClass pc;
    pc.greet();
    pc.setHeight();
    pc.setJobAndAge(Job("Engineer", 85000));
    pc.updateStats({100, 95, 80}, 2, 1.5);

    return 0;
}
