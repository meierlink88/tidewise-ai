# Tidewise AI OpenSPG Object Schema 开发规范

## 范围与权威

本规范适用于 Tidewise AI 中所有 `doctype/*.schema` Object Schema 的设计、创建、修改、
生成与审查。Schema 统一使用本规范批准的 OpenSPG Schema Mark Language 子集；不得使用
JSON Schema，也不得发明 OpenSPG 解析器不支持的关键字。

执行任何 Object Schema 任务前必须完整读取本规范。OpenSPG 官网和官方仓库只提供上游
语法证据；本文件是 Tidewise AI 的直接工程规范。业务术语、对象边界和字段含义仍由对应
Context、ADR 和 GitHub Issue 验收条件决定，本规范不替代领域权威。

## 上游基线

本次核验基于 OpenSPG 官方文档入口，以及 OpenSPG/KAG 官方仓库提交
[`fdab15b3929d2ee40dfcdd388f90233096a6afc9`](https://github.com/OpenSPG/KAG/tree/fdab15b3929d2ee40dfcdd388f90233096a6afc9)。

## 允许的语法

官方 MarkLang 解析器和模板支持以下结构：

```text
namespace Tidewise

Region(区域): EntityType
    desc: 区域对象
    properties:
        regionType(区域类型): Text
            desc: 区域对象的形成或使用类型
            constraint: NotNull, Enum="CONTINENT,GEOGRAPHIC,MULTILATERAL,INVESTMENT"
```

- 文件首先声明 `namespace <Name>`；命名空间只接受 ASCII 字母和数字。
- 类型声明采用 `TypeName(中文名): EntityType`；还支持 `ConceptType`、`EventType`、
  `IndexType`、`StandardType` 和继承声明。
- 属性放在 `properties:` 下，写作 `propertyName(中文名): RangeType`。
- 实体关系放在 `relations:` 下，目标必须是 Schema 类型，而不是基础类型。
- 类型、属性和关系都可以使用 `desc:`。
- 属性约束支持 `NotNull`、`MultiValue`、`Enum="a,b"` 和
  `Regular="..."`；当前 Python MarkLang 解析器没有实现 `Unique`。
- 属性索引支持 `Text`、`Vector`、`SparseVector`、`TextAndVector` 和
  `TextAndSparseVector`。
- 以 `#` 开头的整行是注释。
- 当前 Python MarkLang 内建基础类型只有 `Text`、`Integer` 和 `Float`。

证据：

- [OpenSPG v2 中文文档入口](https://openspg.github.io/v2/docs_ch)
- [官方默认 Schema 模板](https://github.com/OpenSPG/KAG/blob/fdab15b3929d2ee40dfcdd388f90233096a6afc9/kag/templates/schema/%7B%7Bdefault%7D%7D.schema.tmpl)
- [官方 Schema MarkLang 解析器](https://github.com/OpenSPG/KAG/blob/fdab15b3929d2ee40dfcdd388f90233096a6afc9/knext/schema/marklang/schema_ml.py)
- [官方基础类型、约束和索引枚举](https://github.com/OpenSPG/KAG/blob/fdab15b3929d2ee40dfcdd388f90233096a6afc9/knext/schema/model/base.py)
- [官方示例 Schema](https://github.com/OpenSPG/KAG/blob/fdab15b3929d2ee40dfcdd388f90233096a6afc9/kag/examples/FinAlibaba/schema/FinAlibaba3.schema)

## 项目约束与持久化边界

以下内容不是 OpenSPG MarkLang 原生语法，不能在 `.schema` 中发明关键字表达：

- PostgreSQL 表名和列名；
- `VARCHAR(n)` 长度；
- 主键、唯一约束和默认值；
- 数据库生成值和只读属性；
- `TIMESTAMPTZ` 等 PostgreSQL 类型；
- 每个枚举值各自的结构化说明。

这些内容由对应 PostgreSQL migration、Data Adapter 和合同测试表达，并与 Object Schema
保持语义一致：

- OpenSPG 属性使用 camelCase，例如 `regionType`、`nameEn`、`createdAt`；映射到
  PostgreSQL `region_type`、`name_en`、`created_at`。
- OpenSPG 的 `Enum` 负责限制允许值；每个值的业务含义写入属性 `desc:`，必要时辅以
  相邻 `#` 注释。不要新增 OpenSPG 解析器不认识的枚举描述语法。
- PostgreSQL 仍独立实施主键、唯一性、长度、默认值和时间类型约束；Schema 和 migration
  必须通过项目测试保持一致。
- `.schema` 是 OpenSPG 官方项目普遍采用的文件扩展名，但解析器按文本内容读取文件，
  扩展名本身不是语法的一部分；Tidewise 将其固定为项目约定。

## 编写与验证规则

- Object Schema 文件统一使用 `.schema` 扩展名，并直接存放在 owning application 的
  `doctype/` 目录。
- 一个 Object Type 使用一个同名小写文件，例如 `Region` 使用 `region.schema`。
- 每个类型、属性和关系必须提供中文显示名；每个类型和业务属性必须提供非空 `desc:`。
- 每个受控枚举必须同时给出机器约束值和每个枚举值的业务含义。
- 新增或修改 Schema 时，必须使用本规范固定基线兼容的 OpenSPG MarkLang parser 校验。
- Schema 与 PostgreSQL 同时变化时，测试必须覆盖字段映射、必填、枚举、唯一性和默认值，
  防止两侧合同漂移。

## Region 示例

`Region` 是 `EntityType`。`regionType` 是 Region 的单值 `Text` 属性，使用 `NotNull` 和
`Enum` 约束四个值。OpenSPG Schema 描述对象语义和机器可识别的属性约束；它不替代
PostgreSQL `regions` 表及其持久化约束。
