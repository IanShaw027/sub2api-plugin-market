# Sub2API Plugin Market - API 实现文档（历史归档）

> 归档状态：该文档为历史实现记录，已不再作为当前交付基线。  
> 最后同步日期：2026-03-05

## 文档定位

本文件用于保留早期 API 分层设计与落地路径，便于追溯实现演进。  
当前项目状态请以以下文档为准：

- `README.md`（项目总览与启动方式）
- `docs/API.md`（公开 API 行为说明）
- `docs/ADMIN_GUIDE.md`（管理端接口与认证流程）
- `openapi/plugin-market-v1.yaml`（机器可校验契约）
- `docs/ERROR-CODE-REGISTRY.md`（错误码注册表）

## 历史实现要点（保留）

- 架构：`Repository -> Service -> Handler`
- 公开接口：插件列表、插件详情、版本列表、下载、信任密钥查询
- 响应模型：统一 `Envelope`/错误码模型
- 下载策略：`GET /api/v1/plugins/:name/versions/:version/download` 返回 `302` 跳转预签名 URL

## 已完成并纳入主文档的事项

- 存储接入与预签名下载流程
- 签名校验链路（下载前校验）
- 下载日志记录与下载计数更新
- 错误码到 HTTP 状态码映射标准化
- OpenAPI 契约补齐与脚本化校验

## 说明

如果本文件与 `docs/` 或 `openapi/` 下的文档有冲突，以后者为准。  
新需求与变更请直接更新当前文档，不再在本归档中追加 TODO。
