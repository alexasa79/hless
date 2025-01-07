#pragma once

#include <string>

class Editbox {
public:
    Editbox(int x, int y, int length);

    void loop();
    void render();
    void clear();

private:
    // Position of the editbox on the screen
    int x_;
    int y_;
    int length_;

    // Position from which to render the editbox in the string. In other words
    // the first character in the bytes string to render.
    int window_;

    int cursor_; // Position of the cursor relative to the window.
    std::string bytes_;
};