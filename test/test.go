package main
 
import (
    "fmt"
    "os"
    "net"
)
 
// Prints error if present and exits if specified
func checkError(err error, exitIfError bool) {
    if err != nil {
        fmt.Println("ERROR!!: " , err)

        if exitIfError {
            os.Exit(0)
        }
    }
}
 
func main() {
    localAddress, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")  // Random port, technically should see if open
    checkError(err, false)

    serverAddress,err := net.ResolveUDPAddr("udp","127.0.0.1:69")
    checkError(err, false)
 
    connection, err := net.DialUDP("udp", localAddress, serverAddress)
    checkError(err, false)
 
    defer connection.Close()

    //var message []byte
    //message = make([]byte,0,1,0,0,102,0)  // a read request for file f
    message := []byte{0,1,102,0}

    //msg := strconv.Itoa(i)
    //buf := []byte(msg)
    connection.Write(message)
}