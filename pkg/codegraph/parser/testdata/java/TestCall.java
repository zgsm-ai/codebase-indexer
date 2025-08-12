package com.example.test;

public class Test {
    public static void main(String[] args) {


        // class_literal
        Class<String> stringClass = String.class;
		Class<java.util.List> clazz2 = java.util.List.class;
		Class<String[]> clazz3 = String[].class;
		Class<java.util.Map.Entry> clazz4 = java.util.Map.Entry.class;
		Class<Map.Entry<String, Integer>> clazz4b = Map.Entry.class;
		Class<int[][]> clazz5 = int[][].class;
        Class<String[][]> clazz5b = String[][].class;
		Class<?> clazz2D = my.custom.MyType[][].class;
		Class<?> clazz6 = java.util.Map.Entry[][].class;

        //类型转换的情况
        // ---- 非自定义类型初始化 ----
        
        // 1. 初始化时强制转换 - 变量类型和强制转换类型一致
        int intVar1 = (int) 100;
        double doubleVar1 = (double) 3.14;
        float floatVar1 = (float) 2.5f;
        
        // 2. 初始化时强制转换 - 变量类型和强制转换类型不一致
        long longVar1 = (long) 100;  // int转long
        double doubleVar2 = (double) 5;  // int转double
        byte byteVar1 = (byte) 256;  // int转byte
        
        // 3. 带包名的类型转换
        java.lang.Integer integerVar1 = (java.lang.Integer) 42;
        java.util.ArrayList<String> listVar1 = (java.util.ArrayList<String>) new java.util.ArrayList<String>();
        java.lang.Object objVar1 = (java.lang.String) "Hello";
        
        // ---- 自定义类型初始化 ----
        
        // 4. 自定义类型 - 变量类型和强制转换类型一致
        Parent parent1 = (Parent) new Parent();
        Child child1 = (Child) new Child();
        List<String> names = new ArrayList<String>();
        java.util.Map<String, Integer> map = new java.util.HashMap<String, Integer>();
        Map<String, List<Integer>> nested = new HashMap<String, List<Integer>>();

        
        // 5. 自定义类型 - 变量类型和强制转换类型不一致
        Parent parent2 = (Parent) new Child();  // 向上转换
        Object obj1 = (Object) new Parent();    // 向上转换
        
        // ==================== 已声明变量赋值时的类型转换 ====================
        
        // ---- 非自定义类型赋值 ----
        
        // 6. 赋值时强制转换 - 变量类型和强制转换类型一致
        int intVar3;
        intVar3 = (int) 200;
        
        double doubleVar3;
        doubleVar3 = (double) 4.14;
        
        // 7. 赋值时强制转换 - 变量类型和强制转换类型不一致
        long longVar2;
        longVar2 = (long) 150;  // int转long
        
        float floatVar2;
        floatVar2 = (float) 3.14159;  // double转float
        
        // 8. 带包名的类型赋值转换
        java.lang.Integer integerVar2;
        integerVar2 = (java.lang.Integer) 84;
        
        java.lang.Object objVar2;
        objVar2 = (java.lang.String) "World";
        
        // ---- 自定义类型赋值 ----
        
        // 9. 自定义类型赋值 - 类型一致
        Parent parent3;
        parent3 = (Parent) new Parent();
        
        Child child2;
        child2 = (Child) new Child();
        
        // 10. 自定义类型赋值 - 类型不一致
        Parent parent4;
        parent4 = (Parent) new Child();  // 向上转换
        
        Object obj2;
        obj2 = (Object) new Child();     // 向上转换
        
        System.out.println("=== Instanceof Expression Examples ===");
        
        // ==================== Instanceof Expression 各种情况 ====================
        
        // ---- 非自定义类型 instanceof ----
        String str = "Hello";
        Integer integerObj = 42;
        Object obj3 = new Object();
        
        // 11. 基本对象 instanceof
        boolean result1 = str instanceof String;
        boolean result2 = integerObj instanceof Integer;
        boolean result3 = obj3 instanceof Object;
        
        // 12. 带包名的类型 instanceof
        boolean result4 = str instanceof java.lang.String;
        boolean result5 = integerObj instanceof java.lang.Integer;
        java.util.List<String> list = new java.util.ArrayList<>();
        boolean result6 = list instanceof java.util.ArrayList;
        
        // ---- 自定义类型 instanceof ----
        Parent parent5 = new Parent();
        Child child3 = new Child();
        Parent parent6 = new Child();
        
        // 13. 自定义类型 instanceof
        boolean result7 = parent5 instanceof Parent;
        boolean result8 = child3 instanceof Child;
        boolean result9 = parent6 instanceof Child;
        boolean result10 = parent5 instanceof Object;
        
        // 14. instanceof 与 null
        Parent nullParent = null;
        boolean result11 = nullParent instanceof Parent;  // false
        
        // 15. instanceof 用于类型检查后进行转换
        Object unknownObj = new Child();
        if (unknownObj instanceof Child) {
            Child castedChild = (Child) unknownObj;  // 安全转换
        }
        
        if (unknownObj instanceof Parent) {
            Parent castedParent = (Parent) unknownObj;  // 安全转换
        }
        
        // 16. 复杂的 instanceof 表达式
        boolean result12 = (new Child()) instanceof Parent;
        boolean result13 = (getParentInstance()) instanceof Parent;

        java.util.Map.Entry[] entries = new java.util.Map.Entry[5];
        String[][] matrix = new String[3][4];
		Box<String>[] boxArray2 = (Box<String>[]) new Box[5];
        // 自定义类型一维数组
        Dog[] dogs = new Dog[5];

        // 自定义类型二维数组
        Dog[][] dogMatrix = new Dog[3][4];

        // 自定义类型带包名数组
        com.example.test.Dog[] dogsWithPackage = new com.example.test.Dog[2];
		
        // 自定义类型带包名二维数组
    	com.example.test.Dog[][] dogsWithPackageMatrix = new com.example.test.Dog[2][3];

        System.out.println("All examples executed successfully!");
    }
}
