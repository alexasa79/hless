/*
MIT License

Copyright (c) 2010-2020 nsf <no.smile.face@gmail.com>
              2015-2024 Adam Saponara <as@php.net>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

#ifndef TERMBOX_H_INCL
#define TERMBOX_H_INCL

#ifndef _XOPEN_SOURCE
#define _XOPEN_SOURCE
#endif

#ifndef _DEFAULT_SOURCE
#define _DEFAULT_SOURCE
#endif

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <signal.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/select.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <termios.h>
#include <unistd.h>
#include <wchar.h>
#include <wctype.h>

#ifdef PATH_MAX
#define TB_PATH_MAX PATH_MAX
#else
#define TB_PATH_MAX 4096
#endif

#ifdef __cplusplus
extern "C" {
#endif

// __ffi_start

#define TB_VERSION_STR "2.5.0"

/* The following compile-time options are supported:
 *
 *     TB_OPT_ATTR_W: Integer width of fg and bg attributes. Valid values
 *                    (assuming system support) are 16, 32, and 64. (See
 *                    uintattr_t). 32 or 64 enables output mode
 *                    TB_OUTPUT_TRUECOLOR. 64 enables additional style
 *                    attributes. (See tb_set_output_mode.) Larger values
 *                    consume more memory in exchange for more features.
 *                    Defaults to 16.
 *
 *        TB_OPT_EGC: If set, enable extended grapheme cluster support
 *                    (tb_extend_cell, tb_set_cell_ex). Consumes more memory.
 *                    Defaults off.
 *
 * TB_OPT_PRINTF_BUF: Write buffer size for printf operations. Represents the
 *                    largest string that can be sent in one call to tb_print*
 *                    and tb_send* functions. Defaults to 4096.
 *
 *   TB_OPT_READ_BUF: Read buffer size for tty reads. Defaults to 64.
 *
 *  TB_OPT_TRUECOLOR: Deprecated. Sets TB_OPT_ATTR_W to 32 if not already set.
 */

#if defined(TB_LIB_OPTS) || 0 // __tb_lib_opts
/* Ensure consistent compile-time options when using as a shared library */
#undef TB_OPT_ATTR_W
#undef TB_OPT_EGC
#undef TB_OPT_PRINTF_BUF
#undef TB_OPT_READ_BUF
#define TB_OPT_ATTR_W 64
#define TB_OPT_EGC
#endif

/* Ensure sane `TB_OPT_ATTR_W` (16, 32, or 64) */
#if defined TB_OPT_ATTR_W && TB_OPT_ATTR_W == 16
#elif defined TB_OPT_ATTR_W && TB_OPT_ATTR_W == 32
#elif defined TB_OPT_ATTR_W && TB_OPT_ATTR_W == 64
#else
#undef TB_OPT_ATTR_W
#if defined TB_OPT_TRUECOLOR // Deprecated. Back-compat for old flag.
#define TB_OPT_ATTR_W 32
#else
#define TB_OPT_ATTR_W 16
#endif
#endif

