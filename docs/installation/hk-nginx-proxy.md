# 香港 Nginx 反代美西 ALB 部署指南

本文档说明如何在 **业务 EC2 全部保留在美西** 的前提下，于香港部署 Nginx 反向代理，为大陆用户提供更近的 API 入口。

> 适用场景：美西已有 `1 × ALB + 2 × EC2 + 1 × DB`，香港仅做 Proxy，不运行 new-api、不连接数据库。

---

## 架构说明

```text
大陆用户
  │
  ▼
hk.faceapi.ai          ← 香港 Nginx（仅反代）
  │
  ▼ HTTPS 回源
faceapi.ai          ← 美西 ALB
  │
  ├─► 美西 EC2 #1（new-api）
  └─► 美西 EC2 #2（new-api）
        │
        ▼
      美西 DB / Redis
```


| 组件          | 位置  | 职责                 |
| ----------- | --- | ------------------ |
| Nginx Proxy | 香港  | TLS 终结、反向代理、流式透传   |
| ALB + EC2   | 美西  | 运行 new-api（master） |
| DB / Redis  | 美西  | 数据与缓存（香港不直连）       |


**用户用法：**


| 用途                | 域名示例                         |
| ----------------- | ---------------------------- |
| 大陆用户 API Base URL | `https://hk.faceapi.ai` |
| 管理后台 / 海外直连       | `https://faceapi.ai` |


令牌、模型名与美西完全一致，仅更换 Base URL。

---



## 前置条件


| 项目   | 要求                                        |
| ---- | ----------------------------------------- |
| 美西   | ALB 已对外提供 HTTPS，业务正常                      |
| 美西域名 | 已有正式证书，例如 `faceapi.ai`            |
| 香港机器 | 1 台轻量云主机 / VPS（建议 ≥ 1 核 1G，Ubuntu 22.04+） |
| 香港域名 | 已解析或可解析到香港机器公网 IP，例如 `hk.faceapi.ai` |
| 权限   | 可配置美西 ALB / 安全组，可申请或签发 TLS 证书             |


> 说明：香港机器只做 Proxy，规格可远小于业务 EC2。生产建议至少 2 核以便扛并发与 TLS。

---



## 步骤一：准备美西侧放行



### 1. 记录香港 Proxy 出口 IP

在香港机器上执行：

```bash
curl -4 ifconfig.me
```

记下公网 IPv4，下文记为 `HK_PROXY_IP`。 18.162.47.156

### 2. 收紧美西 ALB 安全组（推荐）

美西 ALB 入站建议：


| 协议  | 端口  | 来源               | 说明             |
| --- | --- | ---------------- | -------------- |
| TCP | 443 | `HK_PROXY_IP/32` | 仅允许香港 Proxy 回源 |
| TCP | 443 | 你的运维 IP（可选）      | 便于直接排查美西       |


若暂时需要海外用户直连美西域名，可保留 `0.0.0.0/0:443`，但香港方案本身不依赖公网任意来源。

### 3. 确认美西健康检查与流式超时

- ALB 目标组健康检查：`/api/status`（或你现有路径）
- ALB / Target 空闲超时建议 ≥ `3600` 秒（流式对话较长时尤其重要）
- new-api 侧 `STREAMING_TIMEOUT` 按现网配置保持即可（常见 `300`）

---



## 步骤二：安装香港 Nginx

以 Ubuntu 为例：

```bash
sudo apt update
sudo apt install -y nginx curl
sudo systemctl enable --now nginx
nginx -v
```

开放防火墙（若启用 ufw）：

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

云厂商安全组同步放行：`80`、`443` 入站。

---



## 步骤三：配置 TLS 证书

任选其一。

### 方式 A：Let's Encrypt（推荐）

