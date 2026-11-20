# 自建产物服务器配置（httpd / nginx / Caddy）

本目录给出三款静态文件服务器的**最小配置**，用于托管符合
[`CUSTOM_URL_SPEC.md`](../reference/CUSTOM_URL_SPEC.md) 的 FFmpeg 产物目录。

## 共同原则（为什么这样配）

1. **只提供静态文件**：规范要求服务器不做任何脚本、rewrite、JSON/manifest 解析。
2. **不开目录列表**：客户端通过 `snapshots/latest.txt` + 固定命名规则推导文件名，不依赖目录浏览。
   - 实测：nginx `autoindex off` → 目录返回 403；httpd `Options -Indexes` → 404；Caddy `file_server`（无 `browse`）→ 404。
3. **不可变产物**：发布后不得覆盖历史稳定版/快照文件；修复请发新版本号或新时间戳。此为发布纪律，客户端不校验。

## 配置来源（官方文档，非第三方教程）

| 服务器 | 配置项 | 官方文档来源 | 为什么这样写 |
|---|---|---|---|
| nginx | `autoindex off;` | <https://nginx.org/en/docs/http/ngx_http_autoindex_module.html> — 语法 `autoindex on\|off`，默认 `off` | 显式写出默认值，强调不开目录列表；`try_files` 保证未知路径 404 |
| Apache httpd | `Options -Indexes` | <https://httpd.apache.org/docs/2.4/mod/core.html#options> — `Indexes` 生成目录列表，`-Indexes` 只关闭该项而保留其他继承选项 | 不用 `Options None`（会关闭 FollowSymLinks 等）；`AllowOverride None` 防止 `.htaccess` 重开索引 |
| Caddy | `file_server`（不带 `browse`） | <https://caddyserver.com/docs/caddyfile/directives/file_server> — `browse` 才启用目录列表，默认关闭 | 不写 `browse` 即不列目录；`auto_https off` 仅用于本地 HTTP 实测 |

官方安装方式（实测所用，均为**官方最新稳定版**）：
- httpd 2.4.71：源码下载 <https://httpd.apache.org/download.cgi>（SHA256 与官方 `.sha256` 校验一致），按官方 `./configure && make && make install` 编译。
- nginx 1.30.7：nginx.org 官方 stable 源 <https://nginx.org/en/linux_packages.html>（repo 文件见 `nginx.repo`）。
- Caddy 2.11.9：Caddy 官方 COPR `@caddy/caddy`（<https://caddyserver.com/docs/install>，RHEL/CentOS 段落；Cloudsmith 只用于 Debian 系）。

## 实测记录（Linux VM，AlmaLinux 9.9 x86_64）

三款服务器分别监听 18080/18081/18082，托管相同的规范目录结构：

- `GET /ffmpeg/snapshots/latest.txt` → `20261120`（三款均 200，内容为一行日期）
- `GET /ffmpeg/snapshots/20261120/checksum.sha256` → 200
- `GET /ffmpeg/snapshots/`（目录）→ 403（nginx）/ 404（httpd、Caddy），目录列表关闭
- OneMoreFFmpeg `resolve --provider <id> --target linux64` → 正确生成
  `ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz` 并从 `checksum.sha256` 取到摘要
- 稳定版 `--release-line 8.1.0` → 正确生成 `ffmpeg-8.1.0-gpl-stable-linux-amd64-static.tar.xz`
- `download --release <pin.json>` → 下载后 SHA-256 校验通过并原子落盘

## 文件

- `nginx.conf` — nginx 静态服务配置
- `httpd.conf` — Apache httpd 2.4 配置
- `Caddyfile` — Caddy 配置
- `nginx.repo` — nginx.org 官方 stable 源（RHEL/CentOS 9）
