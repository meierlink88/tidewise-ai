# A：Schema / Data Contract Review（已批准）

## 0. Review 状态与边界

本文记录主对话对 tasks 1.1—1.4 的已批准契约；它仍不是 Apply、migration、seed 或数据库写入授权。基线为 2026-07-13 已 fetch 的 `origin/main@f942d7615afd952840cdc478bbe7b4ecc990616d`，仅审计现有 schema、entityfoundation loader/repository、seed JSON 与 graph projection 实现。

- A 层 Contract Review 已通过；只授权继续准备 B 层 provisional candidate draft。
- 本文不冻结联盟候选、economy 范围或关系候选，不开始 tasks 2.x。
- 本文不修改源码、migration、seed、PostgreSQL 或 Neo4j。
- 下文以“建议”标出的约束，只有主对话逐项批准后才成为实现输入。

## 1. `alliance_org_profiles` 最小字段契约

### 1.1 已批准字段矩阵

| 字段 | 已批准类型与 null/default | trim/长度 | 唯一性与组合约束 | 与 `entity_nodes` 的关系 |
|---|---|---|---|---|
| `entity_id` | `UUID NOT NULL`，无 default；PK/FK `entity_nodes(id)` | 不适用 | 一实体最多一个 profile；写入前还须校验目标 node 的 `entity_type=alliance_org` | 名称、canonical name、aliases、status 均不在 profile 重复保存 |
| `abbreviation` | `TEXT NOT NULL DEFAULT ''` | 写入前 `btrim`；`char_length <= 32`；空串表示没有可审计正式简称，禁止 `NULL`、`—`、`-` 充当缺失值 | **不全局唯一**：不同组织可能同简称；非空值必须在同一 node 的 aliases 中恰好有一个 NFKC + Unicode casefold 等价值 | 保留原始展示大小写，不强制转大写；识别用 alias 规范化值，展示仍读 profile |
| `categories` | `TEXT[] NOT NULL`，无 default | 每项先 `btrim`、ASCII lowercase；单项 `1..64` 字符；数组 `1..8` 项 | 仅允许批准的 22-code allowlist；规范化后去重并按 code 字典序持久化；禁止空项、`/`、`、`、`|` 等拼接值；不得用父类自动补标签 | category 不是实体标签，不进入 aliases，也不改变 `entity_type`/`layer_code` |
| `leadership_summary` | `TEXT NOT NULL`，无 default | `btrim` 后非空；`char_length <= 500` | 候选草案可显示缺失并标为 blocker；最终 approved active alliance 不允许空串，不得以“未知”“—”占位 | 只描述治理、轮值、秘书处或可审计主导方式；不得由文本自动生成 `led_by` |
| `influence_scope_summary` | `TEXT NOT NULL`，无 default | `btrim` 后非空；`char_length <= 1000` | 候选草案可显示缺失并标为 blocker；最终 approved active alliance 不允许空串，不得保存评分或投资判断 | 只描述地理、议题或制度覆盖范围；不替代 categories、关系或 observation |

### 1.2 名称、简称与 aliases 规则

以下规则已作为一个原子契约批准：

1. `entity_nodes.name` 与 `canonical_name` 继续保存规范中文主名称；写入前 `btrim`，不得为空。英文正式名、通行旧称、缩写进入 aliases。
2. aliases 每项 `btrim` 后不得为空或为缺失占位符；单项不超过 128 字符、每实体不超过 64 项。
3. aliases 在 NFKC + Unicode casefold 后按等价值去重；不得重复 `name` 或 `canonical_name` 的等价值。
4. 非空 `abbreviation` 必须能在 aliases 中找到恰好一个规范化等价值；profile 保留展示形式，例如 `ASEAN`，aliases 可包含同一展示值但不得再含 `asean` 作为重复项。
5. abbreviation 不做全局唯一键；跨实体发生等价简称时必须进入候选冲突报告，实体识别不得只凭简称自动合并。
6. `entity_key` 与 UUID 才是稳定 identity。名称、简称、aliases 变更不得产生新 identity。

### 1.3 Categories 首版已批准 allowlist（22 项）

下表是已批准的 schema allowlist。示例只说明分类边界，不代表任何具体联盟候选已获批准。

