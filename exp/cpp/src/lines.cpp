#include "lines.h"
#include "logging.h"

#define LOGGING_MODULE LOGGING_TRACE
#define LOGGING_NAME "lines"

std::ostream& operator<<(std::ostream& os, const Section* section) {
    os << std::format("[Section {:p}, @[{}, {}), name={}]", static_cast<const void*>(section),
        section->read_start, section->read_end, section->name);
    return os;
}

std::ostream& operator<<(std::ostream& os, const Line* line) {
    os << std::format("[Line {:p}, @{}, sec=", static_cast<const void*>(line),
        line->offset) << line->section << "]";
    return os;
}

Lines::Lines(Buffer& buffer) :
    buffer_(buffer)
{
    current_section_ = new Section();
    batch_size_ = 100000;
}

int64_t Lines::length(Line *line)
{
    int64_t res = 0;
    int64_t size = buffer_.size();
    for (; line->offset + res < size; res++) {
        if (buffer_[line->offset+res] == '\n') {
            res++;
            break;
        }
    }
    return res;
}

void Lines::read_forward()
{
    auto line_start = current_section_->read_end;

    auto first = false;
    auto batch_size = batch_size_;

    auto size = buffer_.size();

    logd("Reading forward in section ", current_section_);

    auto position = current_section_->read_end;
    while (position < size && batch_size > 0) {
        auto c = buffer_[position];
        if (c == '\n' || position == size - 1) {
            auto line = new Line{
                line_start,
                current_section_->next_backward_index,
                current_section_
            };

            // The line at the offset position may already exist in the map. It could be that the
            // buffer have grown and now we're going over the same line.
            auto insert = true;
            if (first) {
                auto pred_it = lines_.lower_bound(line_start);
                if (pred_it != lines_.end() &&
                    pred_it->second->offset + length(pred_it->second) > line_start)
                {
                    insert = false;
                }
                first = false;
            }

            if (insert) {
                lines_.insert({line_start, line});

                current_section_->n_lines++;
                current_section_->next_backward_index++;
                batch_size--;

                logt("Inserted ", line);
            } else {
                delete line;
            }
            position++;
            line_start = position;
            current_section_->read_end = position;
        }
        position++;
    }
}

void Lines::jump_to(int64_t offset, std::string name)
{
    auto it = sections_.lower_bound(offset);
    if (it != sections_.end() && it->first == offset) {
        current_section_ = it->second;
        return;
    }

    if (it != sections_.end()) {
        if ((offset >= it->second->read_start && offset < it->second->read_end)
            || offset == it->second->read_end)
        {
            current_section_ = it->second;
            return;
        }
    }

    auto section = new Section{};
    section->name = name;
    section->read_start = offset;
    section->read_end = offset;
    section->next_forward_index = 1;

    sections_.insert({offset, section});
    current_section_ = section;
}

Line* Lines::first()
{
    if (buffer_.size() == 0) {
        return nullptr;
    }


}

Line* Lines::next(Line* line)
{
    Line* next = nullptr;

    auto it = lines_.upper_bound(line->offset);
    if (it != lines_.end()) {
        next = it->second;
    }

    if (next != nullptr) {
        if (next->section != line->section) {
            auto l = length(line);
            if (line->offset+l == next->offset) {
                return next;
            }
        } else {
            return next;
        }
    }

    it = lines_.upper_bound(line->offset);
    if (it == lines_.end() || line->offset+length(line) < it->second->offset) {
        read_forward();
    }
}

void Lines::test_set_batch_size(int64_t bs)
{
    batch_size_ = bs;
}
