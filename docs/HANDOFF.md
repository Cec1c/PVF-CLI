# 开发交接

## 当前实现

独立Go模块 `pvfcli`，Go 1.26.2，外部依赖为 `golang.org/x/text v0.36.0`。入口是 `cmd/pvf`，不依赖游戏服务、数据库或GUI。

- `info / list / read / search / export`：支持NKPI与protected NKPI、raw／文本／token JSON、按字段搜索和受控导出。
- `patch / verify`：schema 1替换既有raw；schema 2编辑protected NKPI token并追加字符串。输入与条目摘要守卫、独立输出、完整保持验证均已接入。
- 合成样本和测试随源码提供；真实PVF、修改计划、运行配置和部署回执不随仓库分发。

命令与边界见 [CLI.md](CLI.md) 和 [PATCH.md](PATCH.md)。来源提交、相对路径和逐文件摘要见 [源码地图](../reference/README.md)。

## 继续开发

先读 [AGENTS.md](../AGENTS.md)，核对工作区状态，再按 [实施计划](IMPLEMENTATION_PLAN.md) 选择任务。Windows使用 `BUILD.bat` 执行测试、vet和构建，其他环境可执行相同Go命令。真实样本测试使用 `PVFCLI_REAL_PVF_SMOKE`；未指定时跳过，不计作通过。

当前打开归档仍整包载入内存。按块扫描搜索不会保留全档解码文本，但不等于磁盘流式打开。磁盘读取、有界缓存及性能测量是后续独立工作；不能用其他项目数据代替本项目实测。

## 保留的边界

阅读文本会缩减浮点精度，部分token显示占位符。编辑必须使用raw或类型／payload位模式；未知值保持原样。CLI尚不支持新增路径、任意文本编译或所有PVF变体。核心内部的模板新增构件不等于CLI已提供该能力。

验证必须区分合成测试、真实样本、全档保真及原生客户端效果，见 [VALIDATION.md](VALIDATION.md)。仓库不携带真实资产，也不自动替换玩家文件或发布网络资源。