| code | 中文名 | 定义 | 准入边界 | 排除边界 | 示例类型 |
|---|---|---|---|---|---|
| `political` | 政治协调 | 国家或政府间外交立场、权力协调与政治协商 | 正式章程或持续机制明确以政治/外交协调为核心职能 | 一般行政规则、纯技术合作、单纯安全行动 | 跨政府外交协调机制 |
| `governance` | 治理与规则 | 跨国制度、规则制定、公共治理或组织治理协调 | 有明确规则制定、监督或全球/区域公共治理职责 | 仅因“是国际组织”而自动归类；普通政治会晤 | 跨境规则制定组织 |
| `security` | 安全 | 非纯军事的国家、地区、边境、网络或综合安全协作 | 明确安全政策、执法或风险协作职能 | 仅武装防御时优先 `military`；经济安全不能因文字出现“安全”自动加入 | 区域安全对话机制 |
| `military` | 军事与防务 | 武装力量、防务安排、集体防御或军事协同 | 条约或机制明确包含军事、防务、联演或集体防御 | 泛安全对话、警务或情报合作 | 集体防御组织 |
| `intelligence` | 情报合作 | 情报收集、交换、协调或共同评估 | 可审计职能明确包含情报合作 | 一般信息共享、研究报告交换 | 多边情报协作机制 |
| `economic` | 宏观经济协调 | 宏观经济、产业总体政策或跨领域经济协调 | 明确覆盖宏观经济政策或综合经济治理 | 不因存在贸易、金融或发展职能而自动补父类 | 宏观经济政策论坛 |
| `trade` | 贸易 | 跨境货物/服务贸易、关税、市场准入或贸易规则 | 正式职能以贸易制度或便利化为核心 | 国内商业协会、纯投资或宏观经济议题 | 区域贸易协定组织 |
| `finance` | 金融与货币 | 货币、金融稳定、银行、资本、支付或多边融资 | 明确金融监管、货币协作或融资职能 | 一般经济协调、单纯发展援助 | 多边金融稳定机构 |
| `development` | 发展合作 | 长期发展、减贫、能力建设、基础设施或发展援助 | 正式使命包含发展融资或发展合作 | 一次性人道救援、一般宏观经济讨论 | 多边发展机构 |
| `energy` | 能源 | 能源供应、燃料、发电、能源政策或能源转型协调 | 组织核心职能直接作用于能源系统或能源商品 | 所有矿产或一般商品组织；不能因某矿产用于能源而自动加入 | 跨国能源协调组织 |
| `mineral` | 矿产资源 | 金属、非金属矿物的开发、供应或行业协作 | 核心对象是矿物或矿产供应链 | 石油天然气等能源商品；农产品 | 矿产生产国协作组织 |
| `agriculture` | 农业 | 农业生产、农产品、粮食生产体系或农业政策 | 核心对象是农业与农产品体系 | 人道救援本身、所有食品议题 | 农产品生产国组织 |
| `technology` | 科技 | 科学技术、数字、标准、研发或创新协作 | 正式使命以科技政策或技术协作为核心 | 单纯使用技术工具的组织 | 国际科技合作机制 |
| `religion` | 宗教 | 宗教共同体、宗教事务或跨宗教协作 | 身份或正式使命明确以宗教为核心 | 具有宗教历史但当前使命不以宗教为核心 | 跨国宗教组织 |
| `health` | 卫生健康 | 公共卫生、疾病防控、医疗或健康治理 | 正式使命包含跨境卫生健康职责 | 一般人道援助、食品供应 | 国际卫生组织 |
| `culture` | 文化 | 文化遗产、文化交流或文化政策协作 | 正式职责明确包含文化 | 教育本身、一般公共外交 | 跨国文化合作组织 |
| `education` | 教育 | 教育政策、学术、人才培养或教育合作 | 正式职责明确包含教育 | 文化活动、一般研究交流 | 国际教育合作机制 |
| `environment` | 环境与气候 | 环境保护、气候、生物多样性或生态治理 | 正式使命直接包含环境或气候 | 能源转型组织不能仅因减排外溢自动加入 | 多边气候治理机制 |
| `nuclear` | 核事务 | 核能、核安全、核保障、核不扩散或核技术治理 | 正式职责直接涉及核事务 | 一般能源或军事组织不能因存在核相关成员自动加入 | 国际核治理机构 |
| `legal` | 法律协调 | 国际法、立法协调、司法合作或法律标准 | 正式使命以法律规则或司法协作为核心 | 仅因章程具有法律效力而加入 | 跨境法律合作组织 |
| `dispute_resolution` | 争端解决 | 仲裁、调解、裁判或制度化争端解决 | 有正式争端解决职能或常设机制 | 一般政治磋商、非制度化协调 | 国际仲裁或裁判机构 |
| `humanitarian` | 人道援助 | 紧急救援、难民、灾害响应或人道保护 | 正式使命以人道行动为核心 | 农业生产、长期发展合作、一般卫生职责 | 多边人道救援组织 |

