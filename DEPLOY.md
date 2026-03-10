# 部署说明

## 打包后端

```bash
build-backend.bat
```

打包后会生成 `server` 可执行文件（Linux amd64）。

---

## Linux 服务器部署

### 1. 上传文件

将以下文件上传到服务器 `/opt/deep-research/` 目录：

```
server                    # 可执行文件
src/configs/config.yaml   # 配置文件
.env.example              # 环境变量模板
```

### 2. 安装依赖

```bash
# PostgreSQL
sudo apt install postgresql postgresql-contrib

# Redis (可选)
sudo apt install redis-server
```

### 3. 配置环境

```bash
cd /opt/deep-research
cp .env.example .env
nano .env  # 编辑配置
```

**必填项：**
- `DEEPSEEK_API_KEY` 或 `ZHIPU_API_KEY`
- `DB_PASSWORD` - PostgreSQL 密码
- `JWT_SECRET` - 随机密钥

### 4. 配置 config.yaml

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: "${DB_PASSWORD:}"
  db_name: go_deep_research

server:
  port: 8080
```

### 5. 创建 systemd 服务

```bash
sudo nano /etc/systemd/system/deep-research.service
```

内容：

```ini
[Unit]
Description=AI Research Platform
After=network.target postgresql.service

[Service]
Type=simple
WorkingDirectory=/opt/deep-research
ExecStart=/opt/deep-research/server -config /opt/deep-research/configs/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 6. 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable deep-research
sudo systemctl start deep-research

# 查看状态
sudo systemctl status deep-research

# 查看日志
sudo journalctl -u deep-research -f
```

### 7. 验证部署

```bash
curl http://localhost:8080/api/v1/health
```

---

## 默认管理员

| 字段 | 默认值 |
|------|--------|
| 邮箱 | admin@example.com |
| 用户名 | admin |
| 密码 | admin123 |

> 生产环境请修改 `.env` 中的管理员配置！
