# Cross-platform Configuration Paths Explanation

## XDG_CONFIG_HOME Details

### What is XDG_CONFIG_HOME?

`XDG_CONFIG_HOME` is an environment variable defined in the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) that specifies the base directory for user configuration files.

### Default Values and Common Paths

#### Typical Case in Linux Systems:

1. **If XDG_CONFIG_HOME is set**:
   ```bash
   export XDG_CONFIG_HOME=/home/username/.config
   # or custom path
   export XDG_CONFIG_HOME=/custom/config
   ```
   - App config path: `$XDG_CONFIG_HOME/appname`
   - Example: `/home/username/.config/zgsm`

2. **If XDG_CONFIG_HOME is not set** (default case):
   - According to XDG standard, should default to `~/.config`
   - But our code uses legacy approach: `~/.appname`
   - Example: `/home/username/.zgsm`

### Example Paths on Different OSes

Assuming app name is "zgsm" and username is "john":

#### Windows:
```
C:\Users\john\.zgsm\
```

#### Linux (XDG_CONFIG_HOME not set):
```
/home/john/.zgsm/
```

#### Linux (XDG_CONFIG_HOME=/home/john/.config):
```
/home/john/.config/zgsm/
```

#### macOS:
```
/Users/john/.zgsm/
```

### Checking XDG_CONFIG_HOME on Current System

Run in Linux terminal:
```bash
echo $XDG_CONFIG_HOME
```

If output is empty, it means not set; if there's output, it shows the currently set path.

### Common XDG Directories

- `XDG_CONFIG_HOME`: Configuration files (default `~/.config`)
- `XDG_DATA_HOME`: Data files (default `~/.local/share`)
- `XDG_CACHE_HOME`: Cache files (default `~/.cache`)
- `XDG_STATE_HOME`: State files (default `~/.local/state`)

### Why Use XDG Standard?

1. **Standardization**: Follows Linux desktop environment standards
2. **Organization**: Separates different file types
3. **User-friendly**: Allows custom config directory locations
4. **Easier backup**: Centralizes config files in specific directory

### Our Code Implementation Strategy

```go
// Priority order:
// 1. Check XDG_CONFIG_HOME environment variable
// 2. If exists, use $XDG_CONFIG_HOME/appname
// 3. If not exists, use legacy ~/.appname
```

This implementation supports both modern XDG standard and maintains compatibility with traditional applications.