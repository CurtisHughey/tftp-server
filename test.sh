#!/bin/sh

cp test/files/f f
cp test/files/g g

sudo go run server.go > test/log/runOutput.log &

sleep 2  # sleeps for 2 seconds to get server started

# Runs tftp commands to check that a full write and read work correctly
tftp <<EOF > test/log/output.log
connect localhost
mode binary
put f cf
get g cg
put writefiledoesntexist
quit
EOF

echo "RESULTS"

sudo go run test/test.go  # Runs some error-checking commands in go

pidInfo=`sudo netstat -nlp | grep 69 | awk '{print $6}' | head -1`
serverPID="${pidInfo%%/*}"
sudo kill -9 $serverPID

sleep 1  # To fully kill

if cmp -s "f" "cf"
then
   echo "Writing: OK"
else
   echo "Writing: ERROR"
   echo "Target:"
   cat f
   echo ""
   echo "Actual:"
   cat cf
   echo ""
fi

if cmp -s "g" "g"
then
   echo "Reading: OK"
else
   echo "Reading: ERROR"
   echo "Target:"
   cat cg
   echo ""
   echo "Actual:"
   cat g
   echo ""
fi

rm -f f
rm -f g
rm -f cf
rm -f cg