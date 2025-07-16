package com.example.test;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.Queue;
import java.util.Stack;
import java.util.LinkedList;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.function.Function;

public class TestClass {


    private int a;
    private int b;
    private int c;
    private int d;
    private int e;
    private int f;
    private int g;
    private int h;
    public int add(int e, int f) {
        return e + f;
    }
    public void test(int a, Function<String, Integer> func, Runnable r, List<String[]> arrs, int[] anums, int... nums) {
        int x = 1;
        double y = 3.14;
        String s = "hello";
        int[] arr = {1, 2, 3};
        List<String> list = new ArrayList<>();
        Map<String, Integer> map = new HashMap<>();
        Set<Double> set = new HashSet<>();
        Runnable localRunnable = new Runnable() {
            @Override
            public void run() {
                System.out.println("Inner Runnable");
            }
        };
        Function<Integer, String> intToString = (Integer i) -> String.valueOf(i);

        list.add("test");
        map.put("key", 100);
        set.add(2.71);

        for (int n : arr) {
            System.out.println(n);
        }

        if (a > 0) {
            System.out.println("a is positive");
        } else {
            System.out.println("a is not positive");
        }

        localRunnable.run();
        String result = intToString.apply(x);
        System.out.println(result);

        return;
    }
    
}