/* ASCII key constants (`tb_event.key`) */
#define TB_KEY_CTRL_TILDE       0x00
#define TB_KEY_CTRL_2           0x00 // clash with `CTRL_TILDE`
#define TB_KEY_CTRL_A           0x01
#define TB_KEY_CTRL_B           0x02
#define TB_KEY_CTRL_C           0x03
#define TB_KEY_CTRL_D           0x04
#define TB_KEY_CTRL_E           0x05
#define TB_KEY_CTRL_F           0x06
#define TB_KEY_CTRL_G           0x07
#define TB_KEY_BACKSPACE        0x08
#define TB_KEY_CTRL_H           0x08 // clash with `CTRL_BACKSPACE`
#define TB_KEY_TAB              0x09
#define TB_KEY_CTRL_I           0x09 // clash with `TAB`
#define TB_KEY_CTRL_J           0x0a
#define TB_KEY_CTRL_K           0x0b
#define TB_KEY_CTRL_L           0x0c
#define TB_KEY_ENTER            0x0d
#define TB_KEY_CTRL_M           0x0d // clash with `ENTER`
#define TB_KEY_CTRL_N           0x0e
#define TB_KEY_CTRL_O           0x0f
#define TB_KEY_CTRL_P           0x10
#define TB_KEY_CTRL_Q           0x11
#define TB_KEY_CTRL_R           0x12
#define TB_KEY_CTRL_S           0x13
#define TB_KEY_CTRL_T           0x14
#define TB_KEY_CTRL_U           0x15
#define TB_KEY_CTRL_V           0x16
#define TB_KEY_CTRL_W           0x17
#define TB_KEY_CTRL_X           0x18
#define TB_KEY_CTRL_Y           0x19
#define TB_KEY_CTRL_Z           0x1a
#define TB_KEY_ESC              0x1b
#define TB_KEY_CTRL_LSQ_BRACKET 0x1b // clash with 'ESC'
#define TB_KEY_CTRL_3           0x1b // clash with 'ESC'
#define TB_KEY_CTRL_4           0x1c
#define TB_KEY_CTRL_BACKSLASH   0x1c // clash with 'CTRL_4'
#define TB_KEY_CTRL_5           0x1d
#define TB_KEY_CTRL_RSQ_BRACKET 0x1d // clash with 'CTRL_5'
#define TB_KEY_CTRL_6           0x1e
#define TB_KEY_CTRL_7           0x1f
#define TB_KEY_CTRL_SLASH       0x1f // clash with 'CTRL_7'
#define TB_KEY_CTRL_UNDERSCORE  0x1f // clash with 'CTRL_7'
#define TB_KEY_SPACE            0x20
#define TB_KEY_BACKSPACE2       0x7f
#define TB_KEY_CTRL_8           0x7f // clash with 'BACKSPACE2'

#define tb_key_i(i)             0xffff - (i)
/* Terminal-dependent key constants (`tb_event.key`) and terminfo caps */
/* BEGIN codegen h */
/* Produced by ./codegen.sh on Tue, 03 Sep 2024 04:17:47 +0000 */
#define TB_KEY_F1               (0xffff - 0)
#define TB_KEY_F2               (0xffff - 1)
#define TB_KEY_F3               (0xffff - 2)
#define TB_KEY_F4               (0xffff - 3)
#define TB_KEY_F5               (0xffff - 4)
#define TB_KEY_F6               (0xffff - 5)
#define TB_KEY_F7               (0xffff - 6)
#define TB_KEY_F8               (0xffff - 7)
#define TB_KEY_F9               (0xffff - 8)
#define TB_KEY_F10              (0xffff - 9)
#define TB_KEY_F11              (0xffff - 10)
#define TB_KEY_F12              (0xffff - 11)
#define TB_KEY_INSERT           (0xffff - 12)
#define TB_KEY_DELETE           (0xffff - 13)
#define TB_KEY_HOME             (0xffff - 14)
#define TB_KEY_END              (0xffff - 15)
#define TB_KEY_PGUP             (0xffff - 16)
#define TB_KEY_PGDN             (0xffff - 17)
#define TB_KEY_ARROW_UP         (0xffff - 18)
#define TB_KEY_ARROW_DOWN       (0xffff - 19)
#define TB_KEY_ARROW_LEFT       (0xffff - 20)
#define TB_KEY_ARROW_RIGHT      (0xffff - 21)
#define TB_KEY_BACK_TAB         (0xffff - 22)
#define TB_KEY_MOUSE_LEFT       (0xffff - 23)
#define TB_KEY_MOUSE_RIGHT      (0xffff - 24)
#define TB_KEY_MOUSE_MIDDLE     (0xffff - 25)
#define TB_KEY_MOUSE_RELEASE    (0xffff - 26)
#define TB_KEY_MOUSE_WHEEL_UP   (0xffff - 27)
#define TB_KEY_MOUSE_WHEEL_DOWN (0xffff - 28)

