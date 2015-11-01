package main
 
import (
    "fmt"
    "net"
    "os"
    "io/ioutil"
    "time"
    "encoding/binary"
    "reflect"
)

const (
    verbose = false
    serverPort = "69"
    connectionType = "udp"
    timeoutTime = 1
)

const (
    opcode_rrq   = 01
    opcode_wrq   = 02
    opcode_data  = 03
    opcode_ack   = 04
    opcode_error = 05
)

// Add one for constants

func main() {
    fmt.Println("Hi there!")
    createServer()
}

// Prints error if present and exits if specified
func checkError(err error, exitIfError bool) {
    if err != nil {
        fmt.Println("ERROR!!: " , err)

        if exitIfError {
            os.Exit(0)
        }
    }
}

// Creates the address, connection, and begins listening
func createServer() {

    /* If it can't open the port, find the process using it with
        > sudo netstat -nlp | grep 69
        Then kill the PID (might need sudo as well)
    */
    serverAddress, err := net.ResolveUDPAddr(connectionType,":"+serverPort)
    checkError(err, true)
 
    serverConnection, err := net.ListenUDP(connectionType, serverAddress)
    checkError(err, true)

    defer serverConnection.Close()
 
    buffer := make([]byte, 1024)
 
    for {
        _, clientAddress, err := serverConnection.ReadFromUDP(buffer)
        checkError(err, false)
        
        if err == nil {      
            if (buffer[1] == opcode_rrq || buffer[1] == opcode_wrq) { // Then it is valid        
                var filename = make([]byte, 0)
                var size = 0
                for i := 2; buffer[i] != 0; i++ {
                    filename = append(filename, buffer[i])
                    size += 1
                }
                go handleConnection(filename, size, clientAddress, buffer[1])  // buffer[1] is the opcode
            } else {
                errorPacket := makeERROR(4) // Illegal TFTP op
                _, err = serverConnection.WriteToUDP(errorPacket, clientAddress) 
            }
        }
    }
}