先将 `hk.faceapi.ai` A 记录指向香港机器公网 IP，再执行：

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d hk.faceapi.ai
```

按提示完成；Certbot 会自动写入 Nginx 并配置续期。

### 方式 B：已有证书

将证书放到例如：

```text
/etc/nginx/ssl/hk.faceapi.ai/fullchain.pem
/etc/nginx/ssl/hk.faceapi.ai/privkey.pem
```

并设置权限：

```bash
sudo mkdir -p /etc/nginx/ssl/hk.faceapi.ai
sudo chmod 600 /etc/nginx/ssl/hk.faceapi.ai/privkey.pem
```

### 方式 C：暂不使用证书（纯 HTTP，临时 / 测试）

如果你**暂时没有证书、也不想申请**，可以先用纯 HTTP 跑通链路。

> ⚠️ **安全提示**：此方式下「客户端 → 香港」为明文传输，`Authorization: Bearer sk-xxx` 令牌会以明文经过公网，存在被窃取风险。**仅建议用于测试或临时验证**，正式对外请尽快切换到方式 A（Let's Encrypt，免费）。
>
> 注意：即使前端是 HTTP，**香港 → 美西的回源仍为 HTTPS**，回源段依旧加密。

完整配置见文末[附录：纯 HTTP 快速部署](#附录纯-http-快速部署无证书临时测试)。

---



## 步骤四：编写 Nginx 反代配置

创建站点配置：

```bash
sudo nano /etc/nginx/sites-available/hk-new-api-proxy.conf
```

写入以下内容（**请替换占位域名**）：

```nginx
# 美西 ALB 上游（建议填美西域名，便于证书校验）
upstream us_new_api {
    server faceapi.ai:443;
    keepalive 64;
}

# HTTP → HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name hk.faceapi.ai;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name hk.faceapi.ai;

    # Certbot 管理时可改用其生成的 ssl_certificate 路径
    ssl_certificate     /etc/nginx/ssl/hk.faceapi.ai/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/hk.faceapi.ai/privkey.pem;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:10m;
    ssl_protocols TLSv1.2 TLSv1.3;

    # 与 new-api MAX_REQUEST_BODY_MB 等限制对齐，可按需调大
    client_max_body_size 128m;

    # 真实客户端 IP（若前面还有 CDN，再按实际调整）
    real_ip_header X-Forwarded-For;
    real_ip_recursive on;

    location / {
        proxy_pass https://us_new_api;
        proxy_http_version 1.1;

        # Host 必须与美西证书 / 应用期望域名一致
        proxy_set_header Host faceapi.ai;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header Connection "";

        # ===== 流式 / SSE 关键：禁止缓冲 =====
        proxy_buffering off;
        proxy_cache off;
        proxy_request_buffering off;
        chunked_transfer_encoding on;
        gzip off;

        # 长连接与流式超时（秒）
        proxy_connect_timeout 10s;
        proxy_send_timeout 3600s;
        proxy_read_timeout 3600s;

        # 上游 TLS
        proxy_ssl_server_name on;
        proxy_ssl_name faceapi.ai;
        # 若美西使用正规公网证书，保持默认校验即可
        # proxy_ssl_verify on;
    }

    # 可选：简单健康检查（仅探活本机 Nginx）
    location = /nginx-health {
        access_log off;
        default_type text/plain;
        return 200 "ok\n";
    }
}
```

启用配置：

```bash
sudo ln -sf /etc/nginx/sites-available/hk-new-api-proxy.conf /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```



### 配置项说明


| 配置                            | 作用                               |
| ----------------------------- | -------------------------------- |
| `proxy_buffering off`         | 避免 SSE/流式被 Nginx 攒包，导致首字延迟或“假卡死” |
| `proxy_request_buffering off` | 大请求体尽快回源，降低内存占用                  |
| `proxy_read_timeout 3600s`    | 覆盖长流式生成                          |
| `Host faceapi.ai`     | 与美西 ALB 证书、虚拟主机一致                |
| `keepalive`                   | 复用到美西的 HTTPS 连接，降低握手开销           |


---



## 步骤五：DNS 配置


| 主机记录     | 类型                  | 值         | 说明       |
| -------- | ------------------- | --------- | -------- |
| `hk.api` | A                   | 香港机器公网 IP | 大陆用户入口   |
| `us.api` | A/CNAME / ALB Alias | 美西 ALB    | 管理端与海外直连 |


TTL 可先设 60–300 秒，验证完成后再调高。

---



## 步骤六：验证



### 1. 香港入口探活

```bash
curl -sS https://hk.faceapi.ai/nginx-health
curl -sS https://hk.faceapi.ai/api/status
```

期望：`nginx-health` 返回 `ok`；`/api/status` 返回 new-api 的 JSON（`success: true`）。

### 2. 非流式 API

```bash
curl -sS https://hk.faceapi.ai/v1/models \
  -H "Authorization: Bearer sk-你的令牌"
```



### 3. 流式 API（重点）

```bash
curl -N https://hk.faceapi.ai/v1/chat/completions \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role":"user","content":"用一句话介绍你自己"}]
  }'
