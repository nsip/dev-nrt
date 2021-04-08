#!/bin/bash

set -e

rm -rf ./build
rm -f NRT-*-v*.*.*.zip

# delete all binary files
find . -type f -executable -exec sh -c "file -i '{}' | grep -q 'x-executable; charset=binary'" \; -print | xargs rm -f

# delete windows executables
for f in $(find ./ -name '*.log' -or -name '*.exe'); do rm $f; done 