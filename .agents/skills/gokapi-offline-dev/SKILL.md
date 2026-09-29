---
name: gokapi-offline-dev
description: Use when building, testing, verifying, or extending the Gokapi file-sharing server in this workspace (D:/desktop2/EMPTY/gok-filer) — a Go + Bootstrap 5.3 + html/template project. Covers how to bootstrap a Go toolchain offline (no system Go, use the aliyun mirror), the go:embed wasm placeholder trick needed to compile the webserver package, the `-tags test` build tag used by every test, the known pre-existing test failures on Windows, why `gofmt -l` flags every file (CRLF repo convention), and the four customisation surfaces added in this repo: file-ID mode (long/short), built-in accounts (admin/user), i18n (en + zh-CN resource files, per-language template sets, cookie-persisted switcher) and the liquid-glass theme (theme.css tokens + theme.js dark/light toggle). Trigger on: gokapi 构建、gokapi 测试、gokapi 模板、加语言、加文案、i18n、短 ID、内置账号、主题、液态玻璃、widget 圆角、编译 gokapi、go 环境没装.
agent_created: true
---

# Gokapi 离线构建 / 验证 / 定制

适用仓库：`D:/desktop2/EMPTY/gok-filer`（Go 1.26，Bootstrap 5.3.2，`html/template`，sqlite/redis 双后端）。

## 0. 先决条件：本机没有 Go

系统 PATH 里**没有 go**（`which go` 为空）。先用阿里云镜像装一份，不要尝试 go.dev / dl.google.com（curl 报 exit 35）。

```bash
mkdir -p /d/dev/tool/go126dl && cd /d/dev/tool/go126dl
# 用 mirrors.aliyun.com 列版本（sort -V 才是正确排序）：
curl -s "https://mirrors.aliyun.com/golang/" | grep -oE 'go1\.(2[0-9])\.[0-9]+\.windows-amd64\.zip' | sort -uV | tail
curl -sL -o go.zip "https://mirrors.aliyun.com/golang/go1.26.8.windows-amd64.zip"
unzip -q -o go.zip -d /d/dev/tool/     # -> /d/dev/tool/go/bin/go.exe
```

固定环境变量（每条命令都要带，或先 export）：

```bash
export GOROOT=/d/dev/tool/go GOPATH=/d/dev/tool/gopath GOCACHE=/d/dev/tool/gocache \
       GOTOOLCHAIN=local GOFLAGS=-mod=mod GOPROXY=https://goproxy.cn,direct
GO=/d/dev/tool/go/bin/go.exe
```

`GOTOOLCHAIN=local` 必须设：`go.mod` 写的是 `go 1.26.0`，否则 go 会尝试自己下载 toolchain 而卡住。
`GOPROXY=goproxy.cn` 必须设：直连 proxy.golang.org 不可用。

## 1. 编译 webserver 包必须先造 wasm 占位文件

`internal/webserver/Webserver.go` 里有 `//go:embed web/main.wasm` 和 `web/e2e.wasm`，这两个文件是 `go:generate` 产物、已进 `.gitignore`，仓库里**不存在**，所以裸跑 `go build ./...` 会报：

```
pattern web/e2e.wasm: no matching files found
```

临时造空占位（跑完**务必删掉**，否则会打出一个 wasm 为空的坏二进制）：

```bash
: > internal/webserver/web/main.wasm && : > internal/webserver/web/e2e.wasm
$GO build ./... && $GO test -tags test ./internal/webserver/ ...
rm -f internal/webserver/web/main.wasm internal/webserver/web/e2e.wasm
```

正式构建走 `makefile` / `go generate ./...`（会调 `build/go-generate/*.go` 里的真 wasm 编译）。

## 2. 测试：一切都要 `-tags test`

`internal/test`、`internal/test/testconfiguration` 带 `//go:build test`；不带 tag 会报 `build constraints exclude all Go files in internal/test`。

```bash
$GO test -tags test -count=1 ./internal/webserver/ ./internal/configuration/ ./internal/i18n/
```

**已知的本机（Windows）既有失败，与业务代码无关，不要试图修**：

