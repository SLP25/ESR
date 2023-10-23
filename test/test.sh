#!/bin/bash

# Make all scripts executable
find test/ -name "*.sh" -exec chmod +x {} \;

CORE_ID=38487
BASE_DIR=/home/core/tp2


for MACHINE in test/$1/*.sh; do
   NAME=$(basename $MACHINE .sh)
   echo $NAME
   vcmd -I -c /tmp/pycore.$CORE_ID/$NAME bash "$BASE_DIR/test/$1/$NAME.sh" &
done

#vcmd -c /tmp/pycore.1/Aladdin