# PVF CLI 命令

程序读取显式指定的本地文件，不连接服务、不启动 GUI，不修改输入 PVF。当前命令：

| 命令 | 主要参数 | 结果 |
| --- | --- | --- |
| `info` | `--pvf`、`--json` | 格式、字节数、SHA256、条目数、分块数、缓存计数 |
| `list` | `--prefix`、`--query`、`--regex`、`--ignore-case`、`--limit` | 归档路径按字典序排列；query 匹配路径 |
| `read` | `--path`、`--format text/raw/tokens`、`--out`、`--json` | 单条目；大小写不敏感的精确路径 |
| `search` | `--prefix`、`--query`、`--regex`、`--ignore-case`、`--section name`、`--limit` | 正文搜索；结果为匹配文件及行，按归档分块／条目顺序 |
| `export` | `--prefix` 或重复 `--path`；`--format`、`--out` | 保留目录结构；token 文件追加 `.tokens.json` |
| `patch` | `--plan`、`--out` | 按严格 raw/token JSON 替换既有条目，完整保持验证后生成新 PVF；见 [PATCH.md](PATCH.md) |
| `verify` | `--plan`、`--out` | 将已存在候选与原档、计划核对，验证所有条目保持 |

所有命令要求 `--pvf`；`--max-bytes` 默认 536870912。输入按原始字节计算摘要。写文件的 `--out` 单文件模式只创建新文件；父目录必须存在；`verify --out` 则读取已有候选文件。raw 直接写 stdout 时不会混入诊断，Windows 建议用 `--out` 避免 shell 重编码。

list 默认限制 1000 个结果，search 默认 100；`--limit 0` 取消数量上限。截断结果在 JSON 的 `truncated` 和 stderr 中明确报告。export 默认最多 1000 个文件，超限在写入前失败，不截断导出。大规模提取可显式增加上限。

搜索只解释 data type 1／3，其他类型数量通过 `skipped_unsupported` 报告；结构错误中止扫描。其他操作遇到无法解释的文本类型则返回错误；需要原始内容时选择 raw。`--section` 对解码文本中的首个完整字段标签取值，适合 `[name]` 等单值字段，不是任意嵌套脚本查询语言。

`read` 按归档实际路径定位，不自动替换版本变体。例如列表引用 `EquipmentShop7.shp` 而档内仅存在 `(r)equipmentshop7.shp` 时，应先 `list`，再明确读取实际变体，避免误读 `(f)` 版本。

## JSON 与退出码

列表／搜索／导出的 envelope 为 `schema: 1`、`results`、`truncated`。text read 的 JSON 包含 `path`、`data_type`、`text`；info 使用归档 snapshot 字段。stdout 放结果，stderr 放诊断。

| 退出码 | 含义 |
| --- | --- |
| 0 | 成功；list/search 截断需另外检查 `truncated` |
| 1 | 无效归档、编码、路径或 I/O 错误 |
| 2 | 参数错误 |
| 3 | 没有匹配条目 |
| 130 | 已取消 |

export 会先检查选择范围、目标类型和输出冲突；写入过程失败时报告已完成文件数，输出目录可能保留部分新文件。输入 PVF 始终只读。

## token JSON

仅 data type 1 支持 token。每个 token 保存 `type` 和 `payload_u32`，字符串类型额外提供 `text` 注解；`tail_hex` 保存末尾不足 5 字节的数据，`raw_sha256` 固定原始内容。JSON 中的字符串注解不参与重建，float32 按原始位模式保存，未知 token 类型也保留。

核心 `TokenDocument.Raw()` 可验证 JSON 往返保持字节。CLI使用独立的schema 2编辑计划，不直接导入token导出文档；不可把修改 `text` 注解当作修改脚本。阅读文本仍会把 float 四舍五入至两位，部分 token 显示占位符。

## 后续写回

`patch/verify` 已接入输入与条目摘要守卫、独立输出、临时文件回读和全量保持验证。schema 1 用 raw，schema 2 另支持 protected NKPI token 编辑和追加新字符串，详见 [PATCH.md](PATCH.md)。命令不自动替换玩家文件或发布。