#define TB_CAP_F1               0
#define TB_CAP_F2               1
#define TB_CAP_F3               2
#define TB_CAP_F4               3
#define TB_CAP_F5               4
#define TB_CAP_F6               5
#define TB_CAP_F7               6
#define TB_CAP_F8               7
#define TB_CAP_F9               8
#define TB_CAP_F10              9
#define TB_CAP_F11              10
#define TB_CAP_F12              11
#define TB_CAP_INSERT           12
#define TB_CAP_DELETE           13
#define TB_CAP_HOME             14
#define TB_CAP_END              15
#define TB_CAP_PGUP             16
#define TB_CAP_PGDN             17
#define TB_CAP_ARROW_UP         18
#define TB_CAP_ARROW_DOWN       19
#define TB_CAP_ARROW_LEFT       20
#define TB_CAP_ARROW_RIGHT      21
#define TB_CAP_BACK_TAB         22
#define TB_CAP__COUNT_KEYS      23
#define TB_CAP_ENTER_CA         23
#define TB_CAP_EXIT_CA          24
#define TB_CAP_SHOW_CURSOR      25
#define TB_CAP_HIDE_CURSOR      26
#define TB_CAP_CLEAR_SCREEN     27
#define TB_CAP_SGR0             28
#define TB_CAP_UNDERLINE        29
#define TB_CAP_BOLD             30
#define TB_CAP_BLINK            31
#define TB_CAP_ITALIC           32
#define TB_CAP_REVERSE          33
#define TB_CAP_ENTER_KEYPAD     34
#define TB_CAP_EXIT_KEYPAD      35
#define TB_CAP_DIM              36
#define TB_CAP_INVISIBLE        37
#define TB_CAP__COUNT           38
/* END codegen h */

/* Some hard-coded caps */
#define TB_HARDCAP_ENTER_MOUSE  "\x1b[?1000h\x1b[?1002h\x1b[?1015h\x1b[?1006h"
#define TB_HARDCAP_EXIT_MOUSE   "\x1b[?1006l\x1b[?1015l\x1b[?1002l\x1b[?1000l"
#define TB_HARDCAP_STRIKEOUT    "\x1b[9m"
#define TB_HARDCAP_UNDERLINE_2  "\x1b[21m"
#define TB_HARDCAP_OVERLINE     "\x1b[53m"

/* Colors (numeric) and attributes (bitwise) (`tb_cell.fg`, `tb_cell.bg`) */
#define TB_DEFAULT              0x0000
#define TB_BLACK                0x0001
#define TB_RED                  0x0002
#define TB_GREEN                0x0003
#define TB_YELLOW               0x0004
#define TB_BLUE                 0x0005
#define TB_MAGENTA              0x0006
#define TB_CYAN                 0x0007
#define TB_WHITE                0x0008

#if TB_OPT_ATTR_W == 16
#define TB_BOLD      0x0100
#define TB_UNDERLINE 0x0200
#define TB_REVERSE   0x0400
#define TB_ITALIC    0x0800
#define TB_BLINK     0x1000
#define TB_HI_BLACK  0x2000
#define TB_BRIGHT    0x4000
#define TB_DIM       0x8000
#define TB_256_BLACK TB_HI_BLACK // `TB_256_BLACK` is deprecated
#else
// `TB_OPT_ATTR_W` is 32 or 64
#define TB_BOLD                0x01000000
#define TB_UNDERLINE           0x02000000
#define TB_REVERSE             0x04000000
#define TB_ITALIC              0x08000000
#define TB_BLINK               0x10000000
#define TB_HI_BLACK            0x20000000
#define TB_BRIGHT              0x40000000
#define TB_DIM                 0x80000000
#define TB_TRUECOLOR_BOLD      TB_BOLD // `TB_TRUECOLOR_*` is deprecated
#define TB_TRUECOLOR_UNDERLINE TB_UNDERLINE
#define TB_TRUECOLOR_REVERSE   TB_REVERSE
#define TB_TRUECOLOR_ITALIC    TB_ITALIC
#define TB_TRUECOLOR_BLINK     TB_BLINK
#define TB_TRUECOLOR_BLACK     TB_HI_BLACK
#endif

