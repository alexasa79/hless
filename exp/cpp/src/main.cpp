#include <iostream>
#include <filesystem>
#include "termbox2.h"
#include "hless.h"
#include "logging.h"
#include "editbox.h"
#include "buffer.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "main"

struct config {
    const char *filename;
};

void loop(Buffer& b)
{
    tb_init();
    tb_set_input_mode(TB_INPUT_ALT | TB_INPUT_ESC);
    tb_set_output_mode(TB_OUTPUT_TRUECOLOR);
    tb_hide_cursor();
    tb_present();

    Hless hless(tb_width(), tb_height(), b);

    hless.loop();

    tb_shutdown();
}

int main(int argc, char *argv[])
{
    assert(argc == 2);

    std::filesystem::path path(argv[1]);
    Buffer b(path);

    setup_logging(std::string("hless.log"));
    logi("Hless started");
    loop(b);
    shutdown_logging();
}