### 1.4 重叠收敛规则

1. **删除 `cross_domain`**：categories 已是多值数组，跨领域直接存多个原子 code。
2. **拆分复合 code**：`culture_education` 拆为 `culture`、`education`，`legal_dispute_resolution` 拆为 `legal`、`dispute_resolution`，`food_humanitarian` 收敛为 `humanitarian`；粮食生产/政策使用 `agriculture`。
3. **删除 `commodity`**：不保留宽泛兜底。跨商品组织直接使用实际满足定义的 `energy`、`mineral`、`agriculture`、`trade` 等多个原子 code。
4. **不自动补宽类**：出现 `trade`、`finance`、`development` 不自动补 `economic`；出现 `military` 不自动补 `security`。只有正式使命独立满足两个定义时才可多选。
5. **政治与治理分开**：`political` 表示政治/外交协调，`governance` 表示规则制定和公共治理。组织具有章程或理事会不构成 `governance` 准入依据。

## 2. Economy identity / ISO 契约

### 2.1 四类 identity 组合校验矩阵

| `identity_kind` | identity 边界 | `entity_key` | `country_code` / ISO 3166 | `currency_code` | `region` | `member_of` 边界 |
|---|---|---|---|---|---|---|
| `sovereign_state` | 经本项目权威 identity Review 认定为主权国家的 economy identity；ISO 代码本身不用于推断主权 | `economy:<alpha2 lowercase>` | `^[A-Z]{2}$` 且必须命中获批时点的 ISO 3166-1 alpha-2；与 key 后缀一致 | ISO 4217 alpha-3；多法定货币仅可作为逐项批准的 `MULTI` 例外，未审阅 fail-closed | 必须是受控 region，不能为 `global` | 可作为正式成员端点 |
| `territory_economy` | 具有独立 ISO 代码和统计/经济身份、但不等同于独立主权国家的地区经济体 | `economy:<alpha2 lowercase>` | `^[A-Z]{2}$` 且命中 ISO 3166-1；与 key 后缀一致 | ISO 4217 alpha-3；多法定货币仅可作为逐项批准的 `MULTI` 例外，未审阅 fail-closed | 必须是受控 region，不能为 `global` | 仅在联盟官方成员来源明确把该 economy identity 列为正式成员时可建边 |
| `supranational_aggregate` | 多个 economy 的超国家聚合身份，不替代组成成员 | 仅批准的内部保留 key；首版为 `economy:eu` | 不是主权国家 ISO 断言；仅批准保留 code，首版 `EU`；key 与保留表一致 | 单一聚合货币可用 ISO 4217，首版 `EUR`；否则须审批 `MULTI` | 受控 region；EU 为 `europe` | 仅当官方成员名单明确把聚合体本身列为正式成员时才可候选；不得替代成员国 |
| `global_aggregate` | 全球统计/分析聚合，不是国家或组织成员 | 首版唯一 `economy:global` | 首版唯一保留 code `GLOBAL` | 必须为 `MULTI` | 必须为 `global` | 禁止作为 `member_of` source |

### 2.2 Economy 字段通用规则

- `identity_kind`：`TEXT NOT NULL`、无 default、仅允许上述四值；不允许用空值推断类型。
- `country_code`：写入前 trim + ASCII uppercase；`NOT NULL`，但不建立无条件全表 `UNIQUE`，因为 inactive/merged source identity 需要保留。validator、manifest 与 Query 必须保证同一 code 只有一个 approved active economy。`EU` 与 `GLOBAL` 是内部保留 code，不得宣传为主权国家 ISO 代码。
- `currency_code`：写入前 trim + ASCII uppercase；主权国家/地区经济体通常是已审计 ISO 4217 alpha-3。`MULTI` 只允许 global、经批准的 supranational aggregate，或主权/地区 economy 的逐项批准多法定货币例外；未审阅 fail-closed，且不得作为缺失占位符。
- `region`：写入前 trim + ASCII lowercase。现有 `africa`、`asia`、`central_asia`、`europe`、`europe_asia`、`global`、`middle_east`、`north_america`、`oceania`、`south_america` 仅作为首轮兼容 allowlist；C 层逐项审计并报告歧义，A 层不另建区域体系。
- `entity_key`：稳定内部 identity，不因中文名、英文名、currency 或 region 变化而改变。国家/地区 key 必须与 alpha-2 code 一致；聚合 identity 只能来自显式保留表。preflight 清理空值/重复后建立全局唯一约束；merged source 保留自身不同 stable key。
- aliases：至少包含可审计英文正式名或通行英文名；遵循 1.2 的 trim、NFKC + casefold 去重规则。ISO code 与 currency code 不自动进入 aliases，除非它本身是广泛使用且经候选 Review 的名称。

