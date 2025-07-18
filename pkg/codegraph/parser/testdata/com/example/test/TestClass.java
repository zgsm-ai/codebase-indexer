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
}
