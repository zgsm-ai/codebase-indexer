package codegraph

import "codebase-indexer/pkg/logger"

// 定时扫描

type IndexChecker struct {
	logger  logger.Logger
	indexer *Indexer
}

func NewIndexerChecker(logger logger.Logger, indexer *Indexer) *IndexChecker {
	return &IndexChecker{
		logger:  logger,
		indexer: indexer,
	}
}

// CheckPeriodically 周期性检查索引和本地文件
func (i *IndexChecker) CheckPeriodically() {

}