#if TB_OPT_ATTR_W == 64
#define TB_STRIKEOUT   0x0000000100000000
#define TB_UNDERLINE_2 0x0000000200000000
#define TB_OVERLINE    0x0000000400000000
#define TB_INVISIBLE   0x0000000800000000
#endif

/* Event types (`tb_event.type`) */
#define TB_EVENT_KEY        1
#define TB_EVENT_RESIZE     2
#define TB_EVENT_MOUSE      3

/* Key modifiers (bitwise) (`tb_event.mod`) */
#define TB_MOD_ALT          1
#define TB_MOD_CTRL         2
#define TB_MOD_SHIFT        4
#define TB_MOD_MOTION       8

/* Input modes (bitwise) (`tb_set_input_mode`) */
#define TB_INPUT_CURRENT    0
#define TB_INPUT_ESC        1
#define TB_INPUT_ALT        2
#define TB_INPUT_MOUSE      4

/* Output modes (`tb_set_output_mode`) */
#define TB_OUTPUT_CURRENT   0
#define TB_OUTPUT_NORMAL    1
#define TB_OUTPUT_256       2
#define TB_OUTPUT_216       3
#define TB_OUTPUT_GRAYSCALE 4
#if TB_OPT_ATTR_W >= 32
#define TB_OUTPUT_TRUECOLOR 5
#endif

/* Common function return values unless otherwise noted.
 *
 * Library behavior is undefined after receiving `TB_ERR_MEM`. Callers may
 * attempt reinitializing by freeing memory, invoking `tb_shutdown`, then
 * `tb_init`.
 */
#define TB_OK                   0
#define TB_ERR                  -1
#define TB_ERR_NEED_MORE        -2
#define TB_ERR_INIT_ALREADY     -3
#define TB_ERR_INIT_OPEN        -4
#define TB_ERR_MEM              -5
#define TB_ERR_NO_EVENT         -6
#define TB_ERR_NO_TERM          -7
#define TB_ERR_NOT_INIT         -8
#define TB_ERR_OUT_OF_BOUNDS    -9
#define TB_ERR_READ             -10
#define TB_ERR_RESIZE_IOCTL     -11
#define TB_ERR_RESIZE_PIPE      -12
#define TB_ERR_RESIZE_SIGACTION -13
#define TB_ERR_POLL             -14
#define TB_ERR_TCGETATTR        -15
#define TB_ERR_TCSETATTR        -16
#define TB_ERR_UNSUPPORTED_TERM -17
#define TB_ERR_RESIZE_WRITE     -18
#define TB_ERR_RESIZE_POLL      -19
#define TB_ERR_RESIZE_READ      -20
#define TB_ERR_RESIZE_SSCANF    -21
#define TB_ERR_CAP_COLLISION    -22

#define TB_ERR_SELECT           TB_ERR_POLL
#define TB_ERR_RESIZE_SELECT    TB_ERR_RESIZE_POLL

/* Deprecated. Function types to be used with `tb_set_func`. */
#define TB_FUNC_EXTRACT_PRE     0
#define TB_FUNC_EXTRACT_POST    1

/* Define this to set the size of the buffer used in `tb_printf`
 * and `tb_sendf`
 */
#ifndef TB_OPT_PRINTF_BUF
#define TB_OPT_PRINTF_BUF 4096
#endif

/* Define this to set the size of the read buffer used when reading
 * from the tty
 */
#ifndef TB_OPT_READ_BUF
#define TB_OPT_READ_BUF 64
#endif

/* Define this for limited back compat with termbox v1 */
#ifdef TB_OPT_V1_COMPAT
#define tb_change_cell          tb_set_cell
#define tb_put_cell(x, y, c)    tb_set_cell((x), (y), (c)->ch, (c)->fg, (c)->bg)
#define tb_set_clear_attributes tb_set_clear_attrs
#define tb_select_input_mode    tb_set_input_mode
#define tb_select_output_mode   tb_set_output_mode
#endif

/* Define these to swap in a different allocator */
#ifndef tb_malloc
#define tb_malloc  malloc
#define tb_realloc realloc
#define tb_free    free
#endif