func handleConnection(filename []byte, filenameSize int, clientAddress *net.UDPAddr, opcode byte)  {

    allocatedAddress,err := net.ResolveUDPAddr(connectionType,":0")
    allocatedConnection, err := net.ListenUDP(connectionType, allocatedAddress)  // Apparently, passing in 0 allocates a port for us that's free
    checkError(err, true)

    defer allocatedConnection.Close()

    fileStringName := string(filename[:filenameSize])

    if opcode == opcode_rrq {  // Then we need to start writing

        serverConnection, err := net.ListenUDP(connectionType, allocatedAddress)
        checkError(err, true)

        _, err = os.Stat(fileStringName)

        if err != nil {  // Strangely, this isn't working in the test code, but it's fine when I run it
            errorPacket := makeERROR(1)  // File cannot be read, probably because it doesn't exist (could also be a permission thing, maybe)
            _, _ = serverConnection.WriteToUDP(errorPacket, clientAddress) 
            return
        }

        defer serverConnection.Close()

        fileContents, err := ioutil.ReadFile(fileStringName)
        checkError(err, false)

        serverConnection.SetReadDeadline(time.Now().Add(timeoutTime * time.Second))  // Change to something more appropriate

        blockid := 1
        endOfFile := false
        for {
            var outBuf []byte

            blockidBytes := make([]byte, 4)
            binary.BigEndian.PutUint32(blockidBytes, uint32(blockid))

            if len(fileContents) >= blockid*512 {  // Then not done after this buffer yet
                endOfFile = false
                outBuf = makeDATA(fileContents[((blockid-1)*512):(blockid*512)], blockidBytes)
            } else {
                endOfFile = true
                outBuf = makeDATA(fileContents[((blockid-1)*512):(len(fileContents))], blockidBytes)
            }

            _, err = serverConnection.WriteToUDP(outBuf, clientAddress)  // Sends it along
            checkError(err,true)

            inBuf := make([]byte, 1024)  // More than enough for any valid packet
            _, clientAddressRec, err := serverConnection.ReadFromUDP(inBuf)
            checkError(err,false)

            if err != nil {
                serverConnection.SetReadDeadline(time.Now().Add(timeoutTime * time.Second))   // Resetting
            }

            if !reflect.DeepEqual(clientAddressRec, clientAddress) {
                errorPacket := makeERROR(5) // Unknown TID
                _, err = serverConnection.WriteToUDP(errorPacket, clientAddressRec) 
            }

            if err != nil {  // Need to resend last packet, assuming timeout
                continue // Nothing else needs to be done
            } else { // Then blockid is incremented, as long as the message is an ack with the right blockid
                if (inBuf[1] == opcode_ack && inBuf[2] == blockidBytes[2] && inBuf[3] == blockidBytes[3]) {  //Then correct ack, not checking number correctly
                    if endOfFile == true {  // Then we're done here
                        break
                    } else {
                        blockid += 1
                    }
                } else { // Illegal op
                    errorPacket := makeERROR(4)
                    _,_ = serverConnection.WriteToUDP(errorPacket, clientAddress)
                }
            }
        }
        _ = serverConnection.Close()  // Not worth catching error, what would we do?
    } else if opcode == opcode_wrq {  // Then we need to start reading (this one's a bit easier).  Note we don't write to file until everything is read
        serverConnection, err := net.ListenUDP(connectionType, allocatedAddress)
        checkError(err, true)

        defer serverConnection.Close()

        if _, err := os.Stat(fileStringName); err == nil {  // Then file already exists
            errorPacket := makeERROR(6)
            _,_ = serverConnection.WriteToUDP(errorPacket, clientAddress)
            return
        }

        serverConnection.SetReadDeadline(time.Now().Add(timeoutTime * time.Second))

        blockid := 0
        endOfFile := false

        var fileBuffer []byte
        fileBuffer = nil

        for {
            blockidBytes := make([]byte, 4)
            binary.BigEndian.PutUint32(blockidBytes, uint32(blockid))
            outBuf := makeACK(blockidBytes)

            _, err = serverConnection.WriteToUDP(outBuf, clientAddress)  // Sends it along
            checkError(err,true)

            if endOfFile == true {  // Nothing more to do, we sent the ack (technically we could wait to see if they didn't get the ack, but I'm making this easy on myself)
                err = ioutil.WriteFile(fileStringName, fileBuffer, 0777)
                checkError(err, false)
                break
            }

            inBuf := make([]byte, 1024)  // More than enough for any valid packet
            n, clientAddressRec, err := serverConnection.ReadFromUDP(inBuf)  // Implement checking for correct addr
            checkError(err, false)

            if err != nil {
                serverConnection.SetReadDeadline(time.Now().Add(timeoutTime * time.Second))   // Resetting
            }
            
            if !reflect.DeepEqual(clientAddressRec, clientAddress) {
                errorPacket := makeERROR(5) // Unknown TID
                _, err = serverConnection.WriteToUDP(errorPacket, clientAddressRec) // clientAddressRec since someone else
            }

            nextBlockidBytes := make([]byte, 4)
            binary.BigEndian.PutUint32(nextBlockidBytes, uint32(blockid+1))

            if (err != nil) {  // We timed out, resend and try again
                continue
            } else if (inBuf[1] == opcode_data && inBuf[2] == nextBlockidBytes[2] && inBuf[3] == nextBlockidBytes[3]) {  // Then the data message is good, not checking number correctly!
                fileBuffer = append(fileBuffer, inBuf[4:n]...)
                blockid += 1
                endOfFile = (n < 516)  
            } else { // Illegal op (also could have been wrong ack, not sure what I would do in that case)
                errorPacket := makeERROR(4)  // Illegal TFTP Operation
                _,_ = serverConnection.WriteToUDP(errorPacket, clientAddress)
            }
        }

        _ = serverConnection.Close()  // Not worth catching error, what would we do?
    } else { // Illegal op, technically the function that called this would already have taken care of it
        errorPacket := makeERROR(4) // Illegal TFTP Operation
        _,_ = allocatedConnection.WriteToUDP(errorPacket, allocatedAddress)
    }
}

func makeDATA(input []byte, blockid []byte) ([]byte) {
    header := make([]byte, 4)
    header[0] = 0
    header[1] = opcode_data
    header[2] = blockid[2]
    header[3] = blockid[3]

    output := append(header[:], input[:]...)
    return output
}

func makeACK(blockid []byte) ([]byte) { 
    output := make([]byte, 4)
    output[0] = 0
    output[1] = opcode_ack
    output[2] = blockid[2]
    output[3] = blockid[3]

    return output
}

func makeERROR(errorCode byte) ([]byte) {
    header := make([]byte, 4)
    header[0] = 0
    header[1] = opcode_error
    header[2] = 0
    header[3] = errorCode

    message := []byte("ERROR")
    output := append(header[:], message[:]...)
    return output
}