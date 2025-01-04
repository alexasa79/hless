#include <iostream>
#include "termbox2.h"
#include "hless.h"
#include "logging.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "main"

void loop()
{
    struct tb_event ev;
    Hless hless;

    tb_init();

    bool done = false;
    while (!done) {
        int err = tb_poll_event(&ev);
        if (err != TB_OK) {
            if (err == TB_ERR_POLL && tb_last_errno() == EINTR) {
                continue;
            }
            fatal("tb_poll_event failed with error ", err);
            break;
        }

        logt("Received an event of type", int(ev.type), "with parameters",
            int(ev.mod), int(ev.key), int(ev.ch), int(ev.w), int(ev.h));

        switch (ev.type) {
        case TB_EVENT_KEY:
            if (ev.ch == 'q') {
                done = true;
                logd("Received event ", ev.type);
                break;
            }
        }

        tb_clear();
        tb_present();

        hless.render(tb_width(), tb_height());
    }

    tb_shutdown();
}

int main()
{
    setup_logging(std::string("hless.log"));
    logi("Hless started");
    loop();
    shutdown_logging();
}
