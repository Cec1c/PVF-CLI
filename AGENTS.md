# PVF-CLI 开发入口

这是独立工具项目。当前已实现 info/list/read/search/export 与受控 raw/token patch/verify，统一入口为 `cmd/pvf`；写回规则见 `docs/PATCH.md`。工作范围为独立CLI与PVF核心，不包含游戏服务部署或玩家资源发布。

## 开始顺序

1. 阅读 `docs/HANDOFF.md`、`docs/IMPLEMENTATION_PLAN.md` 和 `docs/VALIDATION.md`。
2. 查看当前分支、`git status --short` 和相关 diff，保留其他会话的工作。
3. 如果存在，读 `local-notes/START_HERE.md`，核对 `reference/source-manifest.json` 与冻结快照。普通克隆缺少本机材料时，按清单定位来源，不假设存在真实 PVF。
4. 当前独立模块为 `pvfcli`，Go 1.26.2，唯一外部依赖 `golang.org/x/text`。先读 `changlog.md`；构建入口 `BUILD.bat`，不照搬游戏服务端完整依赖。

## 边界

- 目标是独立 CLI 与可复用 PVF 核心；运行路径不依赖 PUT/MCP、GUI、MySQL、GM 或游戏服务。
- 来源项目和其候选工作区只用于读取。修改留在本仓库，不回写、重置或清理来源目录。
- 已实现 `info / list / read / search / export / patch / verify`。写入支持既有条目 raw 替换及 schema 2 token 编辑；新字符串只追加并保持旧偏移，新增路径未接入。
- 明确区分归档格式、条目 data type、token type，不把某个后缀直接当作编码。
- 文本导出供阅读；无损编辑以保留 token 类型、payload 位模式或 raw 字节为依据。未知 token 必须保留或明确报错。
- 写操作先生成独立输出，核对输入/输出是否指向同一文件；验证成功后交付。不要自动替换玩家原始 PVF。
- 批量导出需拒绝路径穿越、绝对路径、盘符逃逸和不安全的链接目标。
- 读取不支持的格式或类型要有明确结果，不能用空文本伪装成功。
- 真实样本、解包内容、性能报告、原始证据和本机路径放 ignored 的 `local-notes/`、`runtime/`、`.cache/`。不提交游戏资产、凭据或来源环境配置。
- 构建缓存使用本项目 `.cache/` 下的独立目录；不清理系统或其他会话的 Go cache。构建输出放 `dist/` 或 `bin/`。
- 不启动游戏/登录器/PUT GUI，不连接线上服务，也不为验证 CLI 修改数据库。

## 验证与记录

- 先迁移合成测试，再补充格式边界、坏输入、退出码、JSON 输出、导出路径和写回保真测试。
- 真实 PVF 测试须显式指定样本并记录 SHA256；条件跳过不能算通过。
- 库回读通过与原生客户端接受是不同事实；报告必须区分。
- 性能记录同时包含文件规模、条目数、缓存策略、峰值内存与耗时，不用可执行文件体积推测运行内存。
- 实现开始后同步维护需求状态和变更记录。未跑的检查写明未跑，不继承来源项目的通过结论。
- 文档直接说明结果、依据和下一步，避免重复免责。公共文档使用相对路径及通用示例。

## Git 与来源

公开仓库为 `Cec1c/PVF-CLI`，默认分支 `main`。提交与发布仅包含脱敏源码、文档和合成测试样本；真实资产及运行资料留在忽略目录。
来源根目录未发现 LICENSE/COPYING/NOTICE 文件，保留来源记录，不自行将复制代码标成 MIT 等许可证。原始快照存于 ignored 目录，公开可跟踪资料包含源码地图和摘要清单。
