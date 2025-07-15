package resolver

//
//// Rust解析器
//type RustResolver struct {
//}
//
//func (r *RustResolver) Resolve(importStmt *Import, currentFilePath string, config *ProjectInfo) error {
//	if importStmt.Name == types.EmptyString {
//		return fmt.Errorf("import is empty")
//	}
//
//	importStmt.FilePaths = []string{}
//	importName := importStmt.Name
//
//	// 处理crate根路径
//	if strings.HasPrefix(importName, "crate::") {
//		importName = strings.TrimPrefix(importName, "crate::")
//	}
//
//	// 将::转换为路径分隔符
//	modulePath := strings.ReplaceAll(importName, "::", "/")
//
//	if len(config.fileSet) == 0 {
//		//TODO log
//		fmt.Println("not support project file list, use default resolve")
//		importStmt.FilePaths = []string{modulePath}
//		return nil
//	}
//
//	foundPaths := []string{}
//
//	// 尝试查找.rs文件或模块目录
//	for _, relDir := range config.dirs {
//		relPath := ToUnixPath(filepath.Join(relDir, modulePath+".rs"))
//		if containsFileIndex(config, relPath) {
//			foundPaths = append(foundPaths, relPath)
//		}
//		modPath := ToUnixPath(filepath.Join(relDir, modulePath, "mod.rs"))
//		if containsFileIndex(config, modPath) {
//			foundPaths = append(foundPaths, modPath)
//		}
//	}
//
//	importStmt.FilePaths = foundPaths
//	if len(importStmt.FilePaths) > 0 {
//		return nil
//	}
//
//	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
//}
