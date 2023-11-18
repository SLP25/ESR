#!/bin/bash

if [ "$1" == "" ]; then
      echo You must specify the test to run as the first argument
      exit 1 
   fi 

if [ "$2" != "start" -a "$2" != "stop" -a "$2" != "" ]; then
      echo Invalid operation: $2
      exit 1 
   fi 

# Make all scripts executable
find test/ -name "*.sh" -exec chmod +x {} \;

#Core config
CORE_ID=1
BASE_DIR=/src/ESR/

if [ "$2" != "stop" ] ; then

      #Clear logs
      rm -f test/$1/logs/*.log

      for MACHINE in test/$1/*.sh ; do
         NAME=$(basename $MACHINE .sh)

         if [ "$NAME" == "common" ] ; then
            continue;
         fi

         vcmd -I -c /tmp/pycore.$CORE_ID/$NAME bash "${BASE_DIR}test/run.sh" $1 "$BASE_DIR" "$NAME" &
      done
   fi


if [ "$2" == "" ] ; then
      read
   fi


if [ "$2" != "start" ] ; then
      for MACHINE in test/$1/*.sh ; do
         NAME=$(basename $MACHINE .sh)

         if [ "$NAME" == "common" ] ; then
            continue;
         fi

         vcmd -I -c /tmp/pycore.$CORE_ID/$NAME bash "${BASE_DIR}test/stop.sh"
      done
   fi