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
