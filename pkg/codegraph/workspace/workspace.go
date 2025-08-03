package workspace

import (
	"bufio"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"fmt"
	"golang.org/x/mod/modfile"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathNotExists = errors.New("no such file or directory")

type WorkspaceReader struct {
	logger logger.Logger
}

func NewWorkSpaceReader(logger logger.Logger) *WorkspaceReader {
	return &WorkspaceReader{
		logger: logger,
	}
}

func (w *WorkspaceReader) FindProjects(ctx context.Context, workspace string, visitPattern types.VisitPattern) []*Project {

	w.logger.Info("find_projects start to scan workspace：%s", workspace)

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
		projects = append(projects, &Project{
			Path: workspace,
			Name: projectName,
			Uuid: generateUuid(projectName, workspace),
		})
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
			if visitPattern.ShouldSkip(currentDir) {
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
						projects = append(projects, &Project{
							Path: subDir,
							Name: projectName,
							Uuid: generateUuid(projectName, subDir),
						})
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
		projects = append(projects, &Project{
			Path: workspace,
			Name: projectName,
			Uuid: generateUuid(projectName, workspace),
		})
	}
	// resolve go module
	for _, p := range projects {
		goModule, err := w.ResolveGoModule(ctx, p.Path)
		if err != nil {
			w.logger.Error("find_projects resolve go module err:%v", err)
		} else {
			w.logger.Error("find_projects resolved go module %s in project %s", goModule, p.Path)
			p.GoModule = goModule
		}
	}

	var projectNames string
	for _, p := range projects {
		projectNames += p.Name + types.Space
	}
	w.logger.Info("find_projects scan finish, found projects：%s", projectNames)

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
			// 如果EndLine > 0 且当前行号大于EndLine，则退出
			if option.EndLine > 0 && lineNum > option.EndLine {
				break
			}
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

// Walk 遍历目录
func (w *WorkspaceReader) Walk(ctx context.Context, dir string, walkFn types.WalkFunc, walkOpts types.WalkOptions) error {
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

	return filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
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

		if walkOpts.VisitPattern.ShouldSkip(relativePath) {
			// 跳过目录
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Convert Windows filePath separators to forward slashes
		relativePath = filepath.ToSlash(relativePath)

		// 只处理文件，不处理目录
		if info.IsDir() {
			return nil
		}

		// 构建 WalkContext
		walkCtx := &types.WalkContext{
			Path:         filePath,
			RelativePath: relativePath,
			Info: &types.FileInfo{
				Name:    info.Name(),
				Path:    relativePath,
				Size:    info.Size(),
				IsDir:   false,
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			},
			ParentPath: filepath.Dir(filePath),
		}
		file, err := os.Open(filePath)
		if err != nil && !walkOpts.IgnoreError {
			return err
		}
		if file == nil {
			return nil
		}

		defer file.Close()

		return walkFn(walkCtx, file)
	})
}

// ResolveGoModule TODO 子目录下的go.mod, go的import以module名开头，特殊处理
func (da *WorkspaceReader) ResolveGoModule(ctx context.Context, projectPath string) (string, error) {
	goMod, err := da.ReadFile(ctx, filepath.Join(projectPath, "go.mod"), types.ReadOptions{})
	if err != nil {
		return types.EmptyString, fmt.Errorf("resolve go project module failed: %v", err)
	}
	return modfile.ModulePath(goMod), nil
}
