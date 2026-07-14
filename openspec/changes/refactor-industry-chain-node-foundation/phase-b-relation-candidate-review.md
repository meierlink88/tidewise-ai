# Phase B relation and physical constraint candidate Review（R0）

## 输入、生成规则与输出

- 输入：[842 node/profile manifest](final-seed-candidate-artifacts/node-profile-seed-manifest.json)，SHA-256=`9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e`。
- 生成器：[generate_relation_candidates.py](relation-candidate-artifacts/generate_relation_candidates.py)，规则版本=`phase-b-semantic-v1`。
- 全量候选：[relation-candidate-review.json](relation-candidate-artifacts/relation-candidate-review.json)，SHA-256=`25c1fe1d23c3e2c44814fe50ffc73b5c256c05149ba37c94af09978c8a90327b`。
- 算法只枚举每个名称自身的后缀并查找已批准节点，复杂度为 `O(sum(name_length))`；不构造 842×842 全排列。

| disposition | 数量 | 含义 |
| --- | ---: | --- |
| `reviewable_semantic` | 96 | definition/name/boundary 足以支持第二遍 Reviewer 判断；`is_subcategory_of`=95、`is_component_of`=1 |
| `blocked_needs_evidence` | 53 | `is_subcategory_of`=47、`input_to`=5、`depends_on`=1，缺权威分类、BOM、工艺、认证或不可替代性证据 |
| physical constraint blocked | 4 | 半导体良率、半导体设备产能、锂矿资源、电池材料纯度；均未达到可写证据标准 |
| rejected candidate | 5 | 明确并列/复合名称，不能推出全集从属 |

另以规则级拒绝固定自环、alias/synonym、旧关系类型、动态事件传导及同机制 `input_to`/`depends_on` 重复。当前自环、禁用关系类型、write-ready input/depends、write-ready physical constraint 均为 0。

## 第一遍 AI double check

`reviewable_semantic` 的方向是窄节点 A → 宽节点 B；每条均保存 from/to 名称、entity key、definition、boundary、mechanism、derivation rule、反例与不确定性。确定性 QA 抽样包括：

- `汽车零部件 -> 汽车`（`is_component_of`）：名称直接表达可识别物理/系统组成；反例条件是“零部件”实际包含非汽车用途。
- `AI芯片 / MCU芯片 / 存储芯片 / 汽车芯片 -> 芯片`（`is_subcategory_of`）：全部实例按产品类别属于芯片；若节点只是主题标签则拒绝。
- `BC电池 / HJT电池 / TOPCon电池 / 固态电池 / 钠离子电池 -> 电池`：名称和产品定义直接支持分类范围；不推导供应或替代。
- `其他专业工程 -> 专业工程`、`其他通信设备 -> 通信设备`：只表达来源分类中的剩余子类；第二遍 Reviewer 必须检查 parent boundary 是否覆盖“其他”桶。
- `降解塑料 -> 塑料`、`麒麟电池 -> 电池`：词面与 definition 同时支持；仍保留“品牌/市场标签不等于稳定子类”的反方条件。

第一遍未把任何投入、依赖或物理约束升级为可写事实。6 条专业线索（锂→锂电池、半导体材料→半导体、半导体→半导体设备、光伏主材→光伏电池组件、稀土产业→稀土永磁、铜产业→铜缆高速连接）逐项写明所需 BOM、工艺、技术标准、认证、产线或不可替代性来源。

## Physical constraint Review

| subject | type | 为什么可能是硬约束 | 当前证据 | 缺失证据 / disposition |
| --- | --- | --- | --- | --- |
| 半导体 | `process_yield` | 良率直接限制合格产出 | 仅节点语义 | 缺工艺节点良率、敏感性和产能损失来源；blocked |
| 半导体设备 | `equipment_capacity` | 设备交付、安装、认证可限制扩产速度 | 仅节点语义 | 缺交付周期、装机、认证与扩产数据；blocked |
| 锂矿 | `resource_availability` | 储量、品位和开发周期限制物理供给 | 仅节点语义 | 缺储量、品位、许可、建设周期和产量来源；blocked |
| 电池 | `material_purity` | 材料纯度可限制性能、良率与合格率 | 仅节点语义 | 缺材料规格、失效机理和资格认证来源；blocked |

价格、政策、情绪、市场表现和动态事件均未作为物理约束。

## 第二遍 Reviewer 与下一步边界

主对话将以 Serenity 的瓶颈/物理约束逻辑逐项复核 `reviewable_semantic`、blocked 与 rejected 全集，决定 approve/reject/blocked。此候选包不是可执行 seed，不授权 migration 17、relation/constraint Write、Neo4j 或 task 2.6/2.7。

## R1 实现与只读边界

- `000017_add_chain_node_relations.sql` 仅为未 apply 的 migration 源码；relation/constraint schema 尚不存在于 live PostgreSQL。
- `entity-seed -chain-node-relation-manifest <path> -chain-node-relation-dry-run` 是当前唯一 relation-only CLI 形状，只在 `REPEATABLE READ READ ONLY` snapshot 中计算 created/updated/unchanged/conflict，且拒绝与普通 entity/mapping seed 参数混用。
- domain/loader 拒绝旧 topology/constraint 字段、四类已删除关系、自环、tuple 重复、inactive/非 chain_node 端点及同机制 `input_to`/`depends_on` 双记。
- 本 checkpoint 未连接或写入 PostgreSQL/Neo4j，也未生成 relation/constraint 可写 seed；第二遍 Review 后仍须分别准备 task 2.6 schema R2 与 task 2.7 data R2。
