# Rosetta — 企业数据治理平台设计方案

> **命名由来**：Rosetta Stone（罗塞塔石碑）— 同一段文字用三种语言刻写，让后世得以破译古埃及象形文字。本平台同样承载"用一种逻辑定义，翻译为多种数据库方言"的核心使命。

---

## 目录

1. [核心定位与哲学](#1-核心定位与哲学)
2. [系统架构](#2-系统架构)
3. [用户权限管理（RBAC）](#3-用户权限管理rbac)
4. [功能模块设计](#4-功能模块设计)
   - [4.1 字典维护](#41-字典维护)
   - [4.2 表结构编辑器](#42-表结构编辑器)
   - [4.3 表结构可视化](#43-表结构可视化)
   - [4.4 DDL 多方言渲染](#44-ddl-多方言渲染)
   - [4.5 数据源实例管理](#45-数据源实例管理)
   - [4.6 反向工程（存量表纳管）](#46-反向工程存量表纳管)
5. [数据库模型](#5-数据库模型)
6. [API 设计](#6-api-设计)
7. [技术选型](#7-技术选型)
8. [交付计划](#8-交付计划)

---

## 1. 核心定位与哲学

### 1.1 平台定位

**Rosetta 是 DDL 的单一来源（Source of Truth）**。所有数据表结构的定义、变更、部署以 Rosetta 为准，数据库只是执行平台指令的终端。

```
旧模式：数据库有什么 → 平台展示什么（被动观测）
新模式：平台定义什么 → 数据库部署什么（主动管控）
```

### 1.2 小而美原则

| 原则 | 说明 |
|------|------|
| **单体优先** | 一个后端服务跑所有模块，不做微服务拆分 |
| **先有后好** | 功能先跑通再优化，不做"一步到位" |
| **能买不造** | DataX 采集、Quartz 调度、ES 搜索，不重复造轮子 |
| **人工+自动混合** | 关键信息（业务标签、注释）允许人工维护，自动采集兜底 |
| **砍掉大而全** | 不做 AI 推荐、自然语言搜索、动态脱敏、行级权限等 P2 功能 |

### 1.3 目标用户

| 角色 | 职责 | 核心使用场景 |
|------|------|-------------|
| **数据开发** | 设计表结构、生成 DDL、部署到实例 | 表结构编辑器、DDL 预览 |
| **数据分析 / BI** | 查找数据、理解字段含义 | 数据目录、单表结构图、ER 图 |
| **数据治理管理员** | 维护字典、管理权限、审批变更 | 字典维护、RBAC 配置 |
| **DBA** | 审核 DDL、管理实例 | 实例管理、DDL 审批、反向工程 |

---

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端 (React + Ant Design)                 │
│   字典维护  │  表结构编辑器  │  表结构可视化  │  RBAC 管理        │
├─────────────────────────────────────────────────────────────────┤
│                        后端 (Spring Boot 单体)                    │
│                                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ 字典服务  │ │ 模型服务  │ │ DDL渲染  │ │ 权限服务  │          │
│  │ DictSvc  │ │ ModelSvc │ │ DdlSvc   │ │ AuthSvc  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ 实例管理  │ │ 反向工程  │ │ 可视化   │ │ 部署执行  │          │
│  │ InstSvc  │ │ RevEngSvc│ │ VizSvc   │ │ DeploySvc│          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                                                 │
│  DDL 方言渲染引擎（策略模式）                                     │
│  ┌─────────────┐  ┌─────────────┐                               │
│  │ MySQL 渲染器 │  │ GaussDB M   │  ← 可扩展新方言               │
│  └─────────────┘  └─────────────┘                               │
├─────────────────────────────────────────────────────────────────┤
│                        存储层                                    │
│              MySQL（平台自身元数据 + 用户权限）                    │
└─────────────────────────────────────────────────────────────────┘
```

### 2.1 核心数据流

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  逻辑模型 IR  │ ──→ │  DDL 渲染器  │ ──→ │  DDL 文本    │ ──→ │  目标数据库  │
│ (方言无关)   │     │ (方言映射)   │     │ (方言相关)   │     │ (MySQL/GaussDB)│
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       ↑                                                          │
       │                                     ┌────────────────────┘
       │                                     ↓
┌─────────────┐                       ┌─────────────┐
│  反向工程    │ ←──────────────────── │  采集元数据   │
│ (JDBC采集)  │                       │ (一致性校验)  │
└─────────────┘                       └─────────────┘
```

---

## 3. 用户权限管理（RBAC）

### 3.1 模型概述

采用 RBAC（Role-Based Access Control）模型，核心实体关系：

```
用户 (User) ──N:M──→ 角色 (Role) ──N:M──→ 权限 (Permission)
                           │
                           ↓
                    角色-模型绑定 (RoleModelBinding)
                        · 可读 (READ)
                        · 可编辑 (WRITE)
```

### 3.2 角色定义

| 角色 | 编码 | 说明 |
|------|------|------|
| **超级管理员** | `SUPER_ADMIN` | 系统级权限，管理用户、角色、实例；所有模型可读写 |
| **数据治理管理员** | `GOVERNANCE_ADMIN` | 管理字典、数据标准；所有模型可读 |
| **数据开发** | `DATA_DEVELOPER` | 在自己被授权的模型上可读写（CREATE TABLE / ALTER TABLE） |
| **数据分析 / BI** | `DATA_ANALYST` | 在自己被授权的模型上只读（浏览表结构、查看 ER 图） |

### 3.3 权限粒度

权限控制粒度到达"逻辑模型级"（即表级）：

| 权限 | 编码 | 说明 |
|------|------|------|
| **可读** | `READ` | 查看表结构、字段信息、ER 图、DDL 预览 |
| **可编辑** | `WRITE` | 编辑表结构（字段/约束/索引）、生成 DDL、部署到实例 |

**权限判定规则**：
1. `SUPER_ADMIN` 拥有所有模型的所有权限，无需显式绑定
2. `GOVERNANCE_ADMIN` 拥有所有模型的 `READ` 权限
3. `DATA_DEVELOPER` 和 `DATA_ANALYST` 的权限由角色-模型绑定决定
4. 用户拥有多个角色时，取权限的**并集**（最高权限生效）

### 3.4 UI 交互设计

#### 用户管理页

```
┌─ 用户列表 ───────────────────────────────────────────────┐
│ [+ 新建用户]                                              │
│                                                            │
│ ┌────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ 用户名  │ 显示名    │ 所属角色  │ 状态     │ 操作        │ │
│ ├────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ zhangsan│ 张三    │ 数据开发  │ 启用     │ ✏️ 🗑️ 🔑  │ │
│ │ lisi   │ 李四    │ 数据分析  │ 启用     │ ✏️ 🗑️ 🔑  │ │
│ │ wangwu │ 王五    │ 超级管理员 │ 启用     │ ✏️ 🗑️ 🔑  │ │
│ └────────┴──────────┴──────────┴──────────┴────────────┘ │
└────────────────────────────────────────────────────────────┘
```

#### 角色-模型授权页（核心权限管理界面）

```
┌─ 角色选择 ───────────────────────────┬─ 模型授权矩阵 ──────────────────┐
│                                       │                                │
│  角色: [数据开发 ▼]                   │  Schema: [全部 ▼]  [搜索模型]  │
│                                       │                                │
│                                       │  ┌─────────────┬──────┬──────┐ │
│                                       │  │ 模型/表名    │ 可读  │ 可编辑│ │
│                                       │  ├─────────────┼──────┼──────┤ │
│                                       │  │ user_order  │  ☑   │  ☑   │ │
│                                       │  │ user_info   │  ☑   │  ☐   │ │
│                                       │  │ order_item  │  ☑   │  ☑   │ │
│                                       │  │ product     │  ☐   │  ☐   │ │
│                                       │  └─────────────┴──────┴──────┘ │
│                                       │                                │
│                                       │ [批量授权] [保存]               │
└───────────────────────────────────────┴────────────────────────────────┘
```

---

## 4. 功能模块设计

### 4.1 字典维护

#### 4.1.1 三类字典

| 字典类型 | 编码 | 用途 | 示例 |
|----------|------|------|------|
| **标准字典** | `STANDARD` | 统一业务术语，字段值域约束 | 用户状态：启用/禁用/注销 |
| **类型映射表** | `TYPE_MAPPING` | 逻辑类型 → 各方言物理类型 | `BIGINT` → MySQL:`BIGINT`, GaussDB:`BIGINT` |
| **参考数据** | `REFERENCE` | 系统级枚举、数据层级定义 | 性别、订单状态、数据层级(ODS/DWD/DWS/ADS) |

#### 4.1.2 UI 布局

```
┌─ 字典左侧树 ──────────────────────┬─ 右侧编辑区 ───────────────┐
│                                   │                            │
│  📁 标准字典                      │  字典名称: 用户状态        │
│    📄 用户状态                    │  字典编码: USER_STATUS     │
│    📄 订单状态                    │                            │
│    📄 审批状态                    │  ┌────┬──────────┬──────┐ │
│  📁 类型映射                      │  │编码 │ 名称     │ 操作  │ │
│    📄 MySQL 类型映射              │  ├────┼──────────┼──────┤ │
│    📄 GaussDB 类型映射            │  │ 1  │ 启用     │ ✏️🗑️ │ │
│  📁 参考数据                      │  │ 0  │ 禁用     │ ✏️🗑️ │ │
│    📄 性别                        │  │ 2  │ 注销     │ ✏️🗑️ │ │
│    📄 是否删除                    │  └────┴──────────┴──────┘ │
│                                   │                            │
│  [+ 新建字典]                     │  [+ 添加条目] [保存]       │
└───────────────────────────────────┴────────────────────────────┘
```

#### 4.1.3 类型映射表设计

类型映射表是 DDL 渲染引擎的数据基础。每条记录定义"一个逻辑类型在某个方言中对应的物理类型"：

| 逻辑类型 | 方言 | 物理类型 | 默认长度 | 备注 |
|----------|------|----------|----------|------|
| BIGINT | MYSQL | BIGINT | - | |
| BIGINT | GAUSSDB_M | BIGINT | - | |
| VARCHAR | MYSQL | VARCHAR | 255 | |
| VARCHAR | GAUSSDB_M | VARCHAR | 255 | |
| DECIMAL | MYSQL | DECIMAL | 18,2 | |
| DECIMAL | GAUSSDB_M | DECIMAL | 18,2 | |
| DATETIME | MYSQL | DATETIME | - | |
| DATETIME | GAUSSDB_M | TIMESTAMP | - | GaussDB 无 DATETIME，用 TIMESTAMP |
| TEXT | MYSQL | TEXT | - | |
| TEXT | GAUSSDB_M | CLOB | - | GaussDB 用 CLOB 替代 TEXT |
| BOOLEAN | MYSQL | TINYINT(1) | - | MySQL 无原生 BOOLEAN |
| BOOLEAN | GAUSSDB_M | BOOLEAN | - | |

**扩展性**：新增数据库方言时，只需在此表中添加对应的逻辑类型映射记录，无需修改代码。

---

### 4.2 表结构编辑器

#### 4.2.1 整体布局

```
┌─ Schema 树 ──────────────────────┬─ 表编辑主区域 ─────────────────────────┐
│                                   │                                        │
│  📁 数据源实例                     │ 表名: user_order          数据层级: ods │
│   📁 mysql-a                     │ 方言: MySQL                            │
│    📁 ods                        │                                        │
│     📄 user_order    ◄ active   │ ┌─ 字段 ─┬─ 约束 ─┬─ 索引 ─┬─ DDL 预览 ┐│
│     📄 user_info                 │ │                                    │ │
│    📁 dwd                        │ │ ┌────┬────────┬────┬────┬────┬──┐ │ │
│     📄 dwd_user_order            │ │ │序号│字段名   │类型 │非空│注释│操作│ │ │
│   📁 gaussdb-b                   │ │ ├────┼────────┼────┼────┼────┼──┤ │ │
│    📁 dws                        │ │ │ 1  │id      │BIG…│ ☑  │主键│✏️🗑️│ │ │
│     📄 dws_sales_summary         │ │ │ 2  │user_id │BIG…│ ☑  │用… │✏️🗑️│ │ │
│                                   │ │ │ 3  │amount  │DEC…│ ☑  │金… │✏️🗑️│ │ │
│                                   │ │ │ 4  │status  │VAR…│ ☑  │状… │✏️🗑️│ │ │
│  [+ 新建表]  [+ 反向工程]         │ │ └────┴────────┴────┴────┴────┴──┘ │ │
│                                   │ │ [+ 添加字段]                       │ │
│                                   │ └────────────────────────────────────┘ │
│                                   │                                        │
│                                   │ [保存] [生成 DDL] [部署到实例]          │
└───────────────────────────────────┴────────────────────────────────────────┘
```

#### 4.2.2 四个 Tab 页详情

**Tab 1 — 字段定义**

可编辑表格，内联编辑模式：

- **字段名**：文本输入，自动转小写加下划线格式（snake_case），实时校验重名
- **逻辑类型**：下拉选择，选项来自类型映射表中定义的逻辑类型集合（BIGINT、VARCHAR、DECIMAL、DATETIME、TEXT、BOOLEAN、INT、FLOAT、DATE 等）
- **长度/精度**：选择 VARCHAR 或 DECIMAL 时展开输入框（如 `VARCHAR(32)` 中的 `32`）
- **是否非空**：Checkbox
- **默认值**：文本输入
- **注释**：文本输入
- **操作**：拖拽排序、删除

**Tab 2 — 约束配置**

```
┌─ 主键 ─────────────────┐  ┌─ 外键 ─────────────────────────┐
│                        │  │                                │
│ 主键列: [id ▼]         │  │ ┌────────┬──────────┬────────┐│
│ [+ 添加联合主键列]      │  │ │外键列   │引用表     │引用列  ││
│                        │  │ ├────────┼──────────┼────────┤│
│                        │  │ │user_id │user_info │id      ││
│                        │  │ └────────┴──────────┴────────┘│
│                        │  │ [+ 添加外键]                   │
└────────────────────────┘  └────────────────────────────────┘
```

**Tab 3 — 索引配置**

```
┌─── 索引列表 ────────┬─── 索引列配置 ───────────┐
│                     │                          │
│ ┌──────────┬──────┐ │  索引名称: unq_phone     │
│ │ 名称      │ 类型  │ │  索引类型: 唯一         │
│ ├──────────┼──────┤ │                          │
│ │idx_user  │普通  │ │  列配置:                 │
│ │unq_phone │唯一  │ │  ┌──────────┬──────┐    │
│ │          │      │ │  │ 列名      │ 排序  │    │
│ └──────────┴──────┘ │  ├──────────┼──────┤    │
│                     │  │ phone    │ ASC  │    │
│                     │  └──────────┴──────┘    │
│                     │  [+ 添加列]              │
│                     │                          │
│ [+ 新建索引]         │  [保存] [删除]           │
└─────────────────────┴──────────────────────────┘
```

**Tab 4 — DDL 预览**

实时渲染，切换目标方言下拉框，DDL 即时变化：

```
目标方言: [MySQL ▼]                     [复制 SQL]

CREATE TABLE `user_order` (
  `id`         BIGINT        NOT NULL  COMMENT '主键',
  `user_id`    BIGINT        NOT NULL  COMMENT '用户ID',
  `amount`     DECIMAL(18,2) NOT NULL  COMMENT '订单金额',
  `status`     VARCHAR(32)   NOT NULL  COMMENT '订单状态',
  PRIMARY KEY (`id`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户订单表';
```

---

### 4.3 表结构可视化

#### 4.3.1 ER 图（`#/diagram`）

使用 **Cytoscape.js**（本地 vendored，664KB）渲染 Schema 级 ER 图：

- **自动布局**：`breadthfirst` 算法，有向图分层排列，根表在左，被引用表在右
- **表节点**：圆角矩形卡片，显示表名、列数、字段列表（字段名+类型+PK/FK 标记）
- **关系连线**：贝塞尔曲线 + 箭头 + 列名标签，自引用边自动跳过
- **缩放平移**：滚轮缩放 + 拖拽平移，Cytoscape 原生支持
- **点击交互**：浏览模式下点击表节点跳转到模型详情页
- **编辑模式**：支持 FK 模式下点击两个表创建外键关系
- **高度计算**：根据字段数量动态计算节点高度（替代已废弃的 `height: 'label'`）

#### 4.3.2 结构图 API

后端提供可视化数据 API：

| 端点 | 说明 |
|------|------|
| `GET /api/models/:id/structure-diagram` | 单表结构数据（字段+PK/FK标记+索引+外键） |
| `GET /api/schemas/:id/er-diagram` | Schema ER 图数据（tables + edges） |

#### 4.3.3 数据库文档汇编页（`#/docs`）

以数据字典形式展示所有模型的完整参考文档：
- 表名、备注、状态
- 字段列表（序号、字段名、类型、非空、主键、默认值、备注）
- 索引和外键详情
- 可折叠的 DDL 预览
- 支持浏览器打印

---

### 4.4 DDL 多方言渲染

#### 4.4.1 架构

采用策略模式，每种数据库方言一个渲染器实现：

```
                    ┌──────────────────────┐
                    │    DdlRenderer        │  ← 接口
                    │  + render(IR): String │
                    └──────┬───────────────┘
                           │
           ┌───────────────┼───────────────┐
           ↓               ↓               ↓
   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
   │ MySqlRenderer│ │GaussDbRenderer│ │  FutureXxx   │
   └──────────────┘ └──────────────┘ └──────────────┘
```

#### 4.4.2 中间表示（IR）

逻辑模型在平台内部以结构化 JSON 存储，与任何方言无关：

```json
{
  "tableName": "user_order",
  "comment": "用户订单表",
  "columns": [
    {
      "name": "id",
      "logicalType": "BIGINT",
      "nullable": false,
      "comment": "主键ID",
      "primaryKey": true
    },
    {
      "name": "user_id",
      "logicalType": "BIGINT",
      "nullable": false,
      "comment": "用户ID"
    },
    {
      "name": "amount",
      "logicalType": "DECIMAL",
      "length": 18,
      "scale": 2,
      "nullable": false,
      "comment": "订单金额"
    },
    {
      "name": "status",
      "logicalType": "VARCHAR",
      "length": 32,
      "nullable": false,
      "comment": "订单状态"
    }
  ],
  "indexes": [
    { "name": "idx_user_id", "type": "NORMAL", "columns": [{"name": "user_id", "order": "ASC"}] }
  ],
  "foreignKeys": [
    { "name": "fk_order_user", "columnName": "user_id", "refTable": "user_info", "refColumn": "id" }
  ]
}
```

#### 4.4.3 方言差异处理

| 特性 | MySQL | GaussDB M |
|------|-------|-----------|
| 自增列 | `AUTO_INCREMENT` | `GENERATED BY DEFAULT AS IDENTITY` |
| 存储引擎 | `ENGINE=InnoDB` | 不支持，渲染时跳过 |
| 字符集 | `DEFAULT CHARSET=utf8mb4` | `CHARACTER SET UTF8` |
| 建表尾缀 | `COMMENT='表注释'` | `COMMENT ON TABLE xxx IS '表注释'`（独立语句） |
| DATETIME 类型 | `DATETIME` | `TIMESTAMP` |
| TEXT 类型 | `TEXT` | `CLOB` |
| BOOLEAN 类型 | `TINYINT(1)` | `BOOLEAN` |
| 索引 | `INDEX idx_name (col)` | `CREATE INDEX idx_name ON tbl (col)`（建表外独立语句） |
| 列注释 | 内联 `COMMENT 'xxx'` | `COMMENT ON COLUMN tbl.col IS 'xxx'`（独立语句） |

---

### 4.5 数据源实例管理

#### 4.5.1 概念模型

```
数据源实例 (Instance)
  ├── 名称: mysql-a-prod
  ├── 类型: MYSQL
  ├── 连接: 10.0.1.100:3306
  ├── 凭证: ****（加密存储）
  └── Schema 列表
        ├── ods
        ├── dwd
        └── dws
```

实例下可创建多个 Schema。Schema 代表一个数据库实例内的一个库/模式。

#### 4.5.2 UI

```
┌─ 实例列表 ──────────────────────────────────────────────────┐
│ [+ 注册实例]  [测试连接]                                      │
│                                                              │
│ ┌──────────┬──────────┬────────────────┬──────┬────────────┐│
│ │ 名称      │ 类型      │ 连接地址        │ 状态  │ 操作       ││
│ ├──────────┼──────────┼────────────────┼──────┼────────────┤│
│ │ mysql-a  │ MySQL    │ 10.0.1.100:3306│ 🟢正常│ 编辑 删除  ││
│ │ gaussdb-b│ GaussDB  │ 10.0.2.50:5432 │ 🟢正常│ 编辑 删除  ││
│ └──────────┴──────────┴────────────────┴──────┴────────────┘│
└──────────────────────────────────────────────────────────────┘
```

实例详情页管理 Schema 列表、测试连通性、查看已部署的模型列表。

---

### 4.6 反向工程（存量表纳管）

已存在的存量表，通过 JDBC 连接自动采集表结构，反向填充为逻辑模型 IR，纳入平台管控。

#### 流程

```
1. 选择数据源实例 → 2. 选择 Schema → 3. 选择表 →
4. 平台自动解析：JDBC DatabaseMetaData API →
    表名、字段名、类型、注释、主键、索引外键 →
5. 生成逻辑模型 IR（人工补充注释、标签）→
6. 标记为"已纳管"状态 →
7. 后续变更在平台发起，不再直接操作数据库
```

#### API

```
POST /api/models/reverse-engineer
{
  "instanceId": 1,
  "schemaName": "ods",
  "tableNames": ["user_order", "user_info"]
}
→ 返回生成的逻辑模型预览，确认后入库
```

---

## 5. 数据库模型

> 以下所有 DDL 基于 MySQL 方言。这是 Rosetta 平台自身的元数据库表结构。

### 5.1 数据源管理

```sql
-- 数据源实例
CREATE TABLE datasource_instance (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(128)  NOT NULL COMMENT '实例名称',
  type          VARCHAR(32)   NOT NULL COMMENT '数据库类型: MYSQL, GAUSSDB_M',
  host          VARCHAR(256)  NOT NULL COMMENT '主机地址',
  port          INT           NOT NULL COMMENT '端口',
  credential    TEXT          NOT NULL COMMENT '加密存储的连接凭证 JSON',
  status        VARCHAR(32)   NOT NULL DEFAULT 'ACTIVE' COMMENT '状态: ACTIVE, INACTIVE',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) COMMENT '数据源实例';

-- Schema（实例下的一个库/模式）
CREATE TABLE datasource_schema (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  instance_id   BIGINT        NOT NULL COMMENT '所属实例ID',
  schema_name   VARCHAR(128)  NOT NULL COMMENT 'Schema名称，如 ods/dwd/dws',
  layer         VARCHAR(32)   NOT NULL COMMENT '数据层级: ODS, DWD, DWS, ADS',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_instance_schema (instance_id, schema_name)
) COMMENT '数据源Schema';
```

### 5.2 逻辑模型

```sql
-- 逻辑模型（表定义 IR，方言无关）
CREATE TABLE logical_model (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  table_name    VARCHAR(256)  NOT NULL COMMENT '表名（snake_case）',
  table_comment VARCHAR(512)  COMMENT '表注释',
  table_status  VARCHAR(32)   NOT NULL DEFAULT 'DRAFT' COMMENT 'DRAFT, PUBLISHED, DEPRECATED',
  source        VARCHAR(32)   NOT NULL DEFAULT 'MANUAL' COMMENT '来源: MANUAL（平台创建）, REVERSE_ENGINEERED（反向工程）',
  created_by    BIGINT        COMMENT '创建人ID',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by    BIGINT        COMMENT '更新人ID',
  updated_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_table_name (table_name)
) COMMENT '逻辑模型';

-- 字段定义
CREATE TABLE model_column (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_id      BIGINT        NOT NULL COMMENT '所属逻辑模型ID',
  ordinal       INT           NOT NULL COMMENT '字段序号（从1开始）',
  column_name   VARCHAR(256)  NOT NULL COMMENT '字段名（snake_case）',
  logical_type  VARCHAR(64)   NOT NULL COMMENT '逻辑类型，如 BIGINT, VARCHAR, DECIMAL',
  type_length   INT           COMMENT '类型长度，如 VARCHAR(32)中的32',
  type_scale    INT           COMMENT '类型精度，如 DECIMAL(18,2)中的2',
  nullable      TINYINT       NOT NULL DEFAULT 1 COMMENT '是否可为空: 1=可空, 0=非空',
  default_value VARCHAR(512)  COMMENT '默认值',
  comment       VARCHAR(512)  COMMENT '字段注释',
  is_primary_key TINYINT      NOT NULL DEFAULT 0 COMMENT '是否主键',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_model_column (model_id, column_name)
) COMMENT '模型字段';

-- 索引定义
CREATE TABLE model_index (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_id      BIGINT        NOT NULL COMMENT '所属逻辑模型ID',
  index_name    VARCHAR(256)  NOT NULL COMMENT '索引名称',
  index_type    VARCHAR(32)   NOT NULL DEFAULT 'NORMAL' COMMENT 'NORMAL, UNIQUE',
  columns       JSON          NOT NULL COMMENT '索引列配置: [{"name":"col","order":"ASC"}]',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_model_index (model_id, index_name)
) COMMENT '模型索引';

-- 外键定义
CREATE TABLE model_foreign_key (
  id              BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_id        BIGINT        NOT NULL COMMENT '所属逻辑模型ID',
  fk_name         VARCHAR(256)  NOT NULL COMMENT '外键约束名',
  column_name     VARCHAR(256)  NOT NULL COMMENT '外键列',
  ref_model_id    BIGINT        NOT NULL COMMENT '引用的逻辑模型ID',
  ref_column_name VARCHAR(256)  NOT NULL COMMENT '引用的列名',
  created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_model_fk_name (model_id, fk_name)
) COMMENT '模型外键';

-- 模型到物理 Schema 的映射部署
CREATE TABLE model_deployment (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  model_id      BIGINT        NOT NULL COMMENT '所属逻辑模型ID',
  schema_id     BIGINT        NOT NULL COMMENT '目标 Schema ID',
  dialect       VARCHAR(32)   NOT NULL COMMENT '部署方言: MYSQL, GAUSSDB_M',
  deployed_ddl  TEXT          COMMENT '部署时实际执行的 DDL',
  deployed_at   DATETIME      COMMENT '部署时间',
  deployed_by   BIGINT        COMMENT '部署人ID',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_model_schema (model_id, schema_id)
) COMMENT '模型部署记录';
```

### 5.3 字典管理

```sql
-- 字典定义
CREATE TABLE dict_definition (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  dict_name     VARCHAR(256)  NOT NULL COMMENT '字典名称',
  dict_code     VARCHAR(128)  NOT NULL COMMENT '字典编码（全局唯一）',
  dict_type     VARCHAR(32)   NOT NULL DEFAULT 'STANDARD' COMMENT 'STANDARD, TYPE_MAPPING, REFERENCE',
  remark        VARCHAR(512)  COMMENT '备注说明',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_dict_code (dict_code)
) COMMENT '字典定义';

-- 字典条目
CREATE TABLE dict_item (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  dict_id       BIGINT        NOT NULL COMMENT '所属字典ID',
  item_code     VARCHAR(128)  NOT NULL COMMENT '条目编码',
  item_name     VARCHAR(256)  NOT NULL COMMENT '条目显示名',
  item_value    VARCHAR(512)  COMMENT '条目值',
  ordinal       INT           NOT NULL DEFAULT 0 COMMENT '排序号',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_dict_item (dict_id, item_code)
) COMMENT '字典条目';
```

### 5.4 RBAC 权限管理

```sql
-- 用户
CREATE TABLE rbac_user (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(128)  NOT NULL COMMENT '用户名（登录用）',
  password      VARCHAR(256)  NOT NULL COMMENT '密码（BCrypt加密）',
  display_name  VARCHAR(128)  NOT NULL COMMENT '显示名',
  email         VARCHAR(256)  COMMENT '邮箱',
  status        VARCHAR(32)   NOT NULL DEFAULT 'ACTIVE' COMMENT 'ACTIVE, DISABLED',
  last_login_at DATETIME      COMMENT '最后登录时间',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_username (username)
) COMMENT '用户';

-- 角色定义
CREATE TABLE rbac_role (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_name     VARCHAR(128)  NOT NULL COMMENT '角色显示名',
  role_code     VARCHAR(64)   NOT NULL COMMENT '角色编码',
  description   VARCHAR(512)  COMMENT '角色描述',
  is_system     TINYINT       NOT NULL DEFAULT 0 COMMENT '是否系统内置角色（不可删除）',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_role_code (role_code)
) COMMENT '角色定义';

-- 用户-角色关联
CREATE TABLE rbac_user_role (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT        NOT NULL,
  role_id       BIGINT        NOT NULL,
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_role (user_id, role_id)
) COMMENT '用户-角色关联';

-- 角色-模型权限绑定（核心权限表）
CREATE TABLE rbac_role_model_permission (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  role_id       BIGINT        NOT NULL COMMENT '角色ID',
  model_id      BIGINT        NOT NULL COMMENT '逻辑模型ID',
  permission    VARCHAR(32)   NOT NULL COMMENT '权限: READ, WRITE',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_role_model_perm (role_id, model_id, permission),
  KEY idx_role_id (role_id),
  KEY idx_model_id (model_id)
) COMMENT '角色-模型权限绑定';
```

#### 5.4.1 系统预置角色数据

```sql
-- 初始化四个预置角色
INSERT INTO rbac_role (role_name, role_code, description, is_system) VALUES
('超级管理员',      'SUPER_ADMIN',       '系统级权限，管理用户、角色、实例；所有模型可读写', 1),
('数据治理管理员',   'GOVERNANCE_ADMIN',  '管理字典、数据标准；所有模型可读',                1),
('数据开发',        'DATA_DEVELOPER',    '被授权模型的读写权限',                            1),
('数据分析',        'DATA_ANALYST',      '被授权模型的只读权限',                            1);

-- 初始化管理员账号（密码需 BCrypt 加密后替换）
INSERT INTO rbac_user (username, password, display_name) VALUES
('admin', '$2a$10$placeholder', '系统管理员');
INSERT INTO rbac_user_role (user_id, role_id) VALUES (1, 1);
```

#### 5.4.2 权限判定逻辑（伪代码）

```
function hasPermission(userId, modelId, requiredPermission):
    user = findUser(userId)
    roles = findRolesByUser(userId)

    for role in roles:
        if role.code == 'SUPER_ADMIN':
            return true  // 超级管理员拥有所有权限

        if role.code == 'GOVERNANCE_ADMIN' AND requiredPermission == 'READ':
            return true  // 治理管理员对所有模型可读

        // 普通角色查权限绑定表
        binding = findBinding(role.id, modelId, requiredPermission)
        if binding != null:
            return true

    return false
```

---

## 6. API 设计

### 6.1 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录，返回 JWT Token |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/me` | 获取当前用户信息及权限 |

### 6.2 用户与角色管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 用户列表（分页） |
| POST | `/api/users` | 创建用户 |
| PUT | `/api/users/{id}` | 更新用户信息 |
| PUT | `/api/users/{id}/password` | 重置密码 |
| PUT | `/api/users/{id}/status` | 启用/禁用用户 |
| GET | `/api/roles` | 角色列表 |
| PUT | `/api/users/{id}/roles` | 为用户分配角色 |
| GET | `/api/roles/{id}/model-permissions` | 获取角色的模型权限矩阵 |
| PUT | `/api/roles/{id}/model-permissions` | 更新角色的模型权限矩阵 |

### 6.3 数据源实例

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/instances` | 实例列表 |
| POST | `/api/instances` | 注册实例 |
| PUT | `/api/instances/{id}` | 更新实例 |
| DELETE | `/api/instances/{id}` | 删除实例 |
| POST | `/api/instances/{id}/test` | 测试连接 |
| GET | `/api/instances/{id}/schemas` | 实例下的 Schema 列表 |
| POST | `/api/instances/{id}/schemas` | 创建 Schema |

### 6.4 逻辑模型

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/models` | 逻辑模型列表（分页 + 搜索 + 权限过滤） |
| POST | `/api/models` | 创建逻辑模型 |
| GET | `/api/models/{id}` | 逻辑模型详情（含字段/约束/索引） |
| PUT | `/api/models/{id}` | 更新逻辑模型基本信息 |
| DELETE | `/api/models/{id}` | 删除逻辑模型 |

### 6.5 模型字段/索引/外键

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/api/models/{id}/columns` | 批量更新字段（含排序） |
| POST | `/api/models/{id}/indexes` | 添加索引 |
| PUT | `/api/models/{id}/indexes/{indexId}` | 更新索引 |
| DELETE | `/api/models/{id}/indexes/{indexId}` | 删除索引 |
| POST | `/api/models/{id}/foreign-keys` | 添加外键 |
| DELETE | `/api/models/{id}/foreign-keys/{fkId}` | 删除外键 |

### 6.6 DDL 渲染与部署

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/models/{id}/ddl?dialect=MYSQL` | 渲染 DDL（指定方言） |
| POST | `/api/models/{id}/deploy` | 部署模型到已映射的 Schema |
| GET | `/api/models/{id}/deployments` | 部署历史记录 |

### 6.7 反向工程

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/instances/{id}/tables?schema=ods` | 浏览实例下可纳管的表 |
| POST | `/api/models/reverse-engineer` | 反向工程，生成逻辑模型预览 |
| POST | `/api/models/reverse-engineer/import` | 确认导入反向工程结果 |

### 6.8 可视化

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/models/{id}/structure-diagram` | 单表结构图数据 |
| GET | `/api/schemas/{id}/er-diagram` | Schema 级 ER 图数据 |

### 6.9 字典

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/dicts` | 字典列表 |
| POST | `/api/dicts` | 创建字典 |
| PUT | `/api/dicts/{id}` | 更新字典 |
| DELETE | `/api/dicts/{id}` | 删除字典 |
| GET | `/api/dicts/{id}/items` | 字典条目列表 |
| POST | `/api/dicts/{id}/items` | 添加条目 |
| PUT | `/api/dicts/{id}/items/{itemId}` | 更新条目 |
| DELETE | `/api/dicts/{id}/items/{itemId}` | 删除条目 |
| PUT | `/api/dicts/{id}/items/order` | 批量更新条目排序 |

---

## 7. 技术选型

### 7.1 后端

| 能力 | 选型 | 版本 | 理由 |
|------|------|------|------|
| 语言 | Go | 1.22+ | 高性能、部署简单（单二进制）、并发原生支持 |
| HTTP 框架 | Gin | 1.x | 高性能 HTTP 框架，路由简洁，中间件生态丰富 |
| ORM | GORM | 2.x | Go 最成熟的 ORM，AutoMigrate 自动建表 |
| 认证 | golang-jwt | 5.x | JWT Token 签发与校验 |
| 密码加密 | golang.org/x/crypto | - | bcrypt 实现 |
| 配置管理 | Viper | 1.x | 支持 YAML/ENV，多环境配置 |
| 数据库驱动 | go-sql-driver/mysql | - | MySQL 驱动 |
| 数据库 | MySQL | 8.0+ | 平台元数据存储 |
| 参数校验 | go-playground/validator | 10.x | 结构体 tag 校验 |
| API 文档 | swaggo/swag | - | 注解生成 Swagger 文档 |
| SQL 解析 | pingcap/tidb/parser | - | 反向工程中解析已有 DDL（可选，P1） | |

### 7.2 前端

| 能力 | 选型 | 版本 | 理由 |
|------|------|------|------|
| 实现方式 | 原生 JS | ES5 | 零框架依赖，单页应用，直接操作 DOM |
| 样式 | 原生 CSS | - | CSS 变量主题系统，侧边栏/表格/弹窗/Toast |
| 路由 | 自研 Hash Router | - | 正则匹配路由，路由守卫，SPA 回退 |
| HTTP | Fetch API | - | 浏览器原生，封装 token 管理 |
| 可视化 | Cytoscape.js | 3.30.4 | ER 图自动布局（breadthfirst），本地 vendored，664KB |
| ER 布局 | breadthfirst | 内置 | 有向图分层布局，避免 dagre 插件的 bundler 依赖 |
| DDL 展示 | 原生 `<pre>` 标签 | - | CSS 模拟深色终端风格 |
| 图标 | Emoji | - | 简洁直观，无外部依赖 |

### 7.3 部署

| 能力 | 选型 |
|------|------|
| 构建 | go build（后端）/ pnpm（前端） |
| 部署 | Docker Compose（Go 二进制 + MySQL + Nginx 托管前端静态资源） |
| Go 版本 | 1.22+ |

---

## 8. 交付计划

### Phase 0：项目初始化（Day 1-2）

```
任务：
  - Go + Gin 项目骨架（backend/ 目录）
  - 数据库初始化脚本（DDL 全量执行）
  - 预置角色、管理员账号数据
  - React + Ant Design + Vite 脚手架
  - 前后端联调基础（CORS、代理配置）
  - 项目 Git 仓库初始化

交付物：可启动的空项目，数据库表已建好，前后端能通信
```

### Phase 1：基础设施（Day 3-7）

```
1. 认证模块
   - 登录/登出 API + 前端页面
   - JWT Token 签发与校验
   - 前端路由守卫（未登录跳转登录页）

2. 用户与角色管理
   - 用户 CRUD + 角色分配
   - 角色列表页
   - 权限判定 AOP 切面（注解式权限校验）

3. 数据源实例管理
   - 实例 CRUD + 连接测试
   - Schema 管理
   - 实例列表 + 详情前端页面

4. 全局布局
   - Ant Design Pro Layout（侧边栏 + 顶栏）
   - 菜单配置 + 路由
```

### Phase 2：核心功能 — 表结构编辑器（Day 8-14）

```
1. 逻辑模型 CRUD 后端
   - 模型基本信息
   - 字段/约束/索引 CRUD

2. 表结构编辑器前端
   - Tab 1：字段可编辑表格（内联编辑 + 拖拽排序）
   - Tab 2：主键/外键约束配置
   - Tab 3：索引配置
   - Tab 4：DDL 预览面板（代码高亮 + 方言切换）

3. DDL 渲染引擎
   - MySqlDdlRenderer
   - GaussDbDdlRenderer
   - 类型映射表驱动

4. 字典维护
   - 左侧树 + 右侧编辑器
   - 字典条目表格编辑
   - 类型映射表预置数据
```

### Phase 3：可视化 + 部署（Day 15-21）

```
1. 单表结构图（G6 自定义节点）
2. Schema 级 ER 图（dagre 自动布局）
3. 角色-模型权限矩阵页面
   - 按角色授权
   - 读/写权限勾选
4. 模型列表按用户权限过滤
5. 反向工程（JDBC 采集 → 填充逻辑模型）
6. 部署执行（DDL 发送到目标数据库实例）
```

### Phase 4：收尾与打磨（Day 22-28）

```
1. 部署历史记录查询
2. 平台定义 vs 数据库实际差异对比（一致性校验）
3. 全局错误处理 + 操作日志
4. 性能优化（N+1 查询、ESLint 清理）
5. Docker Compose 一键部署脚本
6. README + 部署文档
```

---

## 附录 A：GaussDB M 模式注意事项

1. **GaussDB M 模式兼容 MySQL 语法**，大部分 MySQL DDL 可直接执行，但以下需要适配：
   - 建表尾缀 `ENGINE=InnoDB` 需去掉
   - `AUTO_INCREMENT` 用 `GENERATED BY DEFAULT AS IDENTITY` 替代
   - 列注释和表注释语法差异（`COMMENT ON` 独立语句）
   - `DATETIME` 类型映射为 `TIMESTAMP`
   - `TEXT` 类型映射为 `CLOB`
2. **反向工程采集**：GaussDB M 提供 JDBC 驱动，`DatabaseMetaData` API 可用，与 MySQL 兼容
3. **建议每个方言的差异用独立的 DdlRenderer 策略实现**，不要在一套代码里用 `if/else` 分支撑

## 附录 B：与原始"大而全"方案的差异

| 维度 | 原始方案 | Rosetta 方案 |
|------|----------|-------------|
| 定位 | 观测型治理平台（被动） | DDL 源头管控平台（主动） |
| P0 模块 | 元数据采集、数据质量、数据目录 | 字典维护、表结构编辑器、可视化、RBAC |
| 元数据来源 | 全量自动采集 | 平台定义为 Source of Truth，采集仅做反向工程和一致性校验 |
| 用户系统 | 未涉及 | RBAC 模型，角色-模型粒度的 READ/WRITE 权限 |
| 质量监控 | P0（MVP 必备） | P1（表结构纳管后再补） |
| 数据血缘 | P1 | P2 |
| 数据库方言 | 仅提及 MySQL、Hive | GaussDB M + MySQL，含完整映射差异表 |
| 交付周期 | 6 个月（四阶段） | 4 周 MVP（聚焦 UI 可操作的闭环） |

---

## 附录 C：Cytoscape.js 本地化方案

### 为什么不用 CDN

1. CDN 可能被墙或加载超时，导致页面白屏
2. `cytoscape-dagre` 插件是 webpack 打包产物，依赖 `require("dagre")`，在普通 `<script>` 标签下不工作

### 最终方案

```
frontend/js/vendor/
├── cytoscape.min.js    (365KB) — 从 jsdelivr CDN 下载到本地
├── dagre.min.js        (278KB) — 已废弃，breadthfirst 布局不需要
└── cytoscape-dagre.js   (13KB) — 已废弃，不兼容浏览器环境
```

- 使用 Cytoscape.js 内置的 `breadthfirst` 布局，零外部布局依赖
- 适合有向图（FK 关系有明确方向），自动分层排列
- 自引用边（`product_category.parent_id → product_category.id`）自动跳过

### 节点高度计算

Cytoscape v3.30+ 废弃了 `height: 'label'`，改为根据标签行数动态计算：

```js
'height': function(ele) { return ele.data('nodeHeight') || 200; }
// nodeHeight = 标签行数 * 18 + 24 (padding)
```

## 附录 D：100 表 Dummy ERP 系统

```bash
cd backend && ./rosetta-server --seed-dummy
```

| 域 | 表数 | 典型示例 |
|----|:--:|----------|
| 电商 | 20 | product, order, order_item, inventory, coupon, refund |
| CRM | 17 | customer, account, contact, lead, opportunity, contract |
| 财务 | 15 | gl_account, gl_journal, ap_invoice, ar_invoice, budget |
| HR | 17 | employee, department, position, salary, attendance, payroll |
| 物流 | 15 | warehouse, stock, shipment, purchase_order, receiving |
| 分析 | 17 | report_config, dashboard, kpi_definition, etl_job, data_lineage |

首次运行创建 101 张表、582 个字段、103 个外键，全部部署到 `dummy_erp` Schema。