| 失败 | 原因 |
|---|---|
| `internal/environment` `TestTempDir` | 断言 `TMPDIR==""`，而沙箱里 TMPDIR 有值。`TMPDIR="" go test` 就过 |
| `internal/webserver/ssl` `TestGenerateIfInvalidCert` | `os.Remove("test/ssl.crt")` 后仍读到 365 天证书。已在 pristine HEAD 复核过同样失败 |
| `internal/storage` `TestNewFile` panic | `remove test/data/uploadXXX: ... being used by another process`（Windows 文件锁） |
| `internal/storage/.../localstorage` | 路径分隔符断言 `test/` vs `test\` |
| `internal/storage/chunking` | 同上，文件占用 |

判断「是不是我改坏了」的标准做法：`git worktree add /d/dev/tool/gokapi-base HEAD`，在干净副本里跑同一个测试对比，比完 `git worktree remove --force`。

**测试会生成未跟踪产物**，收尾时清掉：`internal/**/test/`、`internal/storage/bigfile`、`internal/webserver/ssl/test/`。

## 3. gofmt 会报「所有文件都不合格」——是 CRLF，不是你的问题

本仓库工作区是 **CRLF**（含新建文件）。`gofmt -l` 会把连你都没碰过的文件全列出来。正确校验方式：先转 LF 再跑。

```bash
tr -d '\r' < internal/i18n/I18n.go > /tmp/f.go && $GO/bin/gofmt.exe -l /tmp/f.go
```

## 4. 四个定制面（本仓库已落地）

### 4.1 文件 ID 模式（long / short）

| 位置 | 作用 |
|---|---|
| `internal/environment/Environment.go` | 字段 `FileIdMode`(`GOKAPI_FILE_ID_MODE`, 默认 `long`)、`LengthShortId`(`GOKAPI_LENGTH_SHORT_ID`, 默认 8, min 5)；常量 `FileIdModeLong/Short`；方法 `UsesShortFileIds()` / `GetFileIdLength()`；`parseFlags` 里归一化（非法值回落 long） |
| `internal/storage/FileServing.go` | `createNewId()` 分支；`createUniqueFileId()` 用 `helper.GenerateRandomString` 生成后查 `database.GetMetaDataById` 查重，最多 `maxShortIdAttempts=8` 次，失败回落长 ID |

要点：短 ID 的唯一性靠**数据库查重**保证，随机强度靠 crypto/rand（`GenerateRandomString`）保证。新增类似需求时不要自己写 rand。

### 4.2 内置账号

| 位置 | 作用 |
|---|---|
| `internal/configuration/BuiltinUsers.go` | `InitializeBuiltinUsers()`，常量 `BuiltinAdminName/Password`、`BuiltinUserName/Password`；同名已存在则跳过（幂等，不覆盖密码）；密码走 `HashPassword`（Argon2id） |
| `cmd/gokapi/Main.go` | 调用顺序必须是 `setDeploymentPassword()` → `InitializeBuiltinUsers()` → `checkIfUserExists()` |

顺序原因：`setDeploymentPassword` 内部走 `EditSuperAdmin`，若库里已有用户但没有 superadmin 会直接退出；内置 admin 是 `UserLevelAdmin`（非 superadmin），不能抢在它前面建。
开关：`GOKAPI_CREATE_BUILTIN_USERS`（默认 true）。

### 4.3 国际化（en + zh-CN）

| 位置 | 作用 |
|---|---|
| `internal/i18n/I18n.go` | 语言表 `supportedLanguages`、`Init()` 读 embed 的 `locales/*.json`、`Translate/TranslateFormat`、`Normalize`（支持 `zh`→`zh-CN`、`en-US`→`en`）、`FromRequest`、`WriteLanguageCookie`、`Translator/TranslatorFormat` |
| `internal/i18n/locales/en.json` `.zh-CN.json` | **扁平 key→文案**，两份必须 key 完全一致 |
| `internal/webserver/Webserver.go` | `templateFolders map[string]*template.Template` —— **每种语言一套模板**，funcMap 里绑定该语言的 `tr`/`trf`/`availableLanguages`/`currentLanguage`。这样并发请求无共享可变状态（不要改成全局 currentLanguage，会 race）。`renderTemplate(w,r,name,data)` 取代原来的 `templateFolder.ExecuteTemplate`；`/setLanguage` 路由 + `setLanguage` 只接受 same-host Referer 做回跳 |
| `internal/webserver/web/templates/*.tmpl` | 文案写成 `{{ tr "section.key" }}`；`{{ template "pagename" "X" }}` 的 X **不要翻译**（是 JS 标识） |

改文案/加文案的标准流程：

1. 在 `en.json` 和 `zh-CN.json` **同时**加同一个 key。
2. 模板里用 `{{ tr "key" }}`。
3. 跑 key 一致性校验（脚本见 `scripts/check-i18n-keys.py`）：

```bash
python scripts/check-i18n-keys.py <repo-root>
```

它对不上会直接列出「模板用了但资源文件没有的 key」。注意 `tr` 的缺失 key 会**原样输出 key 字符串**（有 en 兜底），所以漏加不容易发现，必须靠这个脚本。

4. 模板语法校验（外部小工具，不污染仓库）：`scripts/parse-templates.go` 用 `template.FuncMap` 打桩 `ParseGlob` 一遍所有 `.tmpl`。改模板后一定要跑，否则只有运行时才炸。

语言持久化：Cookie `gokapi_language`（`i18n.CookieName`，一年），服务端渲染时用 `i18n.FromRequest(r)` 读取。

### 4.4 液态玻璃主题 + 深浅色

| 位置 | 作用 |
|---|---|
| `internal/webserver/web/static/css/theme.css` | 设计令牌（`--gk-radius-*` 圆角刻度、`--gk-space-*` 4px 栅格、`--gk-inset`）；内外圆角平行靠 `calc(outer - inset)`；`[data-bs-theme=dark|light]` 两套颜色变量；`.gk-toolbar` 悬浮工具栏 |
| `internal/webserver/web/static/js/theme.js` | 读写 `localStorage["gokapi-theme"]`，改 `<html data-bs-theme>`；`<head>` 里同步加载以免闪主题 |
| `internal/webserver/web/templates/html_header.tmpl` | 引用 `./css/theme.css`、`./js/theme.js`；渲染 `#gk-toolbar`（主题按钮 + 语言切换） |

**踩过的坑**：
- 不要给 `.card` / `.table-responsive` / `.modal-content` 加 `overflow:hidden`——会裁掉 Bootstrap 下拉菜单，并且 `.table-responsive` 会失去横向滚动。
- `.card-body` 的 `padding` **不要加 `!important`**，否则会覆盖 `p-3` / `p-5` 工具类。
- `theme.css` 的 `background.jpg` 相对路径是 `../assets/background.jpg`（文件在 `static/css/`，图在 `static/assets/`）。
- 主题切换按钮的图标用 `id="gk-theme-icon"`，`theme.js` 会改它的 className。
- **按钮组贴合问题**：theme.css 给所有 `.btn` 强制圆角后，Bootstrap `.btn-group` 的 -1px 负 margin + 压平圆角会失效，按钮呈「熔接」状。修法（2026-09-29 已落地）：`.btn-group { gap }` + `.btn-group > .btn { margin-left/right: 0 !important }` + 组内全部用 `--gk-radius-xs`，`.btn-toolbar` 用 `--gk-tool-gap`。
- **旧模板里的结构性 `<br>`**：大量模板直接在 `.card-body` 下用 `<br>` 排版，用 `.card-body > br { display:none }`（只隐藏直接子级，段落内的换行 br 保留）+ `.card-title/.card-text` 统一 margin 补偿。
- 管理页头部布局已从 `html_header.tmpl` 的 float 内联样式改为 theme.css 的 `.masthead .inner` flex 布局（换行/移动端居中）。
- `.card` 上的内联 `style="width: 80%"` 之类在移动端用 `@media (max-width:767px) .card { width:100% !important }` 覆盖（样式表 !important 可胜过内联非 important 声明）。
- **统一顶栏（2026-09-29 晚）**：`html_header.tmpl` 已重构为单个 `.gk-topbar`（sticky、负 margin 抵消 p-3 实现通栏）：品牌 `.gk-brand` 居左 + `.nav-masthead` 居中 + `#gk-toolbar.gk-controls`（主题/语言/用户徽章）居右。测试依赖 `id="gk-toolbar"`、`id="gk-theme-toggle"`、`class="gk-lang-link active"`，重构时必须保留。移动端 nav `order:3; flex-basis:100%` 横向滚动。
- **表头 min-width 的坑**：`th` 字号是 0.78rem，`min-width: 17em` 实际只有 ~230px；要让操作列按钮单行需用 `min-width: 19rem !important`（!important 才能压过模板 th 上的内联 min-width）。
- 系统名 `PublicName`：默认值在 `internal/configuration/Configuration.go`（现为 GrokFiler），运行实例在 `gokapi-config/config.json`；页脚 `html_footer.tmpl` 里的名字是硬编码（footer 拿不到 view 上下文），改品牌要三处同步。
- **退出登录在用户下拉里**（不在导航栏）：`.gk-user-menu`（badge）是 Bootstrap dropdown toggle。**坑**：disabled 表单控件会整体吞掉 click 事件（不冒泡到 document），Bootstrap「点外部关闭」收不到通知导致菜单挂住——`theme.js` 里用 mousedown 捕获阶段兜底关闭（`bootstrap.Dropdown.getOrCreateInstance(toggle).hide()`），Escape 同理加捕获监听。诊断手段：`elementFromPoint(x,y)` + `Dropdown.getInstance(toggle)._config.autoClose`。
- 导航链接靠左、品牌居左：`.gk-topbar .nav-masthead { flex: 0 1 auto; justify-content: flex-start }`。
- **横向滚动容器里固定搜索框**：DataTables 的 `.dataTables_filter`/`.dataTables_info` 在 `.table-responsive` 滚动容器内会跟着滚。sticky 钉边的前提是**包含块必须够宽**——wrapper 默认只有可视区宽，sticky 无移动空间；先 `.table-responsive .dataTables_wrapper { width: max-content; min-width: 100% }`，再对 filter 用 `position:sticky; right:0; width:fit-content; margin-left:auto`（info 对称用 left:0）。诊断法：`getBoundingClientRect()` 对比 filter 右缘与 scroller 右缘。
- **表格内下拉菜单被裁剪**：`.table-responsive` 的 overflow-x:auto 会连带纵向裁剪，最后一行的下拉菜单被切。方案：theme.js 用 **click 事件委托**（`[data-bs-toggle='dropdown']` + closest('.table-responsive')）+ `setTimeout(0)` 后把菜单改为 `position:fixed`（相对卡片定位——`.card` 的 backdrop-filter 是 fixed 的 containing block，坐标用 `cardRect` 换算），fixed 元素可脱离滚动容器裁剪。**教训**：Bootstrap 的 `shown.bs.dropdown` 等自定义事件在冒泡阶段监听收不到（合成事件 bubbles:true 能收到，导致调试误判）；换成 click 委托 100% 可靠。另配 scroll 捕获监听：表格滚动时立即关闭菜单（fixed 菜单不随表格移动）。

## 5. 一键验证清单

```bash
export GOROOT=/d/dev/tool/go GOPATH=/d/dev/tool/gopath GOCACHE=/d/dev/tool/gocache \
       GOTOOLCHAIN=local GOFLAGS=-mod=mod GOPROXY=https://goproxy.cn,direct
GO=/d/dev/tool/go/bin/go.exe
: > internal/webserver/web/main.wasm; : > internal/webserver/web/e2e.wasm   # 临时占位
$GO build ./... && echo BUILD OK
$GO test -tags test -count=1 ./internal/i18n/ ./internal/configuration/ ./internal/webserver/
rm -f internal/webserver/web/main.wasm internal/webserver/web/e2e.wasm
python scripts/check-i18n-keys.py .
```

文档生成器：新增 env 变量后跑它会把 `docs/advanced.rst` 的表格整体重排（列宽会变，属正常）。

```bash
cd build/go-generate && $GO run -tags tools updateEnvVariables.go
```

## 6. Docker Compose 部署（本机实战）

### 6.1 先改这三处，否则踩坑

| 文件 | 为什么 |
|---|---|
| `docker-compose.yaml` | 上游默认 `image: f0rc3/gokapi:latest`，拉的是**上游预构建镜像**，看不到本仓库改动。必须加 `build: {context: ., args: {GOPROXY: ...}}` 并从本地构建 |
| `Dockerfile` | 加 `ARG GOPROXY` + `ENV GOPROXY GOTOOLCHAIN=local`，否则容器内 `go generate ./...` 拉不到依赖 |
| `.dockerignore`（新建） | 不加的话 `COPY . /compile` 会带上 `gokapi-data/` `gokapi-config/`；容器一写数据就污染构建缓存，下次 build 全量重来 |

### 6.2 CRLF 会打断容器启动（本机高频坑）

git `autocrlf` 把 `dockerentry.sh` checkout 成 CRLF，shebang 变成 `#!/bin/sh\r`，内核去找名为 `/bin/sh\r` 的解释器 → 容器无限重启：

```
[FATAL tini (7)] exec /app/run.sh failed: No such file or directory
```

**诊断**：`head -1 dockerentry.sh | od -c` 看行尾是不是 `\r \n`；`git show HEAD:dockerentry.sh | od -c` 对比（git 里应是 LF）。

**修复（三件套一起做）**：
1. 新建 `.gitattributes`：`*.sh text eol=lf` + `Dockerfile text eol=lf`
2. 把工作区文件转成 LF（`python -c "..."` 或 `tr -d '\r'`）。转完 `git hash-object` 应与 `git rev-parse HEAD:<file>` 相同；`git status` 可能仍显示 `M`，那只是 stat 缓存，`git diff` 为空即无差异。
3. Dockerfile 里加兜底：`RUN sed -i 's/\r$//' /app/run.sh && chmod +x /app/run.sh`

### 6.3 网络

- `auth.docker.io` 偶发 `Bad Gateway`（拉基础镜像时 `failed to fetch anonymous token`）→ **先直接重试**，通常是瞬时故障。
- 本机需要 `golang:1.26.2-alpine` + `alpine:3.23`。可用加速源 `docker.m.daocloud.io` / `dockerproxy.net` / `docker.1ms.run`（`registry.cn-hangzhou.aliyuncs.com` 在本机不通）。
- 容器内 `apk add` 走 `dl-cdn.alpinelinux.org`，本机可用。

### 6.4 跳过 Setup 向导（否则首屏是未美化的向导页）

首次启动若 `config/config.json` 不存在，会进 Web 向导（`/setup`，用的是 `internal/configuration/setup/templates` 里那套**独立、未 i18n/未套主题**的模板）。想直接看到新版登录页，就预置配置：

1. 用仓库自带的 `cmd/seedconfig`（**必须放在模块内，`internal/...` 不能被外部模块 import**，所以它是个常驻命令而不是外部脚本）。在**宿主机**从仓库根跑，生成 `./gokapi-config/config.json` + `./gokapi-data/gokapi.sqlite`：

   ```bash
   GOKAPI_CONFIG_DIR=gokapi-config \
   GOKAPI_DATA_DIR=gokapi-data \
   SEED_DATABASE_URL="sqlite://gokapi-data/gokapi.sqlite" \
   go run ./cmd/seedconfig
   ```
2. 宿主与容器路径不同，必须改写 3 个字段：
   - `DataDir`: `gokapi-data` → `/app/data`
   - `DatabaseUrl`: `sqlite://gokapi-data/gokapi.sqlite` → `sqlite:///app/data/gokapi.sqlite`
   - `RedirectUrl`: 向导默认是**外部 URL**；写 `/index` 会让 index 页重定向到自己（死循环），demo 用 `/admin`
3. 对 sqlite 跑 `PRAGMA wal_checkpoint(TRUNCATE)`，避免留下 `-wal`/`-shm`。
4. 种子超管**故意不叫 `admin`**，这样内置账号 admin/admin1234、user/user1234 仍会在首次启动时被创建出来，方便演示。

`Encryption.Level = 0`（NoEncryption）时不需要 Cipher/Salt 任何字段，配置很容易手写。

### 6.5 验证清单

```bash
docker compose build && docker compose up -d
docker compose logs --tail=40          # 期望看到 "Created built-in user: admin/user"
docker inspect --format '{{.State.Health.Status}}' gokapi   # healthy
curl -s http://localhost:53842/index   # meta refresh -> /admin
curl -s -b 'gokapi_language=zh-CN' http://localhost:53842/login | grep 忘记密码
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:53842/css/theme.css   # 200
```

登录后（cookie jar）验证短 ID 端到端：

```bash
CSRF=$(curl -s -c /tmp/cj.txt http://localhost:53842/login | grep -oE 'value="[^"]+"' | head -1 | sed 's/value="//;s/"//')
curl -s -b /tmp/cj.txt -c /tmp/cj.txt -X POST http://localhost:53842/login \
  -d username=admin -d password=admin1234 --data-urlencode "csrf-token=$CSRF"
KEY=$(curl -s -b /tmp/cj.txt -H 'permission: PERM_UPLOAD' http://localhost:53842/auth/token | python -c 'import sys,json;print(json.load(sys.stdin)["key"])')
echo hi > /tmp/f.txt
curl -s -X POST http://localhost:53842/api/files/add -H "apikey: $KEY" -F file=@/tmp/f.txt
# -> {"FileInfo":{"Id":"HbwhVuiF",...}}  8 位即 short 模式生效
```

注意：默认每个上传只能下载 1 次，用 `downloadFile?id=...` 拉过一次后该文件就从 `/admin` 列表消失了（不是 bug）。