#if TB_OPT_ATTR_W == 64
typedef uint64_t uintattr_t;
#elif TB_OPT_ATTR_W == 32
typedef uint32_t uintattr_t;
#else // 16
typedef uint16_t uintattr_t;
#endif

/* A cell in a 2d grid representing the terminal screen.
 *
 * The terminal screen is represented as 2d array of cells. The structure is
 * optimized for dealing with single-width (`wcwidth==1`) Unicode codepoints,
 * however some support for grapheme clusters (e.g., combining diacritical
 * marks) and wide codepoints (e.g., Hiragana) is provided through `ech`,
 * `nech`, and `cech` via `tb_set_cell_ex`. `ech` is only valid when `nech>0`,
 * otherwise `ch` is used.
 *
 * For non-single-width codepoints, given `N=wcwidth(ch)/wcswidth(ech)`:
 *
 * when `N==0`: termbox forces a single-width cell. Callers should avoid this
 *              if aiming to render text accurately. Callers may use
 *              `tb_set_cell_ex` or `tb_print*` to render `N==0` combining
 *              characters.
 *
 *  when `N>1`: termbox zeroes out the following `N-1` cells and skips sending
 *              them to the tty. So, e.g., if the caller sets `x=0,y=0` to an
 *              `N==2` codepoint, the caller's next set should be at `x=2,y=0`.
 *              Anything set at `x=1,y=0` will be ignored. If there are not
 *              enough columns remaining on the line to render `N` width, spaces
 *              are sent instead.
 *
 * See `tb_present` for implementation.
 */
struct tb_cell {
    uint32_t ch;   // a Unicode codepoint
    uintattr_t fg; // bitwise foreground attributes
    uintattr_t bg; // bitwise background attributes
#ifdef TB_OPT_EGC
    uint32_t *ech; // a grapheme cluster of Unicode codepoints, 0-terminated
    size_t nech;   // num elements in ech, 0 means use ch instead of ech
    size_t cech;   // num elements allocated for ech
#endif
};

/* An incoming event from the tty.
 *
 * Given the event type, the following fields are relevant:
 *
 *    when `TB_EVENT_KEY`: `key` xor `ch` (one will be zero) and `mod`. Note
 *                         there is overlap between `TB_MOD_CTRL` and
 *                         `TB_KEY_CTRL_*`. `TB_MOD_CTRL` and `TB_MOD_SHIFT` are
 *                         only set as modifiers to `TB_KEY_ARROW_*`.
 *
 * when `TB_EVENT_RESIZE`: `w` and `h`
 *
 *  when `TB_EVENT_MOUSE`: `key` (`TB_KEY_MOUSE_*`), `x`, and `y`
 */
struct tb_event {
    uint8_t type; // one of `TB_EVENT_*` constants
    uint8_t mod;  // bitwise `TB_MOD_*` constants
    uint16_t key; // one of `TB_KEY_*` constants
    uint32_t ch;  // a Unicode codepoint
    int32_t w;    // resize width
    int32_t h;    // resize height
    int32_t x;    // mouse x
    int32_t y;    // mouse y
};

/* Initialize the termbox library. This function should be called before any
 * other functions. `tb_init` is equivalent to `tb_init_file("/dev/tty")`. After
 * successful initialization, the library must be finalized using `tb_shutdown`.
 */
int tb_init(void);
int tb_init_file(const char *path);
int tb_init_fd(int ttyfd);
int tb_init_rwfd(int rfd, int wfd);
int tb_shutdown(void);

/* Return the size of the internal back buffer (which is the same as terminal's
 * window size in rows and columns). The internal buffer can be resized after
 * `tb_clear` or `tb_present` calls. Both dimensions have an unspecified
 * negative value when called before `tb_init` or after `tb_shutdown`.
 */
int tb_width(void);
int tb_height(void);

/* Clear the internal back buffer using `TB_DEFAULT` or the attributes set by
 * `tb_set_clear_attrs`.
 */
int tb_clear(void);
int tb_set_clear_attrs(uintattr_t fg, uintattr_t bg);

/* Synchronize the internal back buffer with the terminal by writing to tty. */
int tb_present(void);

