TARGET = hh

CPP = g++
CPPFLAGS = -Wall --std=c++20 -g -I$(S)
LDFLAGS = -pthread

S = src
O = obj
T = tst

SRCS = $(wildcard $(S)/*.cpp)
TSTS = $(patsubst $(T)/%.cpp, $(O)/%, $(wildcard $(T)/*.cpp))
OBJS = $(patsubst $(S)/%.cpp, $(O)/%.o, $(SRCS))
DEPS = $(patsubst $(S)/%.cpp, $(O)/%.d, $(SRCS))

all: build test

build: $(TARGET)

$(O)/%.o: $(S)/%.cpp
	$(CPP) $(CPPFLAGS) -c $< -o $@ -MMD -MF $(O)/$*.d

$(TARGET): $(OBJS)
	$(CPP) $(LDFLAGS) $(^) -o $(@)

$(O)/test-%: $(T)/test-%.o $(filter-out $(O)/main.o, $(OBJS))
	$(CPP) $(LDFLAGS) $(^) -o $(@)

test: $(TSTS)
	@for test in $(TSTS); do \
		echo "Running $$test"; \
		./$$test; \
		if [ $$? -ne 0 ]; then \
			echo "Test $$test failed."; \
			exit 1; \
		else \
			echo "Test $$test passed."; \
		fi \
	done

-include $(DEPS)

clean:
	rm -rf $(O)/*
	rm -f $(TARGET)

.PHONY: all clean test build