### 2.3 EU、GLOBAL、地区经济体与主权国家边界

- `economy:eu` 是 `supranational_aggregate`，不等同于任何欧盟成员国，也不能替代成员国的 `member_of`。
- `economy:global` 是 `global_aggregate`，只用于全球范围聚合，禁止生成 `member_of`。
- `economy:hk`、`economy:tw` 等具有 ISO 代码的独立统计 economy，应按 `territory_economy` 审阅；这一定义只描述数据 identity，不作主权判断，规范中文名继续服从主规格。
- 一个现实对象只能有一个 active 稳定 economy identity；同 ISO/code 的重复 active node 必须进入未来 economy exception Review，不能按名称自动 merge。
- 是否存在其他 supranational aggregate、哪些地区 economy 纳入候选，必须等联盟清单批准、官方成员全集形成后再决定；本文不冻结范围。

## 3. 明确不入库与图标签边界

以下内容已批准为硬性排除项：

- 不入 `alliance_org_profiles`、`economy_profiles` 或 `entity_nodes`：子类、CSV 成员数、全球占比、约束力级别、影响力评级。
- CSV 成员数只可在后续 Review 中作非权威对照；正式成员数必须由 approved active `member_of` 关系计算并与官方集合核对。
- 全球占比属于后续 observation；约束力与影响力属于后续分析评价，不伪装为基础实体属性。
- 不新增实体标签机制，不复用事件标签，也不把 categories 投影为 Neo4j labels。
- Neo4j 继续只有单一 `Entity` node label；PostgreSQL 仍是事实源。任何 Neo4j Rebuild 仍受 PG 全部 Query 验收后的独立门禁约束。

## 4. 与 `origin/main` 的 exact diff（只读设计差异）

### 4.1 当前事实

| 边界 | `origin/main` 当前状态 | 本契约已批准目标 | exact diff / 风险 |
|---|---|---|---|
| `alliance_org_profiles` | `entity_id`、`org_code VARCHAR(64) NOT NULL UNIQUE`、`org_type VARCHAR(64) NOT NULL`、`primary_domain VARCHAR(64) DEFAULT ''`、`scope_region VARCHAR(64) DEFAULT ''`、`official_url TEXT DEFAULT ''`；另有 `org_type`、`primary_domain` 索引 | 最小五字段 | 保留 `entity_id`；新增 `abbreviation`、`categories`、`leadership_summary`、`influence_scope_summary`；旧五个业务列及两个索引未来如何兼容/移除必须在 alliance Write Review 展示 |
| alliance loader | 必填 `org_code`、`org_type`，只检查非空字符串 | 改为新字段及 allowlist/alias 组合校验 | 当前 validator 不识别数组、不做长度、trim、NFKC/casefold、类别边界或 node type 组合校验 |
| alliance repository | 固定写旧五列 | 固定写新四个业务字段 | 直接改列会使旧 seed/测试/调用方失败；必须先全仓引用审计和 forward migration 测试 |
| `economy_profiles` | `entity_id`、`country_code VARCHAR(16) NOT NULL`、`currency_code VARCHAR(16) NOT NULL`、`region TEXT DEFAULT ''`；无 code 唯一/check 约束 | 增 `identity_kind` 并加组合校验 | 需先审计现有值和重复；不能用 default 回填 identity kind 掩盖分类错误 |
| economy loader/repository | 只要求非空 `country_code`、`currency_code`；repository 写三字段 | 校验 kind/code/key/currency/region 矩阵 | 当前会接受任意非空 code，无法区分 ISO 与内部保留 code |
| `entity_nodes.entity_key` | `TEXT NOT NULL DEFAULT ''`，只有普通索引 `idx_entity_nodes_entity_key` | 稳定且唯一的业务 identity | 当前 DB 不阻止重复 key；未来唯一约束前须做重复/空 key preflight，且不得误伤 convergence 保留 source identity |
| aliases | `TEXT[] NOT NULL DEFAULT '{}'`；repository 仅过滤空字符串并按**完全相等**去重 | trim + NFKC/casefold 去重、简称组合约束 | 大小写、全半角、首尾空格重复和跨实体简称冲突目前不会被阻止；规范化必须先 dry-run exact diff |
| entity/profile FK | profile FK 只保证 node 存在 | profile 还须匹配正确 `entity_type` | PostgreSQL FK 不能表达 type 条件；未来需在 validator/repository preflight 或经审阅的 DB 约束方案中 fail-closed |
| seed 基线 | 10 个 alliance、50 个 economy；EU/GLOBAL 已存在但无 `identity_kind` | 逐项 Review 后 forward convergence | 不得从当前值自动推断所有 identity kind；本轮不生成 seed |
| Neo4j | writer 使用单一 `Entity` label，投影通用 node 属性 | 保持不变 | categories/identity_kind 是否需要普通属性投影须在未来 Neo4j Review 决定，但不得变成 label |