/* Clear the internal front buffer effectively forcing a complete re-render of
 * the back buffer to the tty. It is not necessary to call this under normal
 * circumstances. */
int tb_invalidate(void);

/* Set the position of the cursor. Upper-left cell is (0, 0). */
int tb_set_cursor(int cx, int cy);
int tb_hide_cursor(void);

/* Set cell contents in the internal back buffer at the specified position.
 *
 * Use `tb_set_cell_ex` for rendering grapheme clusters (e.g., combining
 * diacritical marks).
 *
 * Calling `tb_set_cell(x, y, ch, fg, bg)` is equivalent to
 * `tb_set_cell_ex(x, y, &ch, 1, fg, bg)`.
 *
 * `tb_extend_cell` is a shortcut for appending 1 codepoint to `tb_cell.ech`.
 *
 * Non-printable (`iswprint(3)`) codepoints are replaced with `U+FFFD` at render
 * time.
 */
int tb_set_cell(int x, int y, uint32_t ch, uintattr_t fg, uintattr_t bg);
int tb_set_cell_ex(int x, int y, uint32_t *ch, size_t nch, uintattr_t fg,
    uintattr_t bg);
int tb_extend_cell(int x, int y, uint32_t ch);

/* Set the input mode. Termbox has two input modes:
 *
 * 1. `TB_INPUT_ESC`
 *    When escape (`\x1b`) is in the buffer and there's no match for an escape
 *    sequence, a key event for `TB_KEY_ESC` is returned.
 *
 * 2. `TB_INPUT_ALT`
 *    When escape (`\x1b`) is in the buffer and there's no match for an escape
 *    sequence, the next keyboard event is returned with a `TB_MOD_ALT`
 *    modifier.
 *
 * You can also apply `TB_INPUT_MOUSE` via bitwise OR operation to either of the
 * modes (e.g., `TB_INPUT_ESC | TB_INPUT_MOUSE`) to receive `TB_EVENT_MOUSE`
 * events. If none of the main two modes were set, but the mouse mode was,
 * `TB_INPUT_ESC` is used. If for some reason you've decided to use
 * `TB_INPUT_ESC | TB_INPUT_ALT`, it will behave as if only `TB_INPUT_ESC` was
 * selected.
 *
 * If mode is `TB_INPUT_CURRENT`, return the current input mode.
 *
 * The default input mode is `TB_INPUT_ESC`.
 */
int tb_set_input_mode(int mode);

