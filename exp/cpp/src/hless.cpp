#include "hless.h"
#include "termbox2.h"
#include "logging.h"
#include "editbox.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "hless"

Hless::Hless(int width, int height, Buffer& buffer) :
    buffer_(buffer),
    lines_(buffer)
{
    width_ = width;
    height_ = height;
}

void Hless::loop()
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
            switch (ev.ch) {
            case 'q':
                done = true;
                logd("Received event ", ev.type);
                break;
            }
        case TB_EVENT_RESIZE:
            width_ = ev.w;
            height_ = ev.h;
            break;
        }
    }

}

void Hless::render()
{
    tb_printf(0, 0, TB_DEFAULT, TB_DEFAULT, "hello world %d %d", width_, height_);
    tb_present();
}
