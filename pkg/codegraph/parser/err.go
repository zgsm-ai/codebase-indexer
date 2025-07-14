package parser

import (
	"errors"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

var ErrFileExtNotFound = errors.New("file extension not found")
var ErrLangConfNotFound = errors.New("langConf not found")
var ErrQueryNotFound = errors.New("query not found")

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured def or name node is missing")
	ErrNoDefinition = errors.New("no QueryDefinitions node found")
)

// IsRealQueryErr prevent *sitter.QueryError(nil)
func IsRealQueryErr(err error) bool {
	if err != nil {
		var qe *sitter.QueryError
		if errors.As(err, &qe) && qe == nil {
			return false
		}
		return true
	}
	return false
}

func IsNotSupportedFileError(err error) bool {
	return errors.Is(err, ErrFileExtNotFound) || errors.Is(err, ErrLangConfNotFound)
}
