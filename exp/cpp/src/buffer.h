#pragma once

#include <condition_variable>
#include <cstdint>
#include <filesystem>
#include <map>
#include <thread>
#include <mutex>

struct Chunk {
    const void *ptr;
    int64_t start;
    int64_t end;
};

class Buffer {
public:
    Buffer(const std::filesystem::path path);
    Buffer(const void *data, int64_t length);
    Buffer(int fd);

    ~Buffer();

    char operator[](int64_t i);

    int64_t size();

private:
    void reader_thread();

    std::map<int64_t, Chunk> chunks_;
    int64_t size_;
    int fd_;

    std::thread reader_thread_;
    std::mutex mx_;
    std::condition_variable cv_;
    bool terminated_;

    Chunk last_chunk_;
};
