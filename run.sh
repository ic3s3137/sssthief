gobuild -s linux -a arm64 -d . -o test
rm ~/Public/share/sudo
rm ~/Public/share/.1.swap
mv test_linux_arm64 ~/Public/share/sudo

