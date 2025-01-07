#pragma once

#include <map>
#include "buffer.h"

struct Section {
    std::string name;

    int64_t read_start;
    int64_t read_end;
    int64_t n_lines;

    int64_t next_forward_index;
    int64_t next_backward_index;
};

struct Line {
    int64_t offset;
    int64_t index;
    Section *section;
};

class Lines {
public:
    Lines(Buffer &buffer);

    int64_t length(Line *line);
    Line* first();
    Line* next(Line* line);
    void jump_to(int64_t offset, std::string name);

    void test_set_batch_size(int64_t bs);

private:
    void read_forward();

    Buffer& buffer_;
    std::map<int64_t, Section*> sections_;
    std::map<int64_t, Line*> lines_;

    Section *current_section_;

    int64_t batch_size_;
};