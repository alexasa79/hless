TARGET = hh

CPP = g++
CPPFLAGS = -O3 -Wall --std=c++20 -g -I$(S) -DTB_OPT_ATTR_W=64
LDFLAGS =

S = src
O = out
T = tst

SRCS = $(wildcard $(S)/*.cpp)
OBJS = $(patsubst $(S)/%.cpp, $(O)/$(S)/%.o, $(SRCS))
DEPS = $(patsubst $(S)/%.cpp, $(O)/$(S)/%.d, $(SRCS))
TSTS = $(patsubst $(T)/%.cpp, $(O)/$(T)/%, $(wildcard $(T)/*.cpp))

all: build test

build: $(TARGET)

$(O)/$(S)/%.o: $(S)/%.cpp
	@mkdir -p $(O)/$(S)
	$(CPP) $(CPPFLAGS) -c $< -o $@ -MMD -MF $(O)/$*.d

$(O)/$(T)/%.o: $(T)/%.cpp
	@mkdir -p $(O)/$(T)
	$(CPP) $(CPPFLAGS) -c $< -o $@ -MMD -MF $(O)/$*.d

$(TARGET): $(OBJS)
	$(CPP) $(LDFLAGS) $(^) -o $(@)

$(O)/%-test: $(O)/%-test.o $(filter-out $(O)/$(S)/main.o, $(OBJS))
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
