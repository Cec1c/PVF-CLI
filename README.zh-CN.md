<div align="center">

# PVF-CLI

![状态：受控编辑](https://img.shields.io/static/v1?label=%E7%8A%B6%E6%80%81&message=%E5%8F%97%E6%8E%A7%E7%BC%96%E8%BE%91&color=2563EB&style=flat-square)
![Go](https://img.shields.io/static/v1?label=Go&message=1.26.2&color=00ADD8&style=flat-square&logo=go&logoColor=white)

面向 DNF PVF 的独立命令行工具，提供归档信息、路径列表、文本与 token 读取、搜索、安全导出与受控编辑。

[English](README.md) · [命令说明](docs/CLI.md) · [编辑计划](docs/PATCH.md) · [Agent 入口](AGENTS.md)

</div>

> [!NOTE]
> `info / list / read / search / export / patch / verify` 均已实现。`patch` 按显式计划生成独立候选，`verify` 核对计划与其余条目的完整保持；输入文件不会被覆盖。

## 项目范围

支持 NKPI / protected NKPI。`read` 可输出阅读用文本、原始字节或保留 token 类型与 payload 位模式的 JSON；`search` 可按正文或指定字段筛选，适合核对 `[name]` 中的物品名称。搜索按块扫描，不保留全档文本缓存。

运行不依赖 pvfUtility/PUT、GUI、数据库或游戏服务。导出拒绝已有目标、路径穿越和链接逃逸。其他 PVF 变体需要独立样本验证。

## 快速开始

需要 Go 1.26.2；首次构建需要取得 `golang.org/x/text v0.36.0`。Windows 执行 `BUILD.bat`，测试和构建使用本项目独立缓存，输出 `dist/pvf.exe`。

```text
dist\pvf.exe info --pvf C:\samples\Script.pvf --json
dist\pvf.exe search --pvf C:\samples\Script.pvf --prefix stackable --section name --query 小晶块 --json
dist\pvf.exe read --pvf C:\samples\Script.pvf --path "itemshop/(r)equipmentshop7.shp" --format tokens --out shop.tokens.json
dist\pvf.exe export --pvf C:\samples\Script.pvf --path npc/seria.npc --out extracted
```

`--out` 的文件必须不存在；导出目录可存在，但不会覆盖其中的文件。更多选项、退出码、格式边界见 [命令说明](docs/CLI.md)。

## 受控修改

先读取目标条目的raw或token，按 [编辑计划](docs/PATCH.md) 填写输入及条目摘要，再执行：

```text
dist\pvf.exe patch --pvf baseline.pvf --plan approved.json --out candidate.pvf
dist\pvf.exe verify --pvf baseline.pvf --plan approved.json --out candidate.pvf
```

schema 1支持raw替换；schema 2支持protected NKPI的token编辑及字符串追加。当前CLI不支持新增路径或将阅读用文本直接编译回脚本。

## 开始开发

1. 阅读 [AGENTS.md](AGENTS.md) 和 [开发交接](docs/HANDOFF.md)。
2. 查看 [源码地图](reference/README.md) 和 [逐文件来源清单](reference/source-manifest.json)。
3. 使用仓库中的合成样本验证；真实样本按 `PVFCLI_REAL_PVF_SMOKE` 显式指定，不随仓库分发。
4. 按 [实施计划](docs/IMPLEMENTATION_PLAN.md) 和 [验收标准](docs/VALIDATION.md) 推进。

模块名为 `pvfcli`，核心位于 `internal/pvf`；测试合成样本随源码提供，真实游戏资产放在 ignored 目录。Windows构建已验证；其他目标平台的运行行为需要独立验证。

## 已有基础

来源项目的 PVF 核心实现约 2,100 行，五个 scratch 命令约 1,800 行，不含测试和文档解析包。核心已观察到的唯一第三方导入是 `golang.org/x/text/encoding/simplifiedchinese`，独立 CLI 不需要继承完整服务端依赖。

> [!IMPORTANT]
> 文本解码会缩减浮点数精度，并把部分 token 输出为占位符；编辑依据应使用 raw 或 token JSON。当前打开归档仍会整包载入内存；分块搜索不等于从磁盘流式打开，验证范围见 [验收标准](docs/VALIDATION.md)。

## 参考与来源

[外部项目资料](docs/REFERENCES.md) 记录相关GitHub工具及已核实的能力边界。核心来源为 [ServerS5A3 / DNF90](https://github.com/Cec1c/ServerS5A3)，来源提交及逐文件SHA256见清单。本项目未指定许可证，也不对来源代码授予新的再许可权利。
