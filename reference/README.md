# 源码地图

来源：ServerS5A3 / DNF90。准确的来源 HEAD、文件状态和 SHA256 见 [source-manifest.json](source-manifest.json)。

来源仓库：[ServerS5A3 / DNF90](https://github.com/Cec1c/ServerS5A3)。清单记录提取时的27项源码摘要；核心位于本仓库 `internal/pvf/`，独立CLI位于 `internal/cli/`。仅迁入必要核心，未整体复制源模块。以下为来源相对路径。

## 核心

路径前缀：`go-server/internal/platform/pvf/`。

| 文件 | 用途 |
| --- | --- |
| `pvf.go` | 文件载入、输入规模限制、摘要与原始字节 |
| `archive.go` | 目录、路径索引、归档 API、无缓存扫描入口 |
| `parse.go` | NKPI/protected NKPI 的结构解析 |
| `crypto.go` | 解密与对应布局相关算法 |
| `read.go` | 分块读取、解压缓存与按 data type 解码 |
| `decode.go` | token 到可读文本、共享字符串池、编码与 zlib |
| `path.go` | 归档路径归一化 |
| `patch_string.go` | 指定脚本中的字符串引用替换与块重建 |
| `repack.go` | 同宽池内容替换、现有条目 raw 替换及布局重建 |
| `repack_add.go` | 模板新增脚本、追加池字符串、token 流与 HASH 重建 |
| `repack_verify.go` | 修改之外的条目保持校验 |
| `patch_string_test.go` | 合成样本、字符串操作及条件真实 PVF 测试 |
| `repack_add_test.go` | 依赖指定真实样本的新增/替换/位模式测试 |

## 命令与文档解析依赖

- `go-server/cmd/scratch/pvfdump/main.go`
- `go-server/cmd/scratch/pvftextscan/main.go`
- `go-server/cmd/scratch/pvfpatch/main.go`
- `go-server/cmd/scratch/pvfsetfile/main.go`
- `go-server/cmd/scratch/pvfscriptedit/main.go`
- `go-server/internal/modules/dnf/pvf/*.go`：文档、节、值、列表与索引的解析层；其中索引和测试引用平台 PVF 包。
- `go-server/go.mod`：来源版本证据，保留在快照中供查看；不应整体作为独立工具依赖表。

快照只选取 Go 源码和来源 go.mod，未复制 scratch 中的真实条目 golden/spec、客户端 EXE、PVF、服务器配置或原始私人报告。

## 使用方法

从快照复制需要的实现到本项目正式包，调整模块导入并补充测试。原始快照保持不变，以便比对来源。清单记录的是复制时的实际字节；来源工作区以后有更新，不应无条件同步覆盖新项目。

公开仓库只包含工具源码、文档、来源摘要与合成样本；原始快照和运行资料不随仓库分发。
