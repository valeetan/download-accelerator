# 下载加速代理（Go + Cloudflare 友好）

本项目提供一个最小可运行的下载加速代理服务：
- 访问首页输入源文件 URL
- 服务端尝试探测可下载性
- 生成带签名与过期时间（默认3小时）的代理下载链接
- 通过自有域名进行代理，可结合 Cloudflare CDN 缓存

注意：首版仅包含基础骨架与签名链接生成，实际代理转发与缓存适配会在后续步骤中完善。

## 快速开始

### 运行（本地）
```bash
make run
```
默认监听 `:8080`，浏览器打开 `http://localhost:8080`。

### 构建
```bash
make build
```
产物输出在 `bin/`。

### Docker
```bash
make docker
docker run --rm -p 8080:8080 \
  -e APP_ADDR=":8080" \
  -e APP_SIGNING_KEY="change-me" \
  download-accelerator:local
```

## 配置

支持 YAML + 环境变量 + 命令行参数（优先级：flag > env > yaml > 默认）。

- YAML：复制 `configs/config.yaml.example` 为 `configs/config.yaml` 并修改
  - `addr`: 监听地址，默认 `:8080`
  - `signingKey`: 签名密钥（务必修改）
  - `tokenTTL`: 有效期，默认 `3h`
  - `baseURL`: 外部可访问域名（用于生成绝对链接），如 `https://dl.example.com`

环境变量或命令行参数：
- `APP_ADDR`/`--addr`：监听地址，默认 `:8080`
- `APP_SIGNING_KEY`/`--signing-key`：签名密钥，必须设置为非默认值
- `APP_TOKEN_TTL`/`--token-ttl`：签名有效期，默认 `3h`
- `APP_BASE_URL`/`--base-url`：对外可访问的域名（用于生成绝对下载链接），如 `https://dl.example.com`

## Cloudflare 缓存建议
- 在 CDN 前使用自有域名，如 `dl.example.com`
- 代理路径建议采用稳定的路径模式，如 `/d/{nonce}` 避免过多变体
- 在响应头设置合适的 `Cache-Control`，并结合 CF Page Rules/Cache Rules 覆盖源站 `no-store`
- 对大文件使用 `range` 透传并确保 `Accept-Ranges` 支持，利于分片缓存
- 对不可缓存的鉴权参数放到签名校验中，尽量避免进缓存的 URL 携带真实源站私参

## 目录结构
```
internal/
  config/             # 配置加载（YAML + env + flag）
  controller/         # 控制器（HTTP层）
  service/            # 业务逻辑
  dao/                # 数据访问（占位，后续可接入）
  signer/             # 链接签名与校验
web/
  templates/          # HTML 模板
  static/             # 静态资源
main.go               # 程序入口（Gin）
configs/
  config.yaml.example # 示例配置
```

## 开发
- 代码风格：显式命名、早返回、错误优先处理
- 运行 `make fmt vet test` 保持质量
- 如需新增模块，优先放入 `internal/`

## 许可证
MIT
NULL