/* Set the output mode. Termbox has multiple output modes:
 *
 * 1. `TB_OUTPUT_NORMAL`     => [0..8]
 *
 *    This mode provides 8 different colors:
 *      `TB_BLACK`, `TB_RED`, `TB_GREEN`, `TB_YELLOW`,
 *      `TB_BLUE`, `TB_MAGENTA`, `TB_CYAN`, `TB_WHITE`
 *
 *    Plus `TB_DEFAULT` which skips sending a color code (i.e., uses the
 *    terminal's default color).
 *
 *    Colors (including `TB_DEFAULT`) may be bitwise OR'd with attributes:
 *      `TB_BOLD`, `TB_UNDERLINE`, `TB_REVERSE`, `TB_ITALIC`, `TB_BLINK`,
 *      `TB_BRIGHT`, `TB_DIM`
 *
 *    The following style attributes are also available if compiled with
 *    `TB_OPT_ATTR_W` set to 64:
 *      `TB_STRIKEOUT`, `TB_UNDERLINE_2`, `TB_OVERLINE`, `TB_INVISIBLE`
 *
 *    As in all modes, the value 0 is interpreted as `TB_DEFAULT` for
 *    convenience.
 *
 *    Some notes: `TB_REVERSE` and `TB_BRIGHT` can be applied as either `fg` or
 *    `bg` attributes for the same effect. The rest of the attributes apply to
 *    `fg` only and are ignored as `bg` attributes.
 *
 *    Example usage: `tb_set_cell(x, y, '@', TB_BLACK | TB_BOLD, TB_RED)`
 *
 * 2. `TB_OUTPUT_256`        => [0..255] + `TB_HI_BLACK`
 *
 *    In this mode you get 256 distinct colors (plus default):
 *                0x00   (1): `TB_DEFAULT`
 *       `TB_HI_BLACK`   (1): `TB_BLACK` in `TB_OUTPUT_NORMAL`
 *          0x01..0x07   (7): the next 7 colors as in `TB_OUTPUT_NORMAL`
 *          0x08..0x0f   (8): bright versions of the above
 *          0x10..0xe7 (216): 216 different colors
 *          0xe8..0xff  (24): 24 different shades of gray
 *
 *    All `TB_*` style attributes except `TB_BRIGHT` may be bitwise OR'd as in
 *    `TB_OUTPUT_NORMAL`.
 *
 *    Note `TB_HI_BLACK` must be used for black, as 0x00 represents default.
 *
 * 3. `TB_OUTPUT_216`        => [0..216]
 *
 *    This mode supports the 216-color range of `TB_OUTPUT_256` only, but you
 *    don't need to provide an offset:
 *                0x00   (1): `TB_DEFAULT`
 *          0x01..0xd8 (216): 216 different colors
 *
 * 4. `TB_OUTPUT_GRAYSCALE`  => [0..24]
 *
 *    This mode supports the 24-color range of `TB_OUTPUT_256` only, but you
 *    don't need to provide an offset:
 *                0x00   (1): `TB_DEFAULT`
 *          0x01..0x18  (24): 24 different shades of gray
 *
 * 5. `TB_OUTPUT_TRUECOLOR`  => [0x000000..0xffffff] + `TB_HI_BLACK`
 *
 *    This mode provides 24-bit color on supported terminals. The format is
 *    0xRRGGBB.
 *
 *    All `TB_*` style attributes except `TB_BRIGHT` may be bitwise OR'd as in
 *    `TB_OUTPUT_NORMAL`.
 *
 *    Note `TB_HI_BLACK` must be used for black, as 0x000000 represents default.
 *
 * To use the terminal default color (i.e., to not send an escape code), pass
 * `TB_DEFAULT`. For convenience, the value 0 is interpreted as `TB_DEFAULT` in
 * all modes.
 *
 * Note, cell attributes persist after switching output modes. Any translation
 * between, for example, `TB_OUTPUT_NORMAL`'s `TB_RED` and
 * `TB_OUTPUT_TRUECOLOR`'s 0xff0000 must be performed by the caller. Also note
 * that cells previously rendered in one mode may persist unchanged until the
 * front buffer is cleared (such as after a resize event) at which point it will
 * be re-interpreted and flushed according to the current mode. Callers may
 * invoke `tb_invalidate` if it is desirable to immediately re-interpret and
 * flush the entire screen according to the current mode.
 *
 * Note, not all terminals support all output modes, especially beyond
 * `TB_OUTPUT_NORMAL`. There is also no very reliable way to determine color
 * support dynamically. If portability is desired, callers are recommended to
 * use `TB_OUTPUT_NORMAL` or make output mode end-user configurable. The same
 * advice applies to style attributes.
 *
 * If mode is `TB_OUTPUT_CURRENT`, return the current output mode.
 *
 * The default output mode is `TB_OUTPUT_NORMAL`.
 */
int tb_set_output_mode(int mode);

/* Wait for an event up to `timeout_ms` milliseconds and populate `event` with
 * it. If no event is available within the timeout period, `TB_ERR_NO_EVENT`
 * is returned. On a resize event, the underlying `select(2)` call may be
 * interrupted, yielding a return code of `TB_ERR_POLL`. In this case, you may
 * check `errno` via `tb_last_errno`. If it's `EINTR`, you may elect to ignore
 * that and call `tb_peek_event` again.
 */
int tb_peek_event(struct tb_event *event, int timeout_ms);

/* Same as `tb_peek_event` except no timeout. */
int tb_poll_event(struct tb_event *event);

/* Internal termbox fds that can be used with `poll(2)`, `select(2)`, etc.
 * externally. Callers must invoke `tb_poll_event` or `tb_peek_event` if
 * fds become readable. */