```

期望：终端 **逐段** 出现 `data: {...}`，而不是长时间无输出后一次性刷出。

若整段结束才输出，几乎一定是缓冲未关（检查是否还有外层 CDN/WAF 开启了 response buffering）。

### 4. 对比延迟（可选）

```bash
# 直连美西
curl -o /dev/null -s -w "US TTFB: %{time_starttransfer}\n" https://faceapi.ai/api/status

# 经香港
curl -o /dev/null -s -w "HK TTFB: %{time_starttransfer}\n" https://hk.faceapi.ai/api/status
```

在大陆网络下，香港入口的 TTFB 通常更稳、更低。

---



## 步骤七：给用户的接入说明

将文档/控制台中的 Base URL 改为香港域名，例如：

```text
https://hk.faceapi.ai
```

OpenAI 兼容示例：

```bash
export OPENAI_BASE_URL=https://hk.faceapi.ai/v1
export OPENAI_API_KEY=sk-你的令牌
```

管理后台仍建议使用：

```text
https://faceapi.ai
```

---



## 高可用（可选）

单机 Proxy 足够验证与中小流量。需要更高可用时：

1. 香港再开 1 台 Nginx，配置相同
2. 前面加香港 SLB / NLB，或用 DNS 轮询
3. 两台机器均做证书（或挂载统一证书）
4. 美西 ALB 安全组放行两台 Proxy 的出口 IP

---



## 运维建议



### 日志

```bash
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```



### 常见调优


| 场景          | 建议                                                            |
| ----------- | ------------------------------------------------------------- |
| 并发高         | 增大 `worker_connections`；适当提高 `upstream keepalive`             |
| 上传/长上下文 413 | 同步调大 `client_max_body_size` 与美西 new-api `MAX_REQUEST_BODY_MB` |
| 流式中断        | 检查 ALB idle timeout、Nginx `proxy_read_timeout`、客户端超时          |
| 限流按 IP      | 确认美西能读到 `X-Forwarded-For`；必要时在 new-api / ALB 侧信任香港 Proxy      |




### 安全

- 定期升级 Nginx / OS
- 美西 ALB 尽量只放行香港 Proxy IP
- 证书自动续期：`sudo certbot renew --dry-run`
- 不要在香港机器上存放 DB 密码或 new-api 环境变量

---



## 故障排查


| 现象                        | 可能原因               | 处理                                                   |
| ------------------------- | ------------------ | ---------------------------------------------------- |
| `502 Bad Gateway`         | 香港到美西不通 / 安全组未放行   | 在香港 `curl -vI https://faceapi.ai/api/status` |
| `SSL certificate problem` | `Host` / SNI 与证书不符 | 检查 `proxy_set_header Host` 与 `proxy_ssl_name`        |
| 流式不实时                     | 缓冲未关闭，或前面还有 CDN 缓冲 | 确认 `proxy_buffering off`；排查外层加速产品                    |
| 超时断开                      | 超时过短               | 调大 Nginx / ALB idle timeout                          |
| 状态页通但 API 401             | 令牌问题，与 Proxy 无关    | 用美西域名对比同一令牌                                          |
| 大陆访问香港仍慢                  | 运营商路由差             | 换香港线路/厂商，或叠加国内优质 BGP                                 |


---



## 与「香港跑 slave」的差异


| 项目             | 本方案（Nginx Proxy） | 香港 new-api slave |
| -------------- | ---------------- | ---------------- |
| 美西 EC2         | 全部保留             | 全部保留             |
| 香港是否跑 new-api  | 否                | 是                |
| 是否连美西 DB/Redis | 否                | 是                |
| 配置复杂度          | 低                | 较高               |
| 加速效果           | 改善入口与跨境质量        | 通常更好             |


本方案目标是：**在不改美西应用拓扑的前提下，尽快为大陆用户提供香港入口。**

---



## 检查清单（上线前）

- [ ] `hk.faceapi.ai` 已解析到香港 IP
- [ ] HTTPS 证书有效
- [ ] `nginx -t` 通过且已 reload
- [ ] `/api/status` 经香港可访问
- [ ] 流式 `/v1/chat/completions` 可逐段输出
- [ ] 美西 ALB 已放行香港出口 IP（若已收紧）
- [ ] 用户文档 Base URL 已更新为香港域名
- [ ] 管理后台仍指向美西域名

---



## 附录：一键变量替换示例

部署前可统一替换：

```text
faceapi.ai  →  你的美西域名
hk.faceapi.ai  →  你的香港域名
```

完整回源探测命令：

