#include "hless.h"
#include "termbox2.h"

Hless::Hless()
{

}

void Hless::render(int width, int height)
{
    tb_printf(0, 0, TB_DEFAULT, TB_DEFAULT, "hello world %d %d", width, height);
}