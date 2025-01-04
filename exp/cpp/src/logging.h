#pragma once

#include <string>
#include <fstream>
#include <cassert>
#include <iostream>
#include <sstream>
#include <chrono>
#include <iomanip>

enum {
    LOGGING_FATAL,
    LOGGING_ERROR,
    LOGGING_WARN,
    LOGGING_INFO,
    LOGGING_DEBUG,
    LOGGING_TRACE,
};

#define fatal(...) ({ \
    if (___logf.is_open()) { \
        ____log(LOGGING_FATAL, LOGGING_NAME, __FILE__, __LINE__, ##__VA_ARGS__); \
    } \
    abort(); \
})
#define ___log(level, ...) \
    if (LOGGING_MODULE >= level) { \
        if (___logf.is_open()) { \
            ____log(level, LOGGING_NAME, __FILE__, __LINE__, ##__VA_ARGS__); \
        } \
    }

#define loge(...) ___log(LOGGING_ERROR, ##__VA_ARGS__)
#define logw(...) ___log(LOGGING_WARN, ##__VA_ARGS__)
#define logi(...) ___log(LOGGING_INFO, ##__VA_ARGS__)
#define logd(...) ___log(LOGGING_DEBUG, ##__VA_ARGS__)
#define logt(...) ___log(LOGGING_TRACE, ##__VA_ARGS__)

extern std::ofstream ___logf;

void setup_logging(std::string filename);
void shutdown_logging();

template<typename... Args>
void ____log(int level, const char *module, const char *filename, int line, Args&&... args)
{
    const std::string levels[] = {
        "FATAL",
        "ERROR",
        "WARN ",
        "INFO ",
        "DEBUG",
        "TRACE",
    };

    std::ostringstream oss;
    auto now = std::chrono::system_clock::now();
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()) % 1000;
    auto now_c = std::chrono::system_clock::to_time_t(now);

    oss << std::put_time(std::localtime(&now_c), "%Y-%m-%dT%H:%M:%S") << '.'
        << std::setfill('0') << std::setw(3) << ms.count() << " ";
    oss << levels[level] << " ";
    oss << filename << ":" << line << " ";

    auto add_space = [](auto&& value, std::ostringstream& oss) {
        oss << value << ' ';
    };
    (add_space(std::forward<Args>(args), oss), ...);

    ___logf << oss.str() << std::endl;
    ___logf.flush();
}
