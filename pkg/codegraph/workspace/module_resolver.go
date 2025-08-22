package workspace

import (
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/modfile"
)

// ModuleResolver 模块解析器，用于解析各种语言的包信息
type ModuleResolver struct {
	logger logger.Logger
}

func (mr *ModuleResolver) deduplicateStrings(emptyStrInput []string) []string {
	// 去重
	seen := make(map[string]bool)
	var result []string
	for _, str := range emptyStrInput {
		if !seen[str] && str != "" {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// GoWorkFile go.work文件结构
type GoWorkFile struct {
	Go  string
	Use []GoWorkUse
}

// GoWorkUse go.work文件中的use指令
type GoWorkUse struct {
	Path       string
	ModulePath string // 解析后的模块路径
}

// NewModuleResolver 创建新的模块解析器
func NewModuleResolver(logger logger.Logger) *ModuleResolver {
	return &ModuleResolver{
		logger: logger,
	}
} //TODO go.work submodules 解析有问题，其它语言支持。扫描目录优化； 有些只查询projects，不需要解析包，分离；

// ResolveProjectModules 解析项目的模块信息，递归多层处理，适应子项目、子模块的场景
func (mr *ModuleResolver) ResolveProjectModules(ctx context.Context, project *Project, path string, maxDepth int) error {
	if maxDepth == 0 {
		return nil
	}
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}
	stat, err := os.Stat(path)
	if err != nil {
		mr.logger.Debug("resolve project path %s err:%v", err)
		return nil
	}
	if !stat.IsDir() {
		return nil
	}
	// goStart := time.Now()
	// 解析Go模块
	goModules, err := mr.resolveGoModules(ctx, path)
	if err != nil {
		mr.logger.Debug("project path %s resolve go module err: %v", path, err)
	} else if len(goModules) > 0 {
		project.GoModules = append(project.GoModules, goModules...)
		mr.logger.Debug("project path %s resolved go modules: %v", path, goModules)
	}
	//mr.logger.Debug("resolve project path %s go modules cost %d ms.", path, time.Since(goStart).Milliseconds())

	//// 解析Java包前缀
	//javaPrefixes, err := mr.resolveJavaPackagePrefixes(ctx, path)
	//if err != nil {
	//	mr.logger.Debug("project path %s resolve java package prefixes err: %v", path, err)
	//} else if len(javaPrefixes) > 0 {
	//	project.JavaPackagePrefix = append(project.JavaPackagePrefix, javaPrefixes...)
	//	mr.logger.Debug("project path %s resolved java package prefixes: %v", path, javaPrefixes)
	//}
	//
	//mr.logger.Debug("resolve project path %s java packages cost %d ms.", path, time.Since(goStart).Milliseconds())

	//// 解析Python包
	//pythonPackages, err := mr.resolvePythonPackages(ctx, path)
	//if err != nil {
	//	mr.logger.Debug("project path %s resolved python packages err: %v", path, err)
	//} else if len(pythonPackages) > 0 {
	//	project.PythonPackages = append(project.PythonPackages, pythonPackages...)
	//	mr.logger.Debug("project path %s resolved python packages: %v", path, pythonPackages)
	//}
	//
	//mr.logger.Debug("resolve project path %s python packages cost %d ms.", path, time.Since(goStart).Milliseconds())
	//
	//// 解析C/C++头文件路径
	//cppIncludes, err := mr.resolveCppIncludes(ctx, path)
	//if err != nil {
	//	mr.logger.Debug("project path %s resolved c/cpp head dirEntries err: %v", path, err)
	//} else if len(cppIncludes) > 0 {
	//	project.CppIncludes = append(project.CppIncludes, cppIncludes...)
	//	mr.logger.Debug("project path %s resolved c/cpp head dirEntries: %v", path, cppIncludes)
	//}
	//
	//mr.logger.Debug("resolve project path %s cpp includes cost %d ms.", path, time.Since(goStart).Milliseconds())
	//
	//// 解析JavaScript/TypeScript包
	//jsPackages, err := mr.resolveJsPackages(ctx, path)
	//
	//if err != nil {
	//	mr.logger.Debug("project path %s resolved ts/js package err: %v", path, err)
	//} else if len(jsPackages) > 0 {
	//	project.JsPackages = append(project.JsPackages, jsPackages...)
	//	mr.logger.Debug("project path %s resolved ts/js package err: %v", path, cppIncludes)
	//}

	// mr.logger.Debug("resolve project path %s js packages cost %d ms.", path, time.Since(goStart).Milliseconds())

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		mr.logger.Debug("project path path %s list sub dirs err:%v", err)
		return nil
	}

	for _, f := range dirEntries {
		if f.IsDir() {
			subPath := filepath.Join(path, f.Name())
			if utils.IsHiddenFile(subPath) {
				mr.logger.Debug("%s is hidden dir, skip.", subPath)
				continue
			}
			if err = mr.ResolveProjectModules(ctx, project, subPath, maxDepth-1); err != nil {
				mr.logger.Debug("project path %s resolve err:%v", subPath, err)
			}
		}
	}

	return nil
}

// PomXML Maven POM文件结构
type PomXML struct {
	XMLName    xml.Name `xml:"project"`
	GroupID    string   `xml:"groupId"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
}

// resolveJavaPackagePrefixes 解析Java包前缀
func (mr *ModuleResolver) resolveJavaPackagePrefixes(ctx context.Context, projectPath string) ([]string, error) {
	var prefixes []string

	// 1. 从pom.xml文件解析groupId和artifactId
	pomPath := filepath.Join(projectPath, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		pomPrefixes, err := mr.parsePomXML(pomPath)
		if err != nil {
			mr.logger.Error("resolve pom.xml err: %v", err)
		} else if len(pomPrefixes) > 0 {
			prefixes = append(prefixes, pomPrefixes...)
		}
	}

	// 2. 从src/main/java目录结构推断包前缀
	javaSrcPath := filepath.Join(projectPath, "src", "main", "java")
	if _, err := os.Stat(javaSrcPath); err == nil {
		dirPrefixes, err := mr.inferJavaPrefixFromDir(javaSrcPath)
		if err != nil {
			mr.logger.Error("resolve java package prefix err: %v", err)
		} else if len(dirPrefixes) > 0 {
			prefixes = append(prefixes, dirPrefixes...)
		}
	}

	// 去重
	return utils.DeDuplicate(prefixes), nil
}

// parsePomXML 解析pom.xml文件，提取包前缀
func (mr *ModuleResolver) parsePomXML(pomPath string) ([]string, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("read pom.xml err: %v", err)
	}

	var pom PomXML
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, fmt.Errorf("parse pom.xml err: %v", err)
	}

	var prefixes []string
	if pom.GroupID != "" {
		prefixes = append(prefixes, pom.GroupID)
	}
	if pom.ArtifactID != "" {
		prefixes = append(prefixes, pom.ArtifactID)
	}
	if pom.GroupID != "" && pom.ArtifactID != "" {
		prefixes = append(prefixes, fmt.Sprintf("%s.%s", pom.GroupID, pom.ArtifactID))
	}

	return prefixes, nil
}

// inferJavaPrefixFromDir 从目录结构推断Java包前缀
func (mr *ModuleResolver) inferJavaPrefixFromDir(javaSrcPath string) ([]string, error) {
	var prefixes []string

	err := filepath.Walk(javaSrcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理目录
		if !info.IsDir() {
			return nil
		}

		// 跳过src/main/java目录本身
		if path == javaSrcPath {
			return nil
		}

		// 检查目录下是否有.java文件
		hasJavaFiles := false
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".java") {
				hasJavaFiles = true
				break
			}
		}

		if hasJavaFiles {
			// 将相对路径转换为包前缀
			relPath, err := filepath.Rel(javaSrcPath, path)
			if err != nil {
				return err
			}
			prefix := strings.ReplaceAll(relPath, string(filepath.Separator), ".")
			prefixes = append(prefixes, prefix)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return prefixes, nil
}

// SetupPy setup.py文件结构
type SetupPy struct {
	Name     string `json:"name"`
	Packages struct {
		Find struct {
			Where string `json:"where"`
		} `json:"find"`
	} `json:"packages"`
}

// PyProjectToml pyproject.toml文件结构
type PyProjectToml struct {
	Tool struct {
		Poetry struct {
			Name string `json:"name"`
		} `json:"poetry"`
	} `json:"tool"`
	Project struct {
		Name string `json:"name"`
	} `json:"project"`
}

// resolvePythonPackages 解析Python包
func (mr *ModuleResolver) resolvePythonPackages(ctx context.Context, projectPath string) ([]string, error) {
	var packages []string

	// 1. 从setup.py文件解析包名
	setupPyPath := filepath.Join(projectPath, "setup.py")
	if _, err := os.Stat(setupPyPath); err == nil {
		setupPackages, err := mr.parseSetupPy(setupPyPath)
		if err != nil {
			mr.logger.Error("parse setup.py err: %v", err)
		} else if len(setupPackages) > 0 {
			packages = append(packages, setupPackages...)
		}
	}

	// 2. 从pyproject.toml文件解析包名
	pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		pyprojectPackages, err := mr.parsePyProjectToml(pyprojectPath)
		if err != nil {
			mr.logger.Error("parse pyproject.toml err: %v", err)
		} else if len(pyprojectPackages) > 0 {
			packages = append(packages, pyprojectPackages...)
		}
	}

	// 3. 从项目目录结构查找Python包
	dirPackages, err := mr.findPythonPackages(projectPath)
	if err != nil {
		mr.logger.Error("find python packages err: %v", err)
	} else if len(dirPackages) > 0 {
		packages = append(packages, dirPackages...)
	}

	// 去重
	return utils.DeDuplicate(packages), nil
}

// parseSetupPy 解析setup.py文件
func (mr *ModuleResolver) parseSetupPy(setupPyPath string) ([]string, error) {
	// 注意： setup.py是Python文件，解析比较复杂
	// 这里使用简化的方法，只读取文件内容并尝试提取包名
	data, err := os.ReadFile(setupPyPath)
	if err != nil {
		return nil, fmt.Errorf("read setup.py err: %v", err)
	}

	content := string(data)
	var packages []string

	// 简单提取name字段
	if strings.Contains(content, "name=") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name=") || strings.Contains(line, "name=") {
				// 提取引号中的内容
				start := strings.Index(line, "\"")
				if start == -1 {
					start = strings.Index(line, "'")
				}
				if start != -1 {
					start++
					end := strings.Index(line[start:], "\"")
					if end == -1 {
						end = strings.Index(line[start:], "'")
					}
					if end != -1 {
						name := line[start : start+end]
						packages = append(packages, name)
					}
				}
			}
		}
	}

	return packages, nil
}

// parsePyProjectToml 解析pyproject.toml文件
func (mr *ModuleResolver) parsePyProjectToml(pyprojectPath string) ([]string, error) {
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return nil, fmt.Errorf("read pyproject.toml err: %v", err)
	}

	var pyproject PyProjectToml
	if err := toml.Unmarshal(data, &pyproject); err != nil {
		// 如果JSON解析失败，尝试简单的文本解析
		return mr.parsePyProjectTomlText(string(data)), nil
	}

	var packages []string
	if pyproject.Tool.Poetry.Name != "" {
		packages = append(packages, pyproject.Tool.Poetry.Name)
	}
	if pyproject.Project.Name != "" {
		packages = append(packages, pyproject.Project.Name)
	}

	return packages, nil
}

// parsePyProjectTomlText 简单文本解析pyproject.toml
func (mr *ModuleResolver) parsePyProjectTomlText(content string) []string {
	var packages []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name = ") {
			// 提取引号中的内容
			start := strings.Index(line, "\"")
			if start == -1 {
				start = strings.Index(line, "'")
			}
			if start != -1 {
				start++
				end := strings.Index(line[start:], "\"")
				if end == -1 {
					end = strings.Index(line[start:], "'")
				}
				if end != -1 {
					name := line[start : start+end]
					packages = append(packages, name)
				}
			}
		}
	}

	return packages
}

// findPythonPackages 从项目目录结构查找Python包
func (mr *ModuleResolver) findPythonPackages(projectPath string) ([]string, error) {
	var packages []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理目录
		if !info.IsDir() {
			return nil
		}

		// 跳过隐藏目录和特殊目录
		if strings.HasPrefix(info.Name(), ".") || info.Name() == "__pycache__" {
			return filepath.SkipDir
		}

		// 检查是否是Python包（包含__init__.py文件）
		initPath := filepath.Join(path, "__init__.py")
		if _, err := os.Stat(initPath); err == nil {
			// 计算包名
			relPath, err := filepath.Rel(projectPath, path)
			if err != nil {
				return err
			}
			packageName := strings.ReplaceAll(relPath, string(filepath.Separator), ".")
			packages = append(packages, packageName)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return packages, nil
}

// resolveCppIncludes 解析C/C++头文件路径
func (mr *ModuleResolver) resolveCppIncludes(ctx context.Context, projectPath string) ([]string, error) {
	var includes []string

	// 检查项目中的常见目录
	commonDirs := []string{
		"include",
		"src",
		"lib",
		"libs",
		"thirdparty",
		"third_party",
	}

	for _, dir := range commonDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); err == nil {
			relIncludes, err := mr.findCppHeadersInDir(projectPath, dirPath)
			if err != nil {
				mr.logger.Error("find %s c/cpp head files err: %v", dir, err)
			} else {
				includes = append(includes, relIncludes...)
			}
		}
	}

	// 检查项目根目录下的头文件
	rootIncludes, err := mr.findCppHeadersInDir(projectPath, projectPath)
	if err != nil {
		mr.logger.Error("find %s c/cpp head files err: %v", err)
	} else if len(rootIncludes) > 0 {
		includes = append(includes, rootIncludes...)
	}

	// 去重
	return utils.DeDuplicate(includes), nil
}

// findCppHeadersInDir 在指定目录中查找C/C++头文件
func (mr *ModuleResolver) findCppHeadersInDir(basePath, dirPath string) ([]string, error) {
	var includes []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件
		if info.IsDir() {
			return nil
		}

		// 检查是否是头文件
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".h" || ext == ".hpp" || ext == ".hxx" {
			// 计算相对路径
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				return err
			}
			// 转换为相对路径的目录
			relDir := filepath.Dir(relPath)
			if relDir != "." {
				includes = append(includes, relDir)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return includes, nil
}

// PackageJSON package.json文件结构
type PackageJSON struct {
	Name            string            `json:"name"`
	Private         bool              `json:"private"`
	Workspaces      []string          `json:"workspaces"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// resolveJsPackages 解析JavaScript/TypeScript包
func (mr *ModuleResolver) resolveJsPackages(ctx context.Context, projectPath string) ([]string, error) {
	var packages []string

	// 1. 从package.json文件解析包名
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		jsonPackages, err := mr.parsePackageJson(packageJsonPath)
		if err != nil {
			mr.logger.Error("parse package.json err: %v", err)
		} else if len(jsonPackages) > 0 {
			packages = append(packages, jsonPackages...)
		}
	}

	// 2. 从项目目录结构查找JavaScript/TypeScript包
	dirPackages, err := mr.findJsPackages(projectPath)
	if err != nil {
		mr.logger.Error("find js/ts package err: %v", err)
	} else if len(dirPackages) > 0 {
		packages = append(packages, dirPackages...)
	}

	// 去重
	return utils.DeDuplicate(packages), nil
}

// parsePackageJson 解析package.json文件
func (mr *ModuleResolver) parsePackageJson(packageJsonPath string) ([]string, error) {
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, fmt.Errorf("read package.json file err: %v", err)
	}

	var packageJson PackageJSON
	if err := json.Unmarshal(data, &packageJson); err != nil {
		return nil, fmt.Errorf("parse package.json file err: %v", err)
	}

	var packages []string
	if packageJson.Name != "" && !packageJson.Private {
		packages = append(packages, packageJson.Name)
	}

	// 如果是monorepo，添加workspace名称
	if len(packageJson.Workspaces) > 0 {
		for _, workspace := range packageJson.Workspaces {
			// 简单处理，实际可能需要更复杂的逻辑
			if strings.HasPrefix(workspace, "packages/*") {
				// 尝试从目录结构推断包名
				baseDir := filepath.Dir(packageJsonPath)
				packagesDir := filepath.Join(baseDir, "packages")
				if _, err := os.Stat(packagesDir); err == nil {
					entries, err := os.ReadDir(packagesDir)
					if err == nil {
						for _, entry := range entries {
							if entry.IsDir() {
								packages = append(packages, entry.Name())
							}
						}
					}
				}
			}
		}
	}

	return packages, nil
}

