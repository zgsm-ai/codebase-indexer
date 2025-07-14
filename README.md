# codebase-indexer

<div align="center">

[English](./README.md) | [简体中文](./README_zh.md)

A powerful code indexing context retrieval service for AI programming assistants.

[![Go Report Card](https://goreportcard.com/badge/github.com/zgsm-ai/codebase-indexer)](https://goreportcard.com/report/github.com/zgsm-ai/codebase-indexer)
[![Go Reference](https://pkg.go.dev/badge/github.com/zgsm-ai/codebase-indexer.svg)](https://pkg.go.dev/github.com/zgsm-ai/codebase-indexer)
[![License](https://img.shields.io/github/license/zgsm-ai/codebase-indexer)](LICENSE)

</div>

## Overview

codebase-indexer is the context module of [ZGSM (ZhuGe Smart Mind) AI Programming Assistant](https://github.com/zgsm-ai/zgsm) which running on client. It provides powerful codebase indexing capabilities to support  code call graph relationship retrieval for AI programming systems.

### Key Features

- 📊 Code call graph analysis and retrieval
- 🌐 Multi-language support

## Requirements

- Go 1.24.0 or higher

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/zgsm-ai/codebase-indexer.git
cd codebase-indexer

# Install dependencies
go mod tidy
```

### Running

```bash
# Build the project
make build
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).

## Acknowledgments

This project builds upon the excellent work of:

- [Tree-sitter](https://github.com/tree-sitter) - For providing robust parsing capabilities