### 4.2 未来 migration 风险与门禁

1. **旧列删除/改名风险**：现有 seed、loader、repository、tests 与潜在查询都引用旧列。未来必须先全仓引用审计，采用新增列、受审回填、兼容窗口、再决定旧列处置的 forward migration；禁止 rename 后假定语义等价。
2. **数据回填风险**：`org_type`/`primary_domain` 不能机械映射为 categories；`org_code` 也不能无 Review 直接复制为 abbreviation。所有转换都来自批准候选 manifest。
3. **唯一性风险**：`org_code` 当前全局唯一，而建议 abbreviation 不全局唯一；`entity_key` 当前反而不唯一。改变约束前必须输出冲突清单、目标 identity 与预计 counts。
4. **数组约束风险**：PostgreSQL check 可验证非空和元素形态，但完整 allowlist、规范排序与 aliases 等价规则需要 validator/repository 测试共同保证；不得只依赖应用层未覆盖的隐式约定。
5. **economy 分类风险**：给现有 50 条统一 default identity kind 会把 EU/GLOBAL/地区 economy 混成主权国家。必须逐项生成只读差异并经过 C 层单独 Review。
6. **identity 收敛风险**：任何 duplicate/merge/inactivate 都必须进入未来 exception manifest 与 Write Review；本文不授权清理。
7. **共享实现风险**：本 change 的源码、migration、seed 和数据库写仍须等待 `refactor-industry-chain-node-foundation` Deliver 后，从最新 `origin/main` 重做 overlap audit。

## 5. 已批准结论

1. 批准五字段 null/default/trim/length/alias 组合规则；abbreviation 最长 32 字符且不做全局唯一键，aliases 单项最长 128 字符、每实体最多 64 项，按 NFKC + casefold 去重，冲突进入 Review。
2. categories 为 1—8 个规范化原子 code、无 default、字典序持久化且不自动补宽类；删除 `cross_domain` 和 `commodity`，拆分三个复合 code，首版 allowlist 为 22 项。
3. 两个 summary 的候选草案可以显示缺失并标为 blocker；最终 approved active alliance 必须 `NOT NULL`、无 default、`btrim` 非空，长度上限分别为 500/1000。未来 migration 可有受控兼容窗口，但不得永久 `DEFAULT ''`。
4. 批准 economy 四类 identity、EU/GLOBAL 边界和 `MULTI` fail-closed 例外规则。
5. `country_code` 不建立无条件全表唯一约束；同 code 只有一个 approved active economy。`entity_key` 在 preflight 清理后建立全局唯一约束，merged source 保留不同 stable key。
6. 现有 10 个 region code 仅作为首轮兼容 allowlist；C 层逐项报告歧义，不在 A 层另建区域体系。
7. 批准明确不入库字段、无实体标签机制、Neo4j 单一 `Entity` label，以及 exact diff/migration preflight 边界。

## 6. 后续 Review 边界

- A 批准不代表任何具体联盟候选、categories 映射、summary 文本或 disposition 已获批准。
- B 只能提交 provisional candidate draft；最终 decision 必须由主对话逐项填写。
- B 未通过 task 2.3 前不得启动 C、读取或冻结成员全集、生成 seed 或连接 PostgreSQL/Neo4j。

## 7. 本 checkpoint 的验收结论

tasks 1.1—1.4 已由主对话批准。当前只允许准备 tasks 2.1—2.2 的 provisional review draft，并停在 2.3 等待逐项 Review。