```bash
# 香港机器上执行：确认能直连美西
curl -sS -o /dev/null -w "%{http_code} %{time_starttransfer}\n" \
  https://faceapi.ai/api/status
```

返回 `200` 且延迟可接受后，再启用对外香港域名。

---



## 附录：纯 HTTP 快速部署（无证书，临时/测试）

本附录给出一套**不使用证书**的最小可跑配置，方便在没有 TLS 证书时先验证「香港反代美西」链路是否通。

- 美西域名：`faceapi.ai`
- 香港域名：`hk.faceapi.ai`

> ⚠️ **安全提示**：客户端到香港为明文 HTTP，令牌（`Authorization`）会明文经过公网。**请仅用于测试 / 临时验证**，正式对外务必切到 [方式 A：Let's Encrypt](#方式-alets-encrypt推荐)（免费、自动续期）。
>
> 香港 → 美西的回源仍是 HTTPS，回源段依旧加密。

### 1. 前置

- `hk.faceapi.ai` 的 A 记录已指向香港机器公网 IP
- 香港机器安全组 / 防火墙放行 `80` 入站
- 美西 ALB 已放行香港出口 IP 的 `443`（回源用）

### 2. Nginx 配置（纯 HTTP）

```bash
sudo nano /etc/nginx/sites-available/hk-new-api-proxy-http.conf
```

写入：

```nginx
# 美西 ALB 上游（回源仍走 HTTPS）
upstream us_new_api {
    server faceapi.ai:443;
    keepalive 64;
}

server {
    listen 80;
    listen [::]:80;
    server_name hk.faceapi.ai;

    # 与 new-api MAX_REQUEST_BODY_MB 等限制对齐，可按需调大
    client_max_body_size 128m;

    # 真实客户端 IP（若前面还有 CDN，再按实际调整）
    real_ip_header X-Forwarded-For;
    real_ip_recursive on;

    location / {
        proxy_pass https://us_new_api;
        proxy_http_version 1.1;

        # Host 必须与美西证书 / 应用期望域名一致
        proxy_set_header Host faceapi.ai;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        # 前端为 HTTP，回源标记为 https 以便美西侧正确识别协议
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header Connection "";

        # ===== 流式 / SSE 关键：禁止缓冲 =====
        proxy_buffering off;
        proxy_cache off;
        proxy_request_buffering off;
        chunked_transfer_encoding on;
        gzip off;

        # 长连接与流式超时（秒）
        proxy_connect_timeout 10s;
        proxy_send_timeout 3600s;
        proxy_read_timeout 3600s;

        # 上游 TLS（回源到美西 ALB）
        proxy_ssl_server_name on;
        proxy_ssl_name faceapi.ai;
        # 若美西使用正规公网证书，保持默认校验即可
        # proxy_ssl_verify on;
    }

    # 可选：简单健康检查（仅探活本机 Nginx）
    location = /nginx-health {
        access_log off;
        default_type text/plain;
        return 200 "ok\n";
    }
}
```

启用配置：

```bash
sudo ln -sf /etc/nginx/sites-available/hk-new-api-proxy-http.conf /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

### 3. 验证（HTTP）

```bash
# 本机 Nginx 探活
curl -sS http://hk.faceapi.ai/nginx-health

# 经香港访问 new-api 状态
curl -sS http://hk.faceapi.ai/api/status

# 流式（重点：应逐段输出 data:）
curl -N http://hk.faceapi.ai/v1/chat/completions \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role":"user","content":"用一句话介绍你自己"}]
  }'
```

### 4. 用户接入（HTTP）

```bash
export OPENAI_BASE_URL=http://hk.faceapi.ai/v1
export OPENAI_API_KEY=sk-你的令牌
```

> 部分客户端 / SDK 默认强制 HTTPS，遇到报错时需手动允许 HTTP，或直接升级到方式 A 使用 HTTPS。

### 5. 后续升级到 HTTPS（推荐）

拿到证书前只用 HTTP 验证；验证通过后升级到免费证书：

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d hk.faceapi.ai
```

Certbot 会自动为上面这份配置补上 `443` 监听与证书，并配置自动续期。之后把用户的 Base URL 改回 `https://hk.faceapi.ai/v1` 即可。

---



## 附录：用 CloudFront 替代香港 Nginx（AWS 原生方案）

如果你不想自己维护香港这台 Nginx（机器、证书续期、调优），可以用 **AWS CloudFront** 作为托管替代品：由 CloudFront 做 TLS 终结、就近接入、回源到美西 ALB，并透传流式响应。

