package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"
	"github.com/stretchr/testify/assert"
)

var tempDir = "/tmp/"

var visitPattern = types.VisitPattern{ExcludeDirs: []string{".git", ".idea"}, IncludeExts: []string{".go"}}

// TODO 性能（内存、cpu）监控；各种路径、项目名（中文、符号）测试；索引数量统计；大仓库测试

func cleanIndexStoreTestHelper(ctx context.Context, projects []*workspace.Project, storage store.GraphStorage) error {
	for _, p := range projects {
		if err := storage.DeleteAll(ctx, p.Uuid); err != nil {
			return err
		}
		if storage.Size(ctx, p.Uuid) > 0 {
			return fmt.Errorf("clean workspace index failed, size not equal 0")
		}
	}
	return nil
}

func TestIndexer_IndexWorkspace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()

	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)

	// 测试 IndexWorkspace
	err = indexer.IndexWorkspace(context.Background(), workspaceDir)
	assert.NoError(t, err)
	for _, p := range projects {
		t.Logf("=> storage size: %d", storage.Size(ctx, p.Uuid))
	}
	// 统计项目的go文件数量，确保索引数和文件数一致。
	var goCount int
	err = workspaceReader.Walk(ctx, workspaceDir, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	assert.NoError(t, err)

	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if strings.HasPrefix(key, store.PathKeyPrefix) {
				indexSize++
			}
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, goCount, indexSize)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_IndexProjectFilesWhenProjectHasIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	newVisitPattern := types.VisitPattern{ExcludeDirs: []string{".git", ".idea", "mocks"}, IncludeExts: []string{".go"}}
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: newVisitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, newVisitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 先索引工作区，然后再索引文件
	err = indexer.IndexWorkspace(ctx, workspaceDir)
	assert.NoError(t, err)
	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	pathKeys := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
	}
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !strings.HasPrefix(key, store.PathKeyPrefix) {
				continue
			}
			_, ok := pathKeys[key]
			assert.False(t, ok)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}

	// 测试 index files

	assert.NoError(t, err)
	filesByProject, err := indexer.groupFilesByProject(projects, files)
	assert.NoError(t, err)
	for k, v := range filesByProject {
		err = indexer.indexProjectFiles(context.Background(), k, v)
		assert.NoError(t, err)
	}

	assert.True(t, len(pathKeys) > 0)
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !strings.HasPrefix(key, store.PathKeyPrefix) {
				continue
			}
			t.Logf("key: %s", key)
			delete(pathKeys, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_IndexProjectFilesWhenProjectHasNoIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid)
		assert.Equal(t, size, 0)
	}
	// 测试 index files
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)

	// 统计项目的go文件数量，确保索引数和文件数一致。
	var goCount int
	err = workspaceReader.Walk(ctx, workspaceDir, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	assert.NoError(t, err)

	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if strings.HasPrefix(key, store.PathKeyPrefix) {
				indexSize++
			}
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, goCount, indexSize)

	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_RemoveIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid)
		assert.Equal(t, size, 0)
	}
	// 测试 index files
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)
	pathKeys := make(map[string]any)
	pathKeysBak := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
		pathKeysBak[key] = nil
	}
	assert.True(t, len(pathKeys) > 0)
	assert.True(t, len(pathKeysBak) > 0)

	totalIndexBefore := 0
	// 测试存在
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			totalIndexBefore++
			key := iter.Key()
			delete(pathKeys, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0)
	// 统计项目的go文件数量，确保索引数和文件数一致。
	err = indexer.RemoveIndexes(ctx, workspaceDir, files)
	assert.NoError(t, err)
	totalIndexAfter := 0
	// 测试不存在
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			totalIndexAfter++
			key := iter.Key()
			delete(pathKeysBak, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, len(files), len(pathKeysBak))
	assert.True(t, totalIndexBefore-len(files) <= totalIndexAfter)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_QueryElements(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
	err = indexer.IndexWorkspace(ctx, workspaceDir)
	assert.NoError(t, err)
	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	pathKeys := make(map[string]any)
	for _, f := range files {
		pathKeys[f] = nil
	}
	assert.True(t, len(pathKeys) > 0)

	// 统计项目的go文件数量，确保索引数和文件数一致。
	fileTables, err := indexer.QueryElements(ctx, workspaceDir, files)
	assert.NoError(t, err)
	assert.Equal(t, len(files), len(fileTables))
	for _, f := range fileTables {
		assert.Contains(t, pathKeys, f.Path)
		assert.True(t, len(f.Elements) > 0)
		assert.Equal(t, f.Language, string(lang.Go))
		for _, e := range f.Elements {
			assert.Equal(t, len(e.Range), 4)
			assert.True(t, e.Name != types.EmptyString)
			assert.True(t, e.Path != types.EmptyString)
			assert.Equal(t, e.Path, f.Path)
		}
	}

	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

}

func TestIndexer_QuerySymbols_WithExistFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
	err = indexer.IndexWorkspace(ctx, workspaceDir)
	assert.NoError(t, err)
	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks", "mock_graph_store.go")
	symbolNames := []string{"MockGraphStorage", "BatchSave"}
	// 统计项目的go文件数量，确保索引数和文件数一致。
	symbols, err := indexer.QuerySymbols(ctx, workspaceDir, filePath, symbolNames)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(symbols))
	for _, s := range symbols {
		assert.True(t, slices.Contains(symbolNames, s.Name))
		assert.Equal(t, s.Language, string(lang.Go))
		assert.True(t, len(s.Definitions) > 0)
		for _, d := range s.Definitions {
			assert.Equal(t, len(d.Range), 4)
			assert.True(t, d.Path == filePath)
		}
	}

	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_IndexWorkspace_NotExists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 不存在的项目
	workspaceDir, err := filepath.Abs("/not/exist")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)
	err = indexer.IndexWorkspace(ctx, workspaceDir)
	assert.ErrorContains(t, err, "not exists")
}

func TestIndexer_IndexFiles_NoProject(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 不存在的项目
	workspaceDir, err := filepath.Abs("/not/exist")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)
	err = indexer.IndexFiles(ctx, workspaceDir, []string{"test.go"})
	assert.ErrorContains(t, err, "not exists")
}
