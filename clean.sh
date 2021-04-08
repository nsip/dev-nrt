#!/bin/bash

set -e

rm -rf ./build
rm -f NRT-*-v*.*.*.zip

# delete all Linux binary files
find . -type f -executable -exec sh -c "file -i '{}' | grep -q 'x-executable; charset=binary'" \; -print | xargs rm -f

# delete all Mac binary files
find . -type f -executable -exec sh -c "file -i '{}' | grep -q 'x-mach-binary; charset=binary'" \; -print | xargs rm -f

# delete windows executables
find . -type f -executable -exec sh -c "file -i '{}' | grep -q 'x-dosexec; charset=binary'" \; -print | xargs rm -f

# delete others
for f in $(find ./ -name '*.log' -or -name '*.doc'); do rm $f; done