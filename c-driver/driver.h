#ifndef DRIVER_H
#define DRIVER_H

#include <stdint.h>

#define LED_COUNT 60
#define BITS_PER_LED 24

int ws2812_init(const char* device);
void ws2812_close(int fd);
int ws2812_send_colors(int fd, const uint8_t* rgb_data, int num_leds);

#endif // DRIVER_H
