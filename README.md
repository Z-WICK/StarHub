# GitHub Stars Manager

一个用于管理 GitHub Stars 的项目，包含：
- **Desktop 客户端**（Vue + Electron）
- **API 服务**（Go）
- **部署配置**（Docker Compose + Nginx）

## 项目结构

```text
.
├── apps/
│   └── desktop/        # Vue + Electron 桌面端
├── services/
│   └── api/            # Go API 服务
├── deploy/
│   └── compose/        # Docker Compose 配置
├── scripts/            # 常用脚本（测试、发布等）
└── contracts/          # 接口/契约相关目录
```

## 环境准备

- Node.js 20+
- Go 1.22+
- Docker & Docker Compose（可选，用于本地一键启动依赖）

## 本地开发

### 1) 启动后端依赖（Postgres / Redis）

```bash
cd deploy/compose
cp .env.example .env
# 按需修改 .env 中的配置

docker compose up -d postgres redis
```

### 2) 启动 API 服务

```bash
cd services/api
go run ./cmd/server
```

### 3) 启动桌面端

```bash
cd apps/desktop
npm install
npm run electron:dev
```

## 测试

运行全量测试：

```bash
./scripts/test-all.sh
```

## 部署（Docker Compose）

```bash
cd deploy/compose
cp .env.example .env
# 填写真实配置后启动
docker compose up -d
```

## 说明

- `.env.example` 提供了本地开发参考配置。
- 请勿将真实密钥提交到仓库。