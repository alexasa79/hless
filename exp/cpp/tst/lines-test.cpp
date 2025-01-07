#include "buffer.h"
#include "lines.h"

static void test1()
{
    std::string s("0123456789\n0123456");
    Buffer b(s.c_str(), s.length());
    Lines ll(b);

    ll.test_set_batch_size(1);

}

int main()
{
    test1();
    return 0;
}