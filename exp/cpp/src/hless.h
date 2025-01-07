#pragma once

#include "buffer.h"
#include "lines.h"

class Hless {
public:
    Hless(int width, int height, Buffer& b);

    void loop();
    void render();

private:
    Buffer& buffer_;
    Lines lines_;

    int width_;
    int height_;
};
