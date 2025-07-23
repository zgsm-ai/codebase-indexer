package com.example.test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

// 定义接口
interface Printable {
    void print();
}

interface Savable {
    boolean save(String destination);
}

// 顶层默认访问类（包访问），实现接口
class ReportGenerator implements Printable, Savable {
    private int reportId;
    protected String title;
    public static final String VERSION = "1.0";
    int a,b,c;

    public ReportGenerator(int id, String title) {
        this.reportId = id;
        this.title = title;
    }

    @Override
    public void print() {
        System.out.println("Printing report: " + title);
    }

    @Override
    public boolean save(String destination) {
        System.out.println("Saving report to: " + destination);
        return true;
    }

    public class ReportDetails {
        boolean verified = false;

        public void verify() {
            verified = true;
        }

        private class InternalReview {
            char level = 'B';
        }
    }

    static class ReportMetadata {
        long createdAt;

        static void describe() {
            System.out.println("Static metadata for report");
        }
    }
}

// 父类
class User {
    protected String username;
    public int age;

    public void login() {
        System.out.println(username + " logged in.");
    }
}

// 顶层 public 类，继承父类，继承+实现接口
public class FinancialReport extends User implements Printable, Savable {
    public List<String> authors;
    protected Map<String, Double> monthlyRevenue;
    private final ReportGenerator generator = new ReportGenerator(1001, "Annual Report");

    List<? extends Number> statistics;
    ReportGenerator[] reports;

    public FinancialReport() {
        authors = new ArrayList<>();
        monthlyRevenue = new HashMap<>();
    }

    public static void main(String[] args) {
        System.out.println("Generating Financial Report...");
    }

    @Override
    public void print() {
        System.out.println("Financial report printed.");
    }

    @Override
    public boolean save(String path) {
        System.out.println("Financial report saved to " + path);
        return true;
    }

    private void prepareData() {}

    protected static final int calculateProfit(int revenue, int cost) {
        return revenue - cost;
    }

    // 返回值为复杂泛型，参数也包含复杂泛型和自定义类型
    public Map<String, List<ReportGenerator>> getReportMap(List<? extends User> users, Map<String, List<Double>> revenueMap, int year) {
        Map<String, List<ReportGenerator>> result = new HashMap<>();
        for (User user : users) {
            List<ReportGenerator> reports = new ArrayList<>();
            if (revenueMap.containsKey(user.username)) {
                for (Double revenue : revenueMap.get(user.username)) {
                    reports.add(new ReportGenerator(year, "Report for " + user.username + " revenue: " + revenue));
                }
            }
            result.put(user.username, reports);
        }
        return result;
    }

    // 无参数，返回自定义泛型类型
    public List<FinancialReport> getAllReports() {
        List<FinancialReport> reports = new ArrayList<>();
        reports.add(this);
        return reports;
    }

    // 参数为基础类型、泛型、自定义类，返回复杂嵌套泛型
    public Map<Integer, List<Map<String, ReportGenerator>>> buildComplexStructure(
            int count,
            List<String> names,
            ReportGenerator generator
    ) {
        Map<Integer, List<Map<String, ReportGenerator>>> complexMap = new HashMap<>();
        for (int i = 0; i < count; i++) {
            List<Map<String, ReportGenerator>> list = new ArrayList<>();
            for (String name : names) {
                Map<String, ReportGenerator> innerMap = new HashMap<>();
                innerMap.put(name, generator);
                list.add(innerMap);
            }
            complexMap.put(i, list);
        }
        return complexMap;
    }

    // 返回通配符泛型，参数为自定义类数组
    public List<? super User> processUsers(User[] userArray) {
        List<User> userList = new ArrayList<>();
        for (User u : userArray) {
            userList.add(u);
        }
        return new ArrayList<User>(userList);
    }

    // 返回数组类型，参数为自定义类数组
    public User[] getUserArray(int size) {
        return new User[size];
    }

    // 返回自定义泛型类型，参数为自定义类数组
    public List<User> getUserList(User[] users) {
        return new ArrayList<User>(Arrays.asList(users));
    }
    
    // 返回二维数组，参数为二维数组
    public int[][] transformMatrix(int[][] matrix) {
        int n = matrix.length;
        int[][] result = new int[n][];
        for (int i = 0; i < n; i++) {
            int m = matrix[i].length;
            result[i] = new int[m];
            for (int j = 0; j < m; j++) {
                result[i][j] = matrix[i][j] * 2;
            }
        }
        return result;
    }

    // 泛型方法，参数和返回值均为泛型
    public <T extends Number> List<T> filterList(List<T> list, T threshold) {
        List<T> result = new ArrayList<>();
        for (T item : list) {
            if (item.doubleValue() > threshold.doubleValue()) {
                result.add(item);
            }
        }
        return result;
    }

    // 嵌套泛型和可变参数
    public Map<String, List<User>> groupUsersByPrefix(String prefix, User... users) {
        Map<String, List<User>> map = new HashMap<>();
        for (User user : users) {
            if (user.username.startsWith(prefix)) {
                map.computeIfAbsent(prefix, k -> new ArrayList<>()).add(user);
            }
        }
        return map;
    }

    // 方法内部定义匿名类
    public Runnable createTask(String message) {
        return new Runnable() {
            @Override
            public void run() {
                System.out.println("任务消息: " + message);
            }
        };
    }

    // 递归方法
    public int factorial(int n) {
        if (n <= 1) return 1;
        return n * factorial(n - 1);
    }

    // 抛出异常的方法
    public void checkReportId(int id) throws IllegalArgumentException {
        if (id < 0) {
            throw new IllegalArgumentException("报告ID不能为负数");
        }
    }

    // 使用通配符泛型参数
    public void printAllReports(List<? extends ReportGenerator> reports) {
        for (ReportGenerator report : reports) {
            report.print();
        }
    }

    // 返回Optional类型
    public Optional<User> findUserByName(List<User> users, String name) {
        for (User user : users) {
            if (user.username.equals(name)) {
                return Optional.of(user);
            }
        }
        return Optional.empty();
    }

    
}
    

