#include <iostream>
#include <fstream>
#include "logging.h"

std::ofstream ___logf;

void setup_logging(std::string filename)
{
    ___logf.open(filename);
    if (!___logf.is_open()) {
        std::cerr << "Failed to open a file" << std::endl;
    }
}

void shutdown_logging()
{
    ___logf.close();
}
