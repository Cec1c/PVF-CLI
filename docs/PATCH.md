# 受控 raw 与 token 编辑

`patch` 和 `verify` 已实现，支持现有 NKPI / protected NKPI 归档中的既有条目替换。schema 1 保持共享字符串池；schema 2 可在 protected NKPI 追加新字符串。两者均不新增路径，不将阅读用文本隐式编译为 token。

```text
pvf patch --pvf baseline.pvf --plan approved.json --out candidate.pvf
pvf verify --pvf baseline.pvf --plan approved.json --out candidate.pvf
```

计划使用 schema 1，完整示意如下（摘要和 Base64 必须用真实值替换）：

```json
{
  "schema": 1,
  "input_sha256": "<64位小写SHA256>",
  "edits": [{
    "path": "itemshop/example.shp",
    "data_type": 1,
    "before_sha256": "<原始raw摘要>",
    "after_sha256": "<新raw摘要>",
    "raw_base64": "<新raw字节的Base64>"
  }]
}
```

路径必须与归档内规范路径相符；原档摘要、条目类型和前后 raw 摘要必须一致。重复目标、未知字段、空计划及无变化的条目会拒绝。计划上限 16 MiB；归档默认上限 512 MiB，可用 `--max-bytes` 调整。

`patch` 拒绝已有输出以及输入别名，先写同目录临时文件、刷盘、独立重新打开，再验证后用硬链接原子安装，拒绝并发创建的目标。目标文件系统须支持硬链接；失败不覆盖输入，临时文件会清理。命令不自动替换玩家文件或发布网络资源。

验证覆盖整个条目集合：身份、类型和分组保持；HASH、名称区和共享字符串池逐字节相等；未触及分组的压缩密文与原始长度相等；受影响分组内逐条比较解压后的 raw，只有计划目标允许改变。JSON 回执包含输入／输出摘要、输出大小、修改条目数和其余条目的验证数量。属于结构与内容保真证明，不是抽样校验；不把库回读等同于原生客户端验收。

type-1 的无损编辑应从 `read --format tokens` 导出的类型／payload 构造 raw。浮点位模式、字符串引用和未知 token 不应经阅读文本转换。CLI 接受显式 raw 或下述 token 计划；新条目写入未接入。

## schema 2 token 计划

沿用上述计划的 `input_sha256`、`path`、`data_type: 1` 和 `before_sha256`，将 `schema` 改为 2；每项选择 `raw_base64` 或 `tokens`。token 模式省略 `raw_base64` 与 `after_sha256`，最终 raw 摘要由确定性编译计算并写入验证回执。

```json
"tokens": [
  {"type": 3, "payload_u32": 12345},
  {"type": 6, "text": "原说明\n新增说明"},
  {"type": 0, "payload_u32": 500000}
]
```

此片段仅说明结构，已有引用必须取自实际 token 导出。每个 token 恰选 `payload_u32` 或 `text`：前者逐位保留原始 payload，后者仅用于字符串类型 3/6，追加 UTF-16 字符串并指向新偏移。禁止 NUL；不替换旧共享字符串。导出 token 的 `text` 是注解，不能原样与 `payload_u32` 一起作为编辑输入。

验证要求旧字符串池完整前缀不变、追加字节恰好匹配计划；所有原条目身份、HASH 和未改 raw 仍保持。回执增加 `appended_strings` 与 `entry_sha256`，便于审查实际修改。未知 token 与浮点数应使用原 payload 位模式。新增路径、任意文本编译和其他格式的字符串追加不在当前支持范围。
