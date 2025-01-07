#include <iostream>
#include <filesystem>
#include "buffer.h"

int main(int argc, char *argv[])
{
    if (argc == 1) {
        return 0;
    }

    Buffer buffer(std::filesystem::path{argv[1]});

    int64_t size = buffer.size();
    char sum = 0;
    for (int64_t i = 0; i < size; i++) {
        sum += buffer[i];
    }

    std::cout << "Scanned " << size << " bytes. The sum of all bytes is " <<
        int(sum) << std::endl;
}