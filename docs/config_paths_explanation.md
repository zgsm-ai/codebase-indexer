# 跨平台配置路径说明

## XDG_CONFIG_HOME 详解

### 什么是 XDG_CONFIG_HOME？

`XDG_CONFIG_HOME` 是 [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) 标准中定义的环境变量，用于指定用户配置文件的基础目录。

### 默认值和常见路径

#### Linux 系统中的典型情况：

1. **如果设置了 XDG_CONFIG_HOME**：
   ```bash
   export XDG_CONFIG_HOME=/home/用户名/.config
   # 或者自定义路径
   export XDG_CONFIG_HOME=/custom/config
   ```
   - 应用配置路径：`$XDG_CONFIG_HOME/appname`
   - 示例：`/home/用户名/.config/zgsm`

2. **如果未设置 XDG_CONFIG_HOME**（默认情况）：
   - 按照 XDG 标准，应该默认为 `~/.config`
   - 但我们的代码采用传统方式：`~/.appname`
   - 示例：`/home/用户名/.zgsm`

### 各操作系统的实际路径示例

假设应用名为 "zgsm"，用户名为 "john"：

#### Windows:
```
C:\Users\john\.zgsm\
```

#### Linux (未设置 XDG_CONFIG_HOME):
```
/home/john/.zgsm/
```

#### Linux (设置了 XDG_CONFIG_HOME=/home/john/.config):
```
/home/john/.config/zgsm/
```

#### macOS:
```
/Users/john/.zgsm/
```

### 检查当前系统的 XDG_CONFIG_HOME

在 Linux 终端中运行：
```bash
echo $XDG_CONFIG_HOME
```

如果输出为空，说明未设置；如果有输出，则显示当前设置的路径。

### 常见的 XDG 目录

- `XDG_CONFIG_HOME`: 配置文件 (默认 `~/.config`)
- `XDG_DATA_HOME`: 数据文件 (默认 `~/.local/share`)
- `XDG_CACHE_HOME`: 缓存文件 (默认 `~/.cache`)
- `XDG_STATE_HOME`: 状态文件 (默认 `~/.local/state`)

### 为什么使用 XDG 标准？

1. **标准化**：遵循 Linux 桌面环境的标准
2. **组织性**：将不同类型的文件分类存储
3. **用户友好**：用户可以自定义配置目录位置
4. **备份方便**：配置文件集中在特定目录

### 我们代码的实现策略

```go
// 优先级顺序：
// 1. 检查 XDG_CONFIG_HOME 环境变量
// 2. 如果存在，使用 $XDG_CONFIG_HOME/appname
// 3. 如果不存在，使用传统的 ~/.appname
```

这种实现方式既支持现代的 XDG 标准，又保持了与传统应用的兼容性。