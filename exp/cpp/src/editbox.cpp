#include "editbox.h"
#include "termbox2.h"
#include "logging.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "edit"

Editbox::Editbox(int x, int y, int length)
{
    x_ = x;
    y_ = y;
    length_ = length;

    window_ = 0;
    cursor_ = 0;
}

void Editbox::loop()
{
    struct tb_event ev;

    bool done = false;
    while (!done) {
        render();

        int err = tb_poll_event(&ev);
        if (err != TB_OK) {
            if (err == TB_ERR_POLL && tb_last_errno() == EINTR) {
                continue;
            }
            fatal("tb_poll_event failed with error ", err);
            break;
        }

        logt("Received an event of type ", int(ev.type), " with mod=",
            int(ev.mod), ", key=", int(ev.key), ", ch=", int(ev.ch), ", w=", int(ev.w),
            ", h=", int(ev.h));

        switch (ev.type) {
        case TB_EVENT_KEY:
            if (ev.ch == 'q') {
                done = true;
                break;
            }

            if (ev.ch == 0) {
                if (ev.mod == TB_MOD_CTRL) {
                    if (ev.key == TB_KEY_BACKSPACE || ev.key == TB_KEY_BACKSPACE2) {
                        auto nth = window_ + cursor_ - 1;
                        bytes_ = bytes_.substr(0, nth) +
                            bytes_.substr(nth + 1, bytes_.length() - nth + 1);
                        if (cursor_ > 0) {
                            cursor_--;
                        }
                    } else {
                        done = true;
                        break;
                    }
                } else if (ev.mod == 0 && ev.key == TB_KEY_ARROW_LEFT) {
                    if (cursor_ > 0) {
                        cursor_--;
                    } else {
                        if (window_ > 0) {
                            window_--;
                        }
                    }
                } else if (ev.mod == 0 && ev.key == TB_KEY_ARROW_RIGHT) {
                    if (cursor_ < length_ && cursor_ < bytes_.length()) {
                        cursor_++;
                    } else if (cursor_ == length_ && window_ + length_ < bytes_.length()) {
                        window_++;
                    }
                } else if (ev.mod == 0 && ev.key == TB_KEY_DELETE) {
                    auto nth = window_ + cursor_;
                    logt("Deleting nth ", nth, "character. Text before:", bytes_);
                    if (nth < bytes_.length()) {
                        bytes_ = bytes_.substr(0, nth) +
                            bytes_.substr(nth + 1, bytes_.length() - nth + 1);
                        if (window_ + cursor_ > bytes_.length()) {
                            cursor_--;
                        }
                    }
                    logt("Text after:", bytes_);
                } else if (ev.mod == 0 && ev.key == TB_KEY_ESC) {
                    done = true;
                    break;
                }
            } else {
                auto pos = window_ + cursor_;
                bytes_ = bytes_.substr(0, pos) + char(ev.ch) + bytes_.substr(pos, bytes_.length());

                if (bytes_.length() > length_) {
                    window_++;
                } else {
                    cursor_++;
                }
            }

        }

        tb_printf(0, 0, TB_DEFAULT, TB_DEFAULT, "Last character %d %d %d %d",
            ev.type, ev.mod, ev.key, ev.ch);
        tb_present();
    }

    clear();
}

void Editbox::clear()
{
    tb_set_cursor(x_ + cursor_, y_);
    for (auto i = 0; i < length_; i++) {
        tb_set_cell(x_ + i, y_, ' ', TB_DEFAULT, TB_DEFAULT);
        tb_set_cell(x_ + i, y_ + 1, ' ', TB_DEFAULT, TB_DEFAULT);
    }
    tb_present();
}

void Editbox::render()
{
    clear();

    std::string text = bytes_.substr(window_, length_);
    logi("Rendering editbox. Bytes=", bytes_, ", window=", window_, ", cursor=", cursor_,
        ", rendering text=", text);

    tb_print(x_, y_, TB_DEFAULT, TB_DEFAULT, text.c_str());
    tb_printf(x_, y_ + 1, TB_DEFAULT, TB_DEFAULT, "window %d, cursor %d", window_, cursor_);
    tb_present();
}