int tb_get_fds(int *ttyfd, int *resizefd);

/* Print and printf functions. Specify param `out_w` to determine width of
 * printed string. Strings are interpreted as UTF-8.
 *
 * Non-printable characters (`iswprint(3)`) and truncated UTF-8 byte sequences
 * are replaced with U+FFFD.
 *
 * Newlines (`\n`) are supported with the caveat that `out_w` will return the
 * width of the string as if it were on a single line.
 *
 * If the starting coordinate is out of bounds, `TB_ERR_OUT_OF_BOUNDS` is
 * returned. If the starting coordinate is in bounds, but goes out of bounds,
 * then the out-of-bounds portions of the string are ignored.
 *
 * For finer control, use `tb_set_cell`.
 */
int tb_print(int x, int y, uintattr_t fg, uintattr_t bg, const char *str);
int tb_printf(int x, int y, uintattr_t fg, uintattr_t bg, const char *fmt, ...);
int tb_print_ex(int x, int y, uintattr_t fg, uintattr_t bg, size_t *out_w,
    const char *str);
int tb_printf_ex(int x, int y, uintattr_t fg, uintattr_t bg, size_t *out_w,
    const char *fmt, ...);

/* Send raw bytes to terminal. */
int tb_send(const char *buf, size_t nbuf);
int tb_sendf(const char *fmt, ...);

/* Deprecated. Set custom callbacks. `fn_type` is one of `TB_FUNC_*` constants,
 * `fn` is a compatible function pointer, or NULL to clear.
 *
 * `TB_FUNC_EXTRACT_PRE`:
 *   If specified, invoke this function BEFORE termbox tries to extract any
 *   escape sequences from the input buffer.
 *
 * `TB_FUNC_EXTRACT_POST`:
 *   If specified, invoke this function AFTER termbox tries (and fails) to
 *   extract any escape sequences from the input buffer.
 */
int tb_set_func(int fn_type, int (*fn)(struct tb_event *, size_t *));

/* Return byte length of codepoint given first byte of UTF-8 sequence (1-6). */
int tb_utf8_char_length(char c);

/* Convert UTF-8 null-terminated byte sequence to UTF-32 codepoint.
 *
 * If `c` is an empty C string, return 0. `out` is left unchanged.
 *
 * If a null byte is encountered in the middle of the codepoint, return a
 * negative number indicating how many bytes were processed. `out` is left
 * unchanged.
 *
 * Otherwise, return byte length of codepoint (1-6).
 */
int tb_utf8_char_to_unicode(uint32_t *out, const char *c);

/* Convert UTF-32 codepoint to UTF-8 null-terminated byte sequence.
 *
 * `out` must be char[7] or greater. Return byte length of codepoint (1-6).
 */
int tb_utf8_unicode_to_char(char *out, uint32_t c);

/* Library utility functions */
int tb_last_errno(void);
const char *tb_strerror(int err);
struct tb_cell *tb_cell_buffer(void); // Deprecated
int tb_has_truecolor(void);
int tb_has_egc(void);
int tb_attr_width(void);
const char *tb_version(void);

/* Deprecation notice!
 *
 * The following will be removed in version 3.x (ABI version 3):
 *
 *   TB_256_BLACK           (use TB_HI_BLACK)
 *   TB_OPT_TRUECOLOR       (use TB_OPT_ATTR_W)
 *   TB_TRUECOLOR_BOLD      (use TB_BOLD)
 *   TB_TRUECOLOR_UNDERLINE (use TB_UNDERLINE)
 *   TB_TRUECOLOR_REVERSE   (use TB_REVERSE)
 *   TB_TRUECOLOR_ITALIC    (use TB_ITALICe)
 *   TB_TRUECOLOR_BLINK     (use TB_BLINK)
 *   TB_TRUECOLOR_BLACK     (use TB_HI_BLACK)
 *   tb_cell_buffer
 *   tb_set_func
 *   TB_FUNC_EXTRACT_PRE
 *   TB_FUNC_EXTRACT_POST
 */

#ifdef __cplusplus
}
#endif

#endif // TERMBOX_H_INCL