> 适用判断：优先看你的**流式时长**。若最长单次流式在 CloudFront 超时上限内（见下文），CloudFront 更省运维；若需要**小时级超长流式**或对缓冲/超时要求极致可控，仍建议保留香港 Nginx。

### 与香港 Nginx 的能力对比


| 维度 | 香港 Nginx（本文主方案） | CloudFront |
| ------------------ | -------------------------- | ------------------------------- |
| 就近入口 | 香港单点 | 全球边缘（含香港/亚太），大陆用户自动就近 |
| TLS 终结 | 自签 / Let's Encrypt，需自动续期 | ACM 免费证书，自动续期 |
| 回源 Host 改写 | `proxy_set_header Host` | Origin 的 **Origin Domain / Host header** |
| 流式 SSE 透传 | `proxy_buffering off` 完全可控 | 需关缓存 + 用 CachingDisabled，默认支持分块透传 |
| 流式最大时长 | `proxy_read_timeout 3600s` | 受 origin **response timeout** 限制（默认 30s，可提额，通常上限约 180s） |
| 运维成本 | 自己维护机器/证书/调优 | 全托管，无机器 |
| 源站 IP 收敛 | 放行香港 Proxy 出口 IP | 放行 CloudFront 托管前缀列表（`com.amazonaws.global.cloudfront.origin-facing`） |
| 成本模型 | VPS 固定月租 | 按流量 + 请求计费 |


### 配置要点（控制台 / CLI 通用思路）

1. **创建 Distribution，源站指向美西 ALB**
   - Origin Domain：`faceapi.ai`（美西 ALB 域名）
   - Protocol：**HTTPS only**，回源端口 `443`
   - 若应用按 `Host` 区分虚拟主机/证书校验，设置 Origin 的自定义 **Host header** 或确保 SNI 与美西证书一致（等价于 Nginx 的 `proxy_set_header Host faceapi.ai`）

2. **绑定香港对外域名 + ACM 证书**
   - Alternate domain name（CNAME）：`hk.faceapi.ai`
   - 证书：在 **us-east-1** 区申请/导入 ACM 证书（CloudFront 只认 us-east-1 的证书）
   - DNS 把 `hk.faceapi.ai` CNAME 到 CloudFront 分配的 `xxxx.cloudfront.net`

3. **关闭缓存，保证流式与鉴权正确**（最关键）
   - Cache policy：使用 **`CachingDisabled`**（等价于 Nginx `proxy_cache off`）
   - Origin request policy：转发**全部 Header（含 `Authorization`）、Query String、Cookie**（用 `AllViewer` 或自定义，务必包含 `Authorization`，否则令牌丢失导致 401）
   - 允许方法：`GET/HEAD/OPTIONS/PUT/POST/PATCH/DELETE`
   - 关闭压缩（`Compress = No`，等价于 Nginx `gzip off`，避免干扰 SSE）

4. **调大回源超时以覆盖流式**
   - Origin 的 **Response timeout / Keep-alive timeout** 调到最大（默认 30s，可向 AWS 申请提额，通常上限约 180s）
   - ⚠️ 这是 CloudFront 相对 Nginx `3600s` 的**主要短板**：单次流式若超过该上限会被中断。长思考/长生成场景务必先压测。

5. **收紧美西 ALB 来源（可选，等价于本文步骤一）**
   - ALB 安全组不再放行香港固定 IP，改为放行 CloudFront 托管前缀列表 `com.amazonaws.global.cloudfront.origin-facing`
   - 或在 ALB 上校验自定义 header（CloudFront 注入密钥 header，ALB 规则校验），防止绕过 CloudFront 直连源站

### 验证（与步骤六一致）

```bash
# 就近入口探活
curl -sS https://hk.faceapi.ai/api/status

# 流式（重点：应逐段输出 data:，而非结束才一次性刷出）
curl -N https://hk.faceapi.ai/v1/chat/completions \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "stream": true,
    "messages": [{"role":"user","content":"用一句话介绍你自己"}]
  }'
```

若流式变成"结束才一次性输出"，多半是**缓存/压缩未关**（检查 Cache policy 是否为 `CachingDisabled`、`Compress` 是否关闭）；若长流式**中途断开**，多半是**回源 response timeout 太短**，需提额或改回香港 Nginx 方案。

### 选型建议（一句话）

- **省运维、流式不超上限** → 用 **CloudFront**，无需香港机器与证书维护。
- **需要小时级超长流式 / 缓冲与超时完全可控** → 保留**香港 Nginx**（本文主方案）。