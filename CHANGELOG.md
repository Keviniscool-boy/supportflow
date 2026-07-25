# 变更日志

SupportFlow 的重要变更都会记录在此文件中。

本文档格式参考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，项目版本遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html)。

## [未发布]

### 新增

- 建立 Go 1.24+ 后端模块、配置校验、HTTP 基础中间件和健康检查。
- 增加独立 PostgreSQL Migration Runner 与首个扩展 Migration。
- 增加面向 Lite 服务器的 PostgreSQL、Redis、Migration 和 App Compose 配置。
- 完成 Lite 容器启动、健康检查、扩展安装和重复 Migration 幂等验证。
- 完成 v0.2 核心 PostgreSQL Schema、NovaTech Demo Seed 和 pgvector Mock 知识索引数据。
- 完成 Demo Session、HttpOnly Cookie、CSRF 校验、过期/撤销和 CORS 基础边界。
- 增加后端单元测试、静态检查和 CI 验证入口。
- 产品需求文档、技术架构、数据库设计和 API 设计。
- 开源仓库协作、安全与社区治理基线。

### 变更

- 仓库默认介绍、文档索引和协作入口切换为简体中文。
