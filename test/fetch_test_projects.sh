#!/bin/bash

# 目标目录
TARGET_DIR="/test/tmp/projects"
mkdir -p ${TARGET_DIR}

# 使用普通数组和分隔符定义每种语言的多个项目
LANGUAGES=(
    "c"
    "cpp"
    "csharp"
    "go"
    "java"
    "javascript"
    "kotlin"
    "python"
    "ruby"
    "rust"
    "scala"
    "typescript"
)

# 每个语言对应的项目列表（用|分隔） - 使用SSH地址
C_PROJECTS="git@github.com:redis/redis.git|git@github.com:sqlite/sqlite.git|git@github.com:openssl/openssl.git|git@github.com:netdata/netdata.git"
CPP_PROJECTS="git@github.com:protocolbuffers/protobuf.git|git@github.com:grpc/grpc.git|git@github.com:ggml-org/whisper.cpp.git"
CSHARP_PROJECTS="git@github.com:dotnet/efcore.git|git@github.com:aspnetcore/AspNetCore.git|git@github.com:mono/mono.git"
GO_PROJECTS="git@github.com:gin-gonic/gin.git|git@github.com:kubernetes/kubernetes.git|git@github.com:hypermodeinc/dgraph.git"
JAVA_PROJECTS="git@github.com:apache/hadoop.git|git@github.com:apache/maven.git|git@github.com:macrozheng/mall.git"
JAVASCRIPT_PROJECTS="git@github.com:facebook/react.git|git@github.com:vuejs/vue.git|git@github.com:angular/angular.git"
KOTLIN_PROJECTS="git@github.com:JetBrains/kotlin.git|git@github.com:androidx/androidx.git|git@github.com:Kotlin/kotlinx.coroutines.git"
PYTHON_PROJECTS="git@github.com:django/django.git|git@github.com:pandas-dev/pandas.git|git@github.com:scikit-learn/scikit-learn.git"
RUBY_PROJECTS="git@github.com:rails/rails.git|git@github.com:jekyll/jekyll.git|git@github.com:hashicorp/vagrant.git"
RUST_PROJECTS="git@github.com:rust-lang/rust.git|git@github.com:denoland/deno.git|git@github.com:starship/starship.git"
SCALA_PROJECTS="git@github.com:scala/scala.git|git@github.com:apache/spark.git|git@github.com:akka/akka.git"
TYPESCRIPT_PROJECTS="git@github.com:microsoft/TypeScript.git|git@github.com:vuejs/vue-next.git|git@github.com:sveltejs/svelte.git"

# 检查目标目录是否存在
if [ ! -d "$TARGET_DIR" ]; then
    echo "错误：目标目录 '$TARGET_DIR' 不存在"
    exit 1
fi

# 函数：获取语言对应的项目列表变量名
get_projects_var() {
    local lang="$1"
    case "$lang" in
        "c") echo "C_PROJECTS" ;;
        "cpp") echo "CPP_PROJECTS" ;;
        "csharp") echo "CSHARP_PROJECTS" ;;
        "go") echo "GO_PROJECTS" ;;
        "java") echo "JAVA_PROJECTS" ;;
        "javascript") echo "JAVASCRIPT_PROJECTS" ;;
        "kotlin") echo "KOTLIN_PROJECTS" ;;
        "python") echo "PYTHON_PROJECTS" ;;
        "ruby") echo "RUBY_PROJECTS" ;;
        "rust") echo "RUST_PROJECTS" ;;
        "scala") echo "SCALA_PROJECTS" ;;
        "typescript") echo "TYPESCRIPT_PROJECTS" ;;
        *) echo "" ;;
    esac
}

# 遍历每种语言
for lang in "${LANGUAGES[@]}"; do
    lang_dir="$TARGET_DIR/$lang"
    
    # 获取该语言的项目列表变量名
    projects_var=$(get_projects_var "$lang")
    if [ -z "$projects_var" ]; then
        echo "警告：未知语言 '$lang'，跳过..."
        continue
    fi
    
    # 间接引用变量获取项目列表
    projects="${!projects_var}"
    
    # 创建语言目录（如果不存在）
    mkdir -p "$lang_dir"
    cd "$lang_dir" || {
        echo "错误：无法进入目录 $lang_dir"
        continue
    }
    
    echo "=== 开始拉取 $lang 项目 ==="
    
    # 分割项目列表并遍历
    IFS='|' read -ra repo_list <<< "$projects"
    for repo in "${repo_list[@]}"; do
        # 从URL提取项目名称
        project_name=$(basename "$repo" .git)
        
        # 创建项目子目录
        mkdir -p "$project_name"
        cd "$project_name" || {
            echo "错误：无法创建项目目录 $project_name"
            continue
        }
        
        echo "正在拉取 $lang 项目：$repo"
        git clone --depth 1 "$repo" . || {
            echo "警告：拉取项目 $project_name 失败，跳过..."
        }
        
        cd "$lang_dir"  # 返回语言目录
    done
    
    cd "$SCRIPT_DIR"  # 返回脚本目录
    echo "=== $lang 项目拉取完成 ==="
done

echo "所有语言的项目拉取完成！路径：$TARGET_DIR"
