# CLIProxyAPI fork 差异说明

本仓库是 `router-for-me/CLIProxyAPI` 的 fork。除下列差异外，其余功能、使用方式和配置项以 upstream 为准。
本仓库需要搭配[前端面板](https://github.com/Fwindy/Cli-Proxy-API-Management-Center)使用

## 与 upstream 的主要差异

### 1. Usage 统计恢复并重构为 SQLite 持久化

本 fork 保留并重构了 built-in usage 统计能力：

- usage 原始请求记录持久化到 SQLite，默认数据库文件为日志目录下的 `usage.db`，不再保留运行期内存聚合统计对象。
- `usage-statistics-enabled` 仍用于控制是否记录 usage。
- Management API 返回精简明细结构，支持时间范围查询和按 `id` 删除；usage import/export 接口已删除。

### 2. Usage 明细字段扩展

每条 usage detail 额外记录：

- `first_byte_latency_ms`：保留字段名，值来自 upstream `TTFT`（上游响应首 token/首字节耗时）。
- `generation_ms`：保留字段名，按 `latency_ms - first_byte_latency_ms` 计算。
- `thinking_effort`：保留字段名，值来自 upstream `ReasoningEffort`（最终发给上游 provider 的 reasoning effort）。

### 3. Fork 自动同步相关 workflow

本 fork 调整了 GitHub workflow，用于跟踪 upstream 更新和标签同步；不完全沿用 upstream 的 workflow 配置。

## 友链

[![友链 linux.do](https://img.shields.io/badge/LINUX--DO-Community-blue.svg)](https://linux.do/)

## License

MIT
