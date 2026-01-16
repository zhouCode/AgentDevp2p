# AgentDevp2p — SPEC (V1.0)

## 1. 项目定义

**项目名**：AgentDevp2p

**目标**：一个基于 Geth 源码修改的差分 Fuzzing 工具，用于 Ethereum 客户端兼容性测试。

**核心策略**：

- 在本地异构测试网中，对 6 个不同实现的节点发送“同一份”devp2p/RLPx/ETH 协议输入（含合法与非法/畸形样本）。
- 收集每个客户端在相同输入下的网络层与协议层响应（消息序列、断连理由、耗时、错误模式）。
- 对比差异并做聚类，输出可复现的差分样本（最小化输入 + 目标客户端集合）。

**目标环境**：本地异构测试网，执行层（EL）至少包含：Geth、Besu、Nethermind、Reth、Erigon、ethrex。

**范围**（V1.0 聚焦）：

- Discovery v4
- RLPx Handshake
- ETH Wire Protocol（基于已协商 capability）

**非目标**（本版本不承诺）：

- JSON-RPC 级别差分（除非用于辅助获取 enode/peer 信息）
- 完整区块同步/共识差分

## 2. 参考架构设计（必须复用 Geth devp2p 工具结构）

### 2.1 设计原则

- **复用优先**：CLI 结构、flags 组织、连接/握手逻辑必须参考并复用 `github.com/ethereum/go-ethereum/cmd/devp2p/internal` 的代码结构与实现风格。
- **可重复、可度量**：所有 fuzz case 必须可序列化落盘；所有对比必须有稳定键（输入 hash、目标节点标识、时间窗、握手参数）。
- **协议原始字节优先**：差分对比以原始入站/出站帧（或消息）为主，解码视作辅助视图，避免“解码器差异”掩盖真实差分。
- **安全边界可控**：工具默认不触发高风险行为（如无限连接洪泛）。攻击性能力通过显式开关启用，并具备速率限制。

### 2.2 仓库顶层结构（建议形态）

- `cmd/agent-devp2p/`：主二进制入口（CLI）。
- `internal/`：本项目内部包。
  - `internal/cli/`：命令注册、flags、输出格式。
  - `internal/targets/`：目标集合与连接配置（6 客户端）。
  - `internal/runner/`：差分执行器（并发、超时、重试、记录）。
  - `internal/corpus/`：样本/种子/最小化输入的存储与加载。
  - `internal/compare/`：差分对比与聚类策略。

注：以上为结构建议；实际落地时，命令与连接逻辑以 `cmd/devp2p/internal` 的拆包方式为基准进行映射。

### 2.3 “复用 devp2p/internal” 的具体落点

在实现 CLI 与网络连接时：

- 以 `cmd/devp2p/internal` 的模式提供：
  - 统一的 flag 解析与默认值
  - 基于 enode/enr 的目标解析
  - 连接拨号、RLPx 建链与 capability 协商的公共逻辑
- 在本项目中只新增“差分执行/记录/对比”能力；避免重写已存在的 devp2p 探测/握手工具代码。

### 2.4 数据流与组件职责

**输入**：

- 目标集合：6 个节点的 enode/enr 或 ip:port + 公钥。
- Fuzz Case：
  - 协议阶段：`discv4` / `rlpx` / `eth`
  - 发送方向：client->server
  - 负载：原始字节（frame 或 message 编码后的 bytes）
  - 连接参数：握手版本、capabilities、超时、是否允许非标准行为

**执行**（差分 Runner）：

- 为每个目标创建一次独立会话（session），按照同一条脚本：连接 →（可选）握手 → 发送 payload → 采集响应 → 关闭。
- 采集记录至少包含：
  - 连接阶段事件（拨号结果、握手结果、断连 reason、capabilities）
  - 入站/出站原始字节流摘要（hash + 长度 + 时间戳）
  - 入站消息序列（若可解码：code + 结构化视图）
  - 超时/错误分类（network error / protocol error / remote disconnect / local policy）

**对比**（差分 Comparator）：

- 以“同一输入、同一握手参数、同一时间窗”作为对比组。
- 主要差分维度：
  - 断连与原因（是否断连、断连码、时间点）
  - 响应消息的有无、数量、顺序、消息码
  - 响应负载的 hash 与长度
  - RTT/处理耗时分布（粗粒度即可）

### 2.5 Imposter Nodes（设计预留）

**定义**：Imposter Node 是一种“伪装/对抗型”节点能力：复用 `cmd/devp2p/internal` 的连接与协议栈，但在关键路径上绕过或放宽安全/一致性校验，从而允许发送更具攻击性的序列（例如畸形握手、非标准 capability、异常帧边界）。

**要求**：

- 必须通过显式开关启用（例如 `--imposter`）。
- 必须有节流（并发/速率/每目标最大会话数）。
- 需要在 Geth 源码修改点上保持“最小侵入 + 可维护”：尽量通过注入接口/选项实现，而不是散落的条件分支。

## 3. Feature Status Matrix（唯一状态源）

规则：所有功能默认 `[ ]`。只有收到 Auditor 的 `[AUDIT PASSED]` 信号后才能改为 `[x]`。

| Feature | 状态 | 说明 |
|---|---|---|
| Milestone 1: CLI 骨架（复用 devp2p/internal 结构） | [x] | 子命令、flags、输出格式、基本 help |
| Milestone 2: 连接管理（6 目标并发会话） | [ ] | 目标解析、拨号、超时、记录会话事件 |
| Milestone 3: RLPx 握手差分测试 | [ ] | 对 6 客户端执行同参握手并比对结果 |

## 4. 里程碑验收标准（V1.0）

### Milestone 1：CLI 骨架

- 提供可执行 `agent-devp2p` 二进制。
- 子命令至少覆盖：目标管理/连通性探测/握手测试（命名以 devp2p 工具风格为参考）。
- 输出支持 machine-readable（建议 JSONL）与人类可读两种模式。

### Milestone 2：连接管理

- 能对 6 个目标并发发起会话，并为每个目标产出独立记录。
- 超时、重试、错误分类稳定。
- 记录落盘具备稳定目录结构（按 run_id / case_id / target_id）。

### Milestone 3：RLPx 握手差分测试

- 在相同参数下对 6 客户端执行握手，输出差分摘要：成功/失败矩阵、断连码、capability 协商结果。
- 失败时保存最小可复现信息（目标、时间、握手参数、原始字节摘要）。

## 5. Git 治理（文档驱动开发）

- `SPEC.md` 是唯一规格来源；任何行为性变更必须先改 SPEC，再实现。
- 每次修改 `SPEC.md`，必须伴随一次单独的提交（commit message 需明确版本或变更点）。
