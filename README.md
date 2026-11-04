# Downlink

用 MyGo 构建的桌面程序，把 tdl 的登录与下载放进图形界面。程序本身不实现 Telegram
协议，所有网络操作都由你指定的 tdl 可执行文件完成。

## 准备

1. 从 [tdl 发行版](https://github.com/iyear/tdl/releases) 下载对应平台的二进制
   （macOS arm64 用 `tdl_MacOS_arm64`，Windows 用 `tdl.exe`）。
2. 打开程序的「设置」，把「tdl 可执行文件」指向它，按回车。留空时程序会依次在 PATH、
   应用资源目录、可执行文件所在目录和工作目录下自动查找。

二进制的路径保存在设置里，换机器或换平台时在界面上改一次即可。

## 使用

- 「下载」页粘贴一个或多个消息链接（每行一个），选好保存位置后点「开始下载」。
  链接可以是公开频道、私有频道或 `tg://openmessage` 形式；「粘贴」从剪贴板填入，
  「清空」清掉整栏。「下载选项」窗口里是线程数、并发任务、续传方式（继续或重新开始）、
  跳过同名文件、改写扩展名、按对话分组、Takeout、扩展名筛选和文件名模板。
- 「登录 Telegram」在应用内完成 `tdl login -T code` 的交互：填手机号、验证码、
  两步验证密码。会话写进 tdl 自己的数据目录（默认 `~/.tdl`），和应用外使用的
  tdl 共用；本机已经有会话时，第一次启动会自动选中那个命名空间。
- 下载过程中卡片显示总进度、速度、剩余时间和每个文件的进度，可以取消；结束后可以
  打开目录。
- 「记录」页保存最近 40 次下载，可以打开目录、把链接放回下载页重新开始，或者删掉这条。
- 「设置」页可以改 tdl 可执行文件、命名空间、代理、下载目录、线程数与并发任务数。
- 关掉窗口就退出程序，正在进行的下载会先收到中断信号再被结束，留下的部分文件可以
  用「继续下载」接着下。

## 构建

```sh
go tool mygo build -skip-dmg
```

产物在 `build/<os>-<arch>/Downlink.app`。不加 `-skip-dmg` 时会尝试打包 DMG。`mygo.json`
里的 `icon` 指向 `icon.png`，构建时由 MyGo 转成 `AppIcon.icns` 放进应用包。

窗口用系统的玻璃材质（`mygo.VibrancyGlass`）：侧边栏透出桌面，内容区保持不透明的底色，
弹窗与下载卡片是 `plugins/glass` 画的玻璃，macOS 26 及以上用系统的液态玻璃，
更早的系统与 Windows、Linux 上由 MyGo 自己绘制。按钮一律是圆角的 chip 底色，不带描边。

## 测试

```sh
CGO_ENABLED=0 go test ./...
```

跑真实二进制的端到端用例需要伪终端与到 Telegram 的网络：

```sh
TDL_UI_E2E=1 TDL_UI_E2E_PROXY=http://127.0.0.1:7890 CGO_ENABLED=0 go test -run E2E -v .
```

这些用例把 `HOME` 指到临时目录，不会碰你自己的 `~/.tdl`。要真的下载一个文件（用你
已经登录好的会话）：

```sh
TDL_UI_E2E=1 TDL_UI_E2E_LIVE=1 TDL_UI_E2E_NS=<命名空间> TDL_UI_E2E_DIR=$PWD/.scratch/live \
  TDL_UI_E2E_PROXY=http://127.0.0.1:7890 CGO_ENABLED=0 go test -run E2ELive -v .
```
