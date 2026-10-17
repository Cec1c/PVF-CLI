# 来源与外部参考

资料核查日期：2026-10-15。外部项目已查看 README 和部分入口源码，未下载运行，未用当前 protected NKPI 样本做兼容测试。下面的能力描述只覆盖检查到的范围。

## 本地基础

ServerS5A3 / DNF90 的 Go PVF 核心和五个 scratch 命令是首选基础。来源提交、逐文件 SHA256 与路径见 [清单](../reference/source-manifest.json)，依赖关系见 [源码地图](../reference/README.md)。

当前来源根目录未发现 LICENSE、COPYING 或 NOTICE 文件。本仓库没有擅自指定授权协议，也没有复制游戏资产。引入或公开分发来源代码时需确认授权，外部仓库同样单独核对，公开可读不等于任意再授权。

## GitHub 项目

| 项目 | 已观察到的能力 | 对本项目的用途与限制 |
| --- | --- | --- |
| [mechpaul/ScriptPVF](https://github.com/mechpaul/ScriptPVF) | C CLI；`ScriptPVF/Main.c` 提供 `cache / unpack / update / decrypt` | 可以参考命令与解包实现；`update` 是差异分析，不能据名称推断通用写回 |
| [jingjing54007/pvf-unpack](https://github.com/jingjing54007/pvf-unpack) | Go 解析研究；`unpack_CN.go` 涉及 NKPI、段布局和共享池 | 可比较格式实现；`main()` 直接调用 `fooCN()`，不是完善的参数化 CLI |
| [ariakeumi/PvfPlayer](https://github.com/ariakeumi/PvfPlayer) | C++ 台服 PVF；`pvftest/main.cpp` 接受 PVF 路径并读取目录树 | README 提到打包/解包且仍在开发；本次只确认入口读取，不承诺当前格式写回 |
| [Qswhisper/PVF-Ai-Agent-Workbench](https://github.com/Qswhisper/PVF-Ai-Agent-Workbench) | Windows 工作台、BAT CLI、随包 Node.js 与原生后端、只读回退 | 可参考工作流与校验设计；完整 Rust 后端源码/锁文件不在仓库，不能视为完整可复用开源实现；原生后端不可用时只读 |
| [Zageku/DNF_pvf_python](https://github.com/Zageku/DNF_pvf_python) | Python 台服 PVF/GM 等工具 | 偏 GUI 工具集合，非专门 CLI 首选 |

优先复用当前已有格式证据的 Go 核心。外部项目作为格式或交互参考，采用前确认目标格式、许可证、依赖和实测结果。

## PUT 包体积的讨论结论

PUT 的数百 MB 体积不代表 PVF 核心必须如此大。GUI 框架、运行库、编辑器组件和辅助资源都可能增加安装体积，但本次没有拆包归因。不要把这些可能原因写成已核实的 PUT 组成。