// findJsPackages 从项目目录结构查找JavaScript/TypeScript包
func (mr *ModuleResolver) findJsPackages(projectPath string) ([]string, error) {
	var packages []string

	// 检查常见源代码目录
	srcDirs := []string{
		"src",
		"lib",
		"packages",
	}

	for _, srcDir := range srcDirs {
		srcPath := filepath.Join(projectPath, srcDir)
		if _, err := os.Stat(srcPath); err == nil {
			// 检查该目录下是否有package.json文件
			entries, err := os.ReadDir(srcPath)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						// 检查子目录中是否有package.json
						subPackageJson := filepath.Join(srcPath, entry.Name(), "package.json")
						if _, err := os.Stat(subPackageJson); err == nil {
							packages = append(packages, entry.Name())
						}
					}
				}
			}
		}
	}

	return packages, nil
}

// resolveGoWorkspace 解析go.work文件，提取其中引用的所有模块路径
func (mr *ModuleResolver) resolveGoWorkspace(ctx context.Context, projectPath string) ([]string, error) {
	goWorkPath := filepath.Join(projectPath, "go.work")

	// 检查go.work文件是否存在
	if _, err := os.Stat(goWorkPath); err != nil {
		return nil, nil
	}

	mr.logger.Debug("parsing go.work file: %s", goWorkPath)

	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, fmt.Errorf("read go.work file err: %v", err)
	}

	// 解析go.work文件
	goWorkFile, err := modfile.ParseWork(goWorkPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.work file err: %v", err)
	}

	var modules []string

	// 遍历go.work中的use指令，获取所有模块路径
	for _, use := range goWorkFile.Use {
		usePath := filepath.Join(projectPath, use.Path)

		// 解析每个use路径下的go.mod文件，获取模块路径
		goModPath := filepath.Join(usePath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			modData, err := os.ReadFile(goModPath)
			if err != nil {
				mr.logger.Debug("read go.mod file %s err: %v", goModPath, err)
				continue
			}

			modulePath := modfile.ModulePath(modData)
			if modulePath != "" {
				modules = append(modules, modulePath)
				mr.logger.Debug("resolved go module from go.work: %s -> %s", use.Path, modulePath)
			}
		} else {
			mr.logger.Debug("go.mod file not found in use path: %s", usePath)
		}
	}

	mr.logger.Debug("resolved %d modules from go.work file: %s", len(modules), goWorkPath)

	return modules, nil
}

// resolveGoModules 解析Go模块，优先使用go.work文件，不存在时解析go.mod文件
func (mr *ModuleResolver) resolveGoModules(ctx context.Context, projectPath string) ([]string, error) {
	var modules []string

	// 首先尝试解析go.work文件
	goWorkModules, err := mr.resolveGoWorkspace(ctx, projectPath)
	if err != nil {
		mr.logger.Debug("resolve go.work file err: %v", err)
	}
	if len(goWorkModules) > 0 {
		// 如果go.work文件存在且解析成功，直接返回结果
		modules = append(modules, goWorkModules...)
	}

	// 如果没有go.work文件或解析失败，则解析go.mod文件
	goModPath := filepath.Join(projectPath, "go.mod")

	if _, err := os.Stat(goModPath); err == nil {
		mr.logger.Debug("parsing go.mod file: %s", goModPath)

		data, err := os.ReadFile(goModPath)
		if err != nil {
			return nil, fmt.Errorf("module_resover parse go.mod file err: %v", err)
		}

		modulePath := modfile.ModulePath(data)
		if modulePath != "" {
			modules = append(modules, modulePath)
			mr.logger.Debug("resolved go module: %s", modulePath)
		}

	}

	return utils.DeDuplicate(modules), nil
}
