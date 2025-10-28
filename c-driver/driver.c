#include "driver.h"
#include <fcntl.h>
#include <linux/spi/spidev.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <unistd.h>

const uint8_t spi_mode = SPI_MODE_0;
const uint8_t spi_bits = 8;
const uint32_t spi_speed = 6400000;

#define DATA_HIGH 0b00011111
#define DATA_LOW  0b00000011
#define RESET_BYTES 40

int ws2812_init(const char* device) {
    int fd;
    if ((fd = open(device, O_RDWR)) < 0) {
        perror("[C] SPI 设备打开失败");
        return -1;
    }
    ioctl(fd, SPI_IOC_WR_MODE, &spi_mode);
    ioctl(fd, SPI_IOC_WR_BITS_PER_WORD, &spi_bits);
    ioctl(fd, SPI_IOC_WR_MAX_SPEED_HZ, &spi_speed);
    printf("[C] SPI 设备 '%s' 初始化成功 (fd: %d)\n", device, fd);
    return fd;
}

void ws2812_close(int fd) {
    if (fd >= 0) {
        close(fd);
        printf("[C] SPI 设备关闭。\n");
    }
}

static inline void encode_byte(uint8_t color_byte, uint8_t* buffer_ptr) {
    for (int i = 7; i >= 0; i--) {
        *buffer_ptr++ = (color_byte & (1 << i)) ? DATA_HIGH : DATA_LOW;
    }
}

int ws2812_send_colors(int fd, const uint8_t* rgb_data, int num_leds) {
    if (fd < 0 || num_leds <= 0) return -1;

    const int buffer_size = num_leds * BITS_PER_LED + RESET_BYTES;
    uint8_t *tx_buffer = (uint8_t*) calloc(1, buffer_size);
    if (!tx_buffer) {
        perror("[C] 内存分配失败");
        return -1;
    }

    uint8_t* current_ptr = tx_buffer;

    for (int i = 0; i < num_leds; i++) {
        const uint8_t r = rgb_data[i * 3 + 0];
        const uint8_t g = rgb_data[i * 3 + 1];
        const uint8_t b = rgb_data[i * 3 + 2];

        encode_byte(g, current_ptr); current_ptr += 8;
        encode_byte(r, current_ptr); current_ptr += 8;
        encode_byte(b, current_ptr); current_ptr += 8;
    }

    struct spi_ioc_transfer tr = {
        .tx_buf = (unsigned long)tx_buffer,
        .len = buffer_size,
        .speed_hz = spi_speed,
        .bits_per_word = spi_bits,
    };

    int ret = ioctl(fd, SPI_IOC_MESSAGE(1), &tr);

    free(tx_buffer);
    if (ret < 0) {
        perror("[C] SPI 信息发送失败");
        return -1;
    }
    return ret;
}
