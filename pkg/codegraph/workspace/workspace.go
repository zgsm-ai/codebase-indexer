package workspace

import (
	"bufio"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var ErrPathNotExists = errors.New("no such file or directory")

type WorkspaceReader struct {
	logger logger.Logger
	Path   string
}

func NewWorkSpaceReader(path string, logger logger.Logger) *WorkspaceReader {
	return &WorkspaceReader{
		Path:   path,
		logger: logger,
	}
}

func (w *WorkspaceReader) FindProjects() []*types.Project {
	if w.logger != nil {
		w.logger.Info("[FindProjects] 开始扫描工作区：%s", w.Path)
	}

	var projects []*types.Project
	maxDepth := 3
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
	if hasGitDir(w.Path) {
		projects = append(projects, &types.Project{
			Path: w.Path,
			Name: filepath.Base(w.Path),
		})
		foundGit = true
	} else {
		// 2. 递归子目录，查找 git 仓库
		var walk func(dir string, depth int)
		walk = func(dir string, depth int) {
			if entryCount >= maxEntries || depth > maxDepth {
				return
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return
			}
			for _, entry := range entries {
				if entryCount >= maxEntries {
					return
				}
				if entry.IsDir() {
					subDir := filepath.Join(dir, entry.Name())
					if hasGitDir(subDir) {
						projects = append(projects, &types.Project{
							Path: subDir,
							Name: filepath.Base(subDir),
						})
						foundGit = true
						// 不递归 .git 仓库下的子目录
						continue
					}
					// 跳过隐藏目录
					if strings.HasPrefix(entry.Name(), ".") {
						continue
					}
					entryCount++
					walk(subDir, depth+1)
				}
			}
		}
		walk(w.Path, 1)
	}

	// 3. 没有发现任何 git 仓库，将当前目录作为唯一项目
	if !foundGit {
		projects = append(projects, &types.Project{
			Path: w.Path,
			Name: filepath.Base(w.Path),
		})
	}

	if w.logger != nil {
		w.logger.Info("[FindProjects] 扫描完成，发现项目数：%d", len(projects))
	}
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
		fileExt := filepath.Ext(relativePath)
		if slices.Contains(walkOpts.ExcludeExts, fileExt) {
			return nil
		}

		if len(walkOpts.IncludeExts) > 0 && !slices.Contains(walkOpts.IncludeExts, fileExt) {
			return nil
		}

		for _, p := range walkOpts.ExcludePrefixes {
			if strings.HasPrefix(relativePath, p) {
				return nil
			}
		}

		for _, p := range walkOpts.IncludePrefixes {
			if !strings.HasPrefix(relativePath, p) {
				return nil
			}
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
