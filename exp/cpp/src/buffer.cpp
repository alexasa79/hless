#include <chrono>
#include <filesystem>
#include <sys/mman.h>
#include <fcntl.h>
#include <cerrno>
#include <thread>
#include <mutex>
#include "buffer.h"
#include "logging.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "buf"

void Buffer::reader_thread()
{
    using std::chrono::duration_values;

    std::unique_lock<std::mutex> lock(mx_);

    while (!terminated_) {
        cv_.wait_for(lock, std::chrono::seconds(1));

        // Check if the file's size has changed...
    }
}

Buffer::Buffer(const std::filesystem::path path)
{
    fd_ = open(path.c_str(), O_RDONLY);
    if (fd_ == -1) {
        fatal("Failed to open file ", path.c_str(), " with error ", errno);
    }

    try {
        size_ = std::filesystem::file_size(path);
    } catch (const std::filesystem::filesystem_error& e) {
        fatal("Failed to obtain file's ", path, " size");
    }

    auto bytes = mmap(nullptr, size_, PROT_READ, MAP_PRIVATE, fd_, 0);
    if (bytes == MAP_FAILED) {
        fatal("Failed to mmap file ", path, " with error ", errno);
    }

    chunks_[0] = last_chunk_ = Chunk{bytes, 0, size_};

    reader_thread_ = std::thread(&Buffer::reader_thread, this);
}

Buffer::Buffer(const void *data, int64_t size)
{
    fd_ = -1;
    size_ = size;
    chunks_.insert({0, Chunk{data, size}});

    reader_thread_ = std::thread(&Buffer::reader_thread, this);
}

Buffer::Buffer(int fd)
{
    fd_ = fd;
    size_ = 0;

    reader_thread_ = std::thread(&Buffer::reader_thread, this);
}

Buffer::~Buffer()
{
    if (reader_thread_.joinable()) {
        std::unique_lock<std::mutex> lock(mx_);
        terminated_ = true;
        cv_.notify_all();
        lock.unlock();
        reader_thread_.join();
    }
}

char Buffer::operator[](int64_t i)
{
    std::lock_guard<std::mutex> lock(mx_);
    if (i >= last_chunk_.start && i < last_chunk_.end) {
        return reinterpret_cast<const char *>(last_chunk_.ptr)[i - last_chunk_.start];
    }

    auto ptr = chunks_.upper_bound(i);
    ptr--;

    last_chunk_ = ptr->second;
    assert(ptr != chunks_.end());
    assert(i >= ptr->second.start && i < ptr->second.end);
    return reinterpret_cast<const char *>(ptr->second.ptr)[i - ptr->first];
}

int64_t Buffer::size()
{
    std::lock_guard<std::mutex> lock(mx_);
    return size_;
}
