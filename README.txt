TFTP Server written in Go by Curtis Hughey.

All server code is written in server.go, see comments inside for documentation.  My test suite is a little sparse, it doesn't check for timeout ack/data messages being sent, although it can be manually tested pretty easily, and it doesn't test for a full range of error messages.  The server code itself of course handles all relevant errors.

To run the server, run
sudo go run server.go
We need sudo because the server runs on port 69.  Note this means that if another process is already using 69 the server will error out, since I figured you wouldn't want me randomly killing processes on your computer.  If another process is using 69, just do
> sudo netstat -nlp | grep 69
And then get the PID and kill it
If it's really awkward to be using 69, you can change it in the list of constants at the top of server.go.  You'd also have to change the test code to make sure that 


To run the test code, run:
> sudo ./test.sh
Ditto as above with port 69.
My bash skills aren't great, so apologies for janky scripting.  Also, never used Go before, so there are definitely parts of the code that probably look weird.  The test code requires the xinetd tftp is installed.  If all is successful, output should look something like:
> sudo ./test.sh
tftp: writefiledoesntexist: No such file or directory
RESULTS
signal: Killed
Writing: OK
Reading: OK
>
