# 大鼠

“大鼠”是一个运行在 Windows 桌面上的透明 Codex 状态宠物。程序只读监听本机
Codex 会话，并在待机、悬停、拖动、工作、无聊、权限请求、完成、失败和退出时
播放对应动画。

![九种状态预览](docs/contact-sheet.png)

## 功能

- 透明、置顶、可拖动的 Windows 桌宠
- 九种状态：待机、悬停、拖动、工作、无聊、权限、完成、失败、退出
- 只读解析 `%CODEX_HOME%\sessions` 和 `logs_2.sqlite`
- 系统托盘隐藏、召回和彻底退出
- 当前用户登录自启动和桌面快捷方式
- 单文件 Windows EXE，无需单独安装 Python
- 不显示 Codex 用量浮层

## 下载和安装

1. 解压 `DaShuPet-v1.0.2-win64.zip`。
2. 直接双击 `DaShuPet.exe` 可以便携运行。
3. 运行 `install.ps1` 会安装到 `D:\DaShuPet`，同时创建登录自启动和桌面快捷方式。
4. 运行 `uninstall.ps1` 可卸载；加 `-RemoveSettings` 会同时删除设置和日志。

发布包未做商业代码签名，Windows SmartScreen 可能显示未知发布者。请只运行自己
构建或从可信发布位置获得的安装包。

## 运行条件

- Windows 10 或 Windows 11 x64
- 已使用过 Codex Desktop 或 Codex CLI
- 默认读取 `%USERPROFILE%\.codex`；设置了 `CODEX_HOME` 时自动使用该目录

监听不可用时，宠物仍支持待机、悬停、拖动、无聊、退出和托盘功能。程序不会
上传、保存或展示提示词、回复和工作区文件内容。设置和轮转日志位于
`D:\DaShuPet\data`。

## 素材规则

`assets/source` 中的九张横向序列是最终角色素材。`tools/build_assets.py` 只执行：

- 按固定帧数切分原始序列
- 对白底图执行连通背景移除
- 统一透明画布、缩放和基线
- 在安全的相邻关键帧之间加入一个双向光流运动补偿插帧

轮廓、位移或面积检查不通过时会保留原关键帧而不插帧。脚本不会生成、重绘或
替换角色特征；每张原图的 SHA-256 和每个插帧的来源记录在
`assets/generated/qa/provenance.json`。

## 从源码运行

```powershell
py -3.13 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install -r requirements-dev.txt
python tools\build_assets.py
python run_pet.pyw
```

测试：

```powershell
$env:QT_QPA_PLATFORM = "offscreen"
python -m unittest discover -s tests -v
```

## 本地构建

```powershell
.\build.ps1 -Python python -Version 1.0.2
```

构建结果：

- `dist\DaShuPet.exe`
- `release\DaShuPet-v1.0.2-win64.zip`

## 状态触发

| 动画 | 触发条件 |
| --- | --- |
| 待机呼吸 | 默认状态 |
| 悬停提示 | 鼠标悬停 |
| 跟随移动 | 拖动桌宠 |
| 工作运行 | Codex 任务开始 |
| 无聊发呆 | 600 秒未使用 Codex |
| 权限请求 | 等待权限或用户批准 |
| 任务完成 | 任务完成后播放一次 |
| 任务失败 | 活动任务明确失败或额度错误 |
| 关闭退出 | 隐藏或彻底退出前播放一次 |

## License

程序源代码按 [GPL-3.0](LICENSE) 发布。角色素材不包含在 GPL 授权中，详见
[ASSET_LICENSE.md](ASSET_LICENSE.md)。这是非官方项目，与 OpenAI 无隶属或背书关系。

