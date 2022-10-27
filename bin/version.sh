#!/bin/bash
git describe --tags --abbrev=0 > version.txt
date +%FT%T%z > build.txt