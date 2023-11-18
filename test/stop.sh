PID=$(ps | tail -7 | head -1 | awk '{print $1;}')

if [ "$PID" != "1" ] ; then
        kill $PID
        sleep .1
    fi