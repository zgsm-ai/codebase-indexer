package workspace

import (
	"bufio"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrPathNotExists = errors.New("no such file or directory")

var DefaultVisitPattern = &types.VisitPattern{ExcludeDirs: []string{".git", ".idea", ".vscode", "node_modules", "vendor"}}

const ReadFileMaxLine = 5_000
const MaxFileVisitLimit = 20_0000

type WorkspaceReader struct {
	logger logger.Logger
}

func NewWorkSpaceReader(logger logger.Logger) *WorkspaceReader {
	return &WorkspaceReader{
		logger: logger,
	}
}

func (w *WorkspaceReader) FindProjects(ctx context.Context, workspace string, resolveModule bool, visitPattern *types.VisitPattern) []*Project {

	start := time.Now()
	w.logger.Info("start to scan workspace：%s", workspace)
	if visitPattern == nil {
		visitPattern = DefaultVisitPattern
	}
	// 创建 ModuleResolver 实例
	moduleResolver := NewModuleResolver(w.logger)

	var projects []*Project
	maxLayer := 3
	maxEntries := 2000
	entryCount := 0
	foundGit := false

	// 辅助函数：判断目录下是否有 .git 目录
	hasGitDir := func(dir string) bool {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		return err == nil && info.IsDir()
	}

	// 1. 当前目录是 git 仓库
	if hasGitDir(workspace) {
		projectName := filepath.Base(workspace)
		project := &Project{
			Path: workspace,
			Name: projectName,
			Uuid: generateUuid(projectName, workspace),
		}
		projects = append(projects, project)
		if resolveModule {
			if err := moduleResolver.ResolveProjectModules(ctx, project, project.Path, 2); err != nil {
				w.logger.Error("resolve project modules err:%v", err)
			}
		}

		foundGit = true
	} else {
		// 2. 使用广度优先遍历查找 git 仓库
		type queueItem struct {
			dir   string
			depth int
		}

		queue := []queueItem{{dir: workspace, depth: 1}}

		for len(queue) > 0 && entryCount < maxEntries {
			current := queue[0]
			queue = queue[1:]

			if current.depth > maxLayer {
				continue
			}
			currentDir := current.dir

			// 应用过滤规则
			if skip, _ := visitPattern.ShouldSkip(&types.FileInfo{
				Path: currentDir, IsDir: true}); skip {
				continue
			}

			entries, err := os.ReadDir(currentDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entryCount >= maxEntries {
					break
				}

				if entry.IsDir() {
					subDir := filepath.Join(currentDir, entry.Name())

					// 跳过隐藏目录
					if strings.HasPrefix(entry.Name(), types.Dot) {
						continue
					}

					if hasGitDir(subDir) {
						projectName := filepath.Base(subDir)
						project := &Project{
							Path: subDir,
							Name: projectName,
							Uuid: generateUuid(projectName, subDir),
						}
						projects = append(projects, project)
						if resolveModule {
							if err = moduleResolver.ResolveProjectModules(ctx, project, project.Path, 2); err != nil {
								w.logger.Error("resolve project modules err:%v", err)
							}
						}

						foundGit = true
						// 不递归 .git 仓库下的子目录
						continue
					}

					entryCount++
					queue = append(queue, queueItem{dir: subDir, depth: current.depth + 1})
				}
			}
		}
	}

	// 3. 没有发现任何 git 仓库，将当前目录作为唯一项目
	if !foundGit {
		projectName := filepath.Base(workspace)
		project := &Project{
			Path: workspace,
			Name: projectName,
			Uuid: generateUuid(projectName, workspace),
		}
		projects = append(projects, project)
		if resolveModule {
			if err := moduleResolver.ResolveProjectModules(ctx, project, project.Path, 2); err != nil {
				w.logger.Error("resolve project modules err:%v", err)
			}
		}
	}

	var projectNames string
	var goModules []string
	for _, p := range projects {
		projectNames += p.Name + types.Space
		goModules = append(goModules, p.GoModules...)
	}
	w.logger.Info("scan finish, cost %d ms, found projects：%s, go modules:%s",
		time.Since(start).Milliseconds(), projectNames, goModules)

	return projects
}

// ReadFile 读取单个文件
func (w *WorkspaceReader) ReadFile(ctx context.Context, path string, option types.ReadOptions) ([]byte, error) {

	exists, err := w.Exists(ctx, path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPathNotExists
	}

	// 如果StartLine <= 0，设置为1
	if option.StartLine <= 0 {
		option.StartLine = 1
	}
	// endLine 设置默认值，且不超过最大值
	if option.EndLine <= 0 || option.EndLine > ReadFileMaxLine {
		option.EndLine = ReadFileMaxLine
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 创建reader来读取文件
	reader := bufio.NewReader(file)
	var lines []string
	lineNum := 1

	// 读取行
	for {

		// 如果EndLine > 0 且当前行号大于EndLine，则退出
		if lineNum > option.EndLine {
			break
		}

		// 读取一行，允许超过默认缓冲区大小
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// 处理可能被截断的行
		var lineBuffer []byte
		lineBuffer = append(lineBuffer, line...)
		for isPrefix {
			line, isPrefix, err = reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			lineBuffer = append(lineBuffer, line...)
		}

		// 转换为字符串
		lineStr := string(lineBuffer)

		// 如果当前行号大于等于StartLine，则添加到结果中
		if lineNum >= option.StartLine {
			lines = append(lines, lineStr)
		}
		lineNum++
	}

	// 将结果转换为字节数组
	return []byte(strings.Join(lines, types.LF)), nil
}

// Exists 判断文件/目录是否存在
func (w *WorkspaceReader) Exists(ctx context.Context, path string) (bool, error) {
	if path == types.EmptyString {
		return false, errors.New("path cannot be empty")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// WalkFile 遍历目录下的文件
func (w *WorkspaceReader) WalkFile(ctx context.Context, dir string, walkFn types.WalkFunc, walkOpts types.WalkOptions) error {
	if dir == types.EmptyString {
		return errors.New("dir cannot be empty")
	}

	exists, err := w.Exists(ctx, dir)
	if err != nil {
		return err
	}

	if !exists {
		return ErrPathNotExists
	}
	if walkOpts.VisitPattern.MaxVisitLimit <= 0 {
		walkOpts.VisitPattern.MaxVisitLimit = MaxFileVisitLimit
	}

	var visitCount int

	return filepath.WalkDir(dir, func(filePath string, info fs.DirEntry, err error) error {
		if err != nil && !walkOpts.IgnoreError {
			return err
		}

		// 跳过隐藏文件和目录
		if utils.IsHiddenFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath, err := filepath.Rel(dir, filePath)
		if err != nil && !walkOpts.IgnoreError {
			return err
		}

		if relativePath == types.Dot {
			return nil
		}

		skip, err := walkOpts.VisitPattern.ShouldSkip(
			&types.FileInfo{
				Name:  info.Name(),
				Path:  filePath,
				IsDir: info.IsDir(),
			})
		if skip {
			// 跳过目录
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if errors.Is(err, filepath.SkipDir) || errors.Is(err, filepath.SkipAll) {
			return err
		}

		// Convert Windows filePath separators to forward slashes
		relativePath = filepath.ToSlash(relativePath)

		visitCount++
		if visitCount > walkOpts.VisitPattern.MaxVisitLimit {
			return filepath.SkipAll
		}

		// 只处理文件，不处理目录
		if info.IsDir() {
			return nil
		}

		// 构建 WalkContext
		walkCtx := &types.WalkContext{
			Path:         filePath,
			RelativePath: relativePath,
			Info: &types.FileInfo{
				Name: info.Name(),
				Path: filePath,
			},
			ParentPath: filepath.Dir(filePath),
		}
		fileInfo, _ := info.Info()
		if fileInfo != nil {
			walkCtx.Info.Size = fileInfo.Size()
			walkCtx.Info.ModTime = fileInfo.ModTime()
			walkCtx.Info.Mode = fileInfo.Mode()
			walkCtx.Info.IsDir = fileInfo.IsDir()
		}

		return walkFn(walkCtx)
	})
}

func (l *WorkspaceReader) Tree(ctx context.Context, workspacePath string, subDir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	if workspacePath == types.EmptyString {
		return nil, errors.New("workspacePath cannot be empty")
	}

	exists, err := l.Exists(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPathNotExists
	}

	// 使用 map 来构建目录树
	nodeMap := make(map[string]*types.TreeNode)
	walkBasePath := filepath.Join(workspacePath, subDir)

	err = filepath.WalkDir(walkBasePath, func(absFilePath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件和目录
		if utils.IsHiddenFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 获取相对路径，相对workspacePath
		codeBaseRelativePath, err := filepath.Rel(workspacePath, absFilePath)
		if err != nil {
			return err
		}
		// 获取相对路径，相对workspacePath + subdir
		walkBaseRelativePath, err := filepath.Rel(walkBasePath, absFilePath)
		if err != nil {
			return err
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(walkBaseRelativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(walkBaseRelativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查深度限制
		if option.MaxDepth > 0 {
			// 相对根+subdir 的depth
			depth := len(strings.Split(walkBaseRelativePath, string(filepath.Separator)))
			if depth > option.MaxDepth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		var currentPath string
		var parts []string

		// 如果是根目录本身，跳过
		if walkBaseRelativePath == types.Dot || utils.PathEqual(walkBaseRelativePath, subDir) {
			return nil
		}

		// 如果是根目录下的文件或目录
		if !strings.Contains(walkBaseRelativePath, string(filepath.Separator)) {
			currentPath = walkBaseRelativePath
			parts = []string{walkBaseRelativePath}
		} else {
			// 处理子目录中的文件和目录
			parts = strings.Split(walkBaseRelativePath, string(filepath.Separator))
			currentPath = parts[0]
		}

		// 处理路径中的每一级
		for i, part := range parts {
			if part == "" {
				continue
			}

			if i > 0 {
				currentPath = filepath.Join(currentPath, part)
			}

			// 如果节点已存在，跳过
			if _, exists := nodeMap[currentPath]; exists {
				continue
			}

			// 创建新节点
			isLast := i == len(parts)-1

			node := &types.TreeNode{
				FileInfo: types.FileInfo{
					Name: part,
					Path: codeBaseRelativePath,
				},
				Children: make([]*types.TreeNode, 0),
			}

			fileInfo, _ := info.Info()
			if fileInfo != nil {
				node.FileInfo.Size = fileInfo.Size()
				node.FileInfo.ModTime = fileInfo.ModTime()
				node.FileInfo.Mode = fileInfo.Mode()
				node.FileInfo.IsDir = isLast && info.IsDir()
			}

			// 将节点添加到 map
			nodeMap[currentPath] = node

			// 如果不是根级节点，添加到父节点的子节点列表
			if i > 0 {
				parentPath := filepath.Dir(currentPath)
				if parent, exists := nodeMap[parentPath]; exists {
					parent.Children = append(parent.Children, node)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// 构建根节点列表
	var rootNodes []*types.TreeNode
	for path, node := range nodeMap {
		if !strings.Contains(path, string(filepath.Separator)) {
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes, nil
}

func (w *WorkspaceReader) GetProjectByFilePath(ctx context.Context, workspace string, filePath string, resolveModule bool) (*Project, error) {
	projects := w.FindProjects(ctx, workspace, resolveModule, DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("found no projects in workspace %s", workspace)
	}
	for _, p := range projects {
		if strings.HasPrefix(filePath, p.Path) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("failed to find project which file %s belongs to in workspace %s", filePath, workspace)
}

func (w *WorkspaceReader) Stat(filePath string) (*types.FileInfo, error) {
	stat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, ErrPathNotExists
	}
	if err != nil {
		return nil, err
	}
	return &types.FileInfo{
		Name:    filepath.Base(filePath),
		Path:    filePath,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
	}, nil
}

func (w *WorkspaceReader) List(ctx context.Context, path string) ([]*types.FileInfo, error) {
	// 检查上下文是否已取消（尽早退出）
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory err: %w", err)
	}

	// 预分配切片容量，避免动态扩容
	files := make([]*types.FileInfo, 0, len(entries))

	for _, e := range entries {
		// 循环中检查上下文，支持中途取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		// 构建完整路径（确保路径格式正确）
		fullPath := filepath.Join(path, e.Name())

		fileInfo := &types.FileInfo{
			Name:  e.Name(),
			Path:  fullPath,
			IsDir: e.IsDir(),
		}

		// 非目录文件获取详细信息
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				// 记录错误但不中断，仅文件元信息可能不完整
				w.logger.Warn("workspace_reader failed to get file info for %s: %v", fullPath, err)
			} else {
				fileInfo.Size = info.Size()
				fileInfo.Mode = info.Mode()
				fileInfo.ModTime = info.ModTime()
			}
		}

		files = append(files, fileInfo)
	}

	return files, nil
}
