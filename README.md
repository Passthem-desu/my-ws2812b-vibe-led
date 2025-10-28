# How to build

```sh
sudo apt install gcc-aarch64-linux-gnu golang-go

git clone https://github.com/Passthem-desu/my-ws2812b-vibe-led.git
cd my-ws2812b-vibe-led
go mod tidy

make
```

`Makefile` 里面的 `deploy` 仅用于方便我部署到我的个人设备上，目标设备是 aarch64 的香橙派。

请查阅你使用的 Linux 开发版的引脚定义和 SPIDEV 设备定义，以正确连线。

