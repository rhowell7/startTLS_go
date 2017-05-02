package main

// go build
// ./smtp

import (
    "net"
    "regexp"
    "fmt"
    "time"
    "bufio"
    // "golang.org/x/net/ipv4"
    // go get -u golang.org/x/net/ipv4
    // "os"
    // "log"
    // "reflect"
)

func main() {
    //---------------------------- Set up ------------------------------//
    // workers := 10 // number of threads, should be a flag


    
    //------------------- Build the queue of IP Addresses --------------------//
    // file, err := os.Open("ipAddresses.txt")
    // if err != nil {
    //     log.Fatal(err)
    // }
    // defer file.Close()

    // ipAddresses := make(chan string)
    // scanner := bufio.NewScanner(file)
    // for scanner.Scan() {
        // fmt.Println(scanner.Text())
        // tmp := string(scanner.Text())
        // fmt.Println(tmp)
        // fmt.Println(reflect.TypeOf(tmp))
        // ipAddresses <- tmp
        // fmt.Println("read one IP")
    // }
    // close(ipAddresses)
    // fmt.Println("Read in IP Addresses\n")

    // if err := scanner.Err(); err != nil {
    //     log.Fatal(err)
    // }

    // fmt.Println("scanner read in ip addresses:\n")
    // for i := 0; i < len(ipAddresses); i++ {
    //     fmt.Println(<-ipAddresses)
    // }


    // -------TODO: get the next IP address from a global queue ---------------------//
    target := "mail.rchowell.net:25"
    // target := "128.32.80.167:25"
    // target := "173.194.69.26:25"
    // target := "128.32.78.35:25"
    // target := "128.248.41.50:25"
    // target := "66.196.118.35:25"
    // target := "65.55.33.135:25"
    // target := "128.32.78.14:25"
    // target := "131.193.46.40:25" // dial timeout

    // minTTL := 10
    // maxTTL := 15

    //------------------------- Open the connection --------------------------//
    fmt.Println("Attempting connection to: ", target)
    timeOut := time.Duration(5) * time.Second
    // Dial: returns a new Client connected to a server at addr
    // conn, err := net.Dial("tcp", target)
    conn, err := net.DialTimeout("tcp", target, timeOut)
    if err != nil {
        fmt.Println("dial error:", err)
        return
    }
    defer fmt.Println("Closing Connection, test")
    defer conn.Close()
    
    // Wait for 220 banner
    banner, err := bufio.NewReader(conn).ReadString('\n')
    fmt.Println(banner)

    // Are we being greylisted/blacklisted?
    bannerGood, err := regexp.MatchString("220 ", string(banner))

    if !bannerGood {
        fmt.Println("This server did not give us a good banner: ", target)
        return
    }

    //---------------------- Send EHLO, receive Extensions -------------------//
    // time.Sleep(100 * time.Millisecond)
    conn.Write([]byte("EHLO ME\r\n"))
    // while "250-*", reading lines. if "250 *", break
    bufReader := bufio.NewReader(conn)
    for {
        // Read tokens delimited by newline
        extensions, err := bufReader.ReadBytes('\n')
        if err != nil { // if there's an error
            fmt.Println()
            break
        }

        fmt.Printf("%s", extensions)

        matched, err := regexp.MatchString("250 ", string(extensions))
        if matched {
            // fmt.Println("Found the last extension\n")
            fmt.Println()
            break
        }

        // 500 5.5.1 Command unrecognized: "XXXX ME"
        ehloError, err := regexp.MatchString("500 ", string(extensions))
        if ehloError {
            fmt.Println("got a 500 error: ", string(extensions))
            fmt.Println("Sending 'HELO' instead")
            conn.Write([]byte("HELO ME\r\n"))
        }
    }


    // timeoutDuration := 1 * time.Second
    // bufReader := bufio.NewReader(conn)
    // for {
    //     // Set a deadline for reading. Read operation will fail if no data
    //     // is received after deadline.
    //     conn.SetReadDeadline(time.Now().Add(timeoutDuration))

    //     // Read tokens delimited by newline
    //     extensions, err := bufReader.ReadBytes('\n')
    //     if err != nil {
    //         fmt.Println()
    //         break
    //     }

    //     fmt.Printf("%s", extensions)
    // }

    //--------------------- Send magic StartTLS packets ----------------------//
    // test regular STARTTLS request
    // time.Sleep(10 * time.Millisecond)
    fmt.Println("Sending STARTTLS")
    conn.Write([]byte("STARTTLS\r\n"))
    tlsResponse, err := bufio.NewReader(conn).ReadBytes('\n')
    fmt.Println(string(tlsResponse))
    // fmt.Println(err)

    starttlsGood, err := regexp.MatchString("220 ", string(tlsResponse))
    starttlsBad, err := regexp.MatchString("500 ", string(tlsResponse))

    if starttlsGood {
        fmt.Println("This server is good to start TLS: ", target)
    } else if starttlsBad {
        fmt.Println("This server is unable to start TLS: ", target)
    } else {
        fmt.Println("Response did not contain 220 or 500, trying again")
        conn.Write([]byte("STARTTLS\n"))
        tlsResponse, err = bufio.NewReader(conn).ReadBytes('\n')
        fmt.Println(string(tlsResponse))
    }



    // Make a new ipv4 connection from the original one
    // startTlsConn := ipv4.NewConn(conn)
    // if err != nil {
    //     fmt.Println("Forking conn error:", err)
    //     return
    // }
    // defer startTlsConn.Close()

    // // Send the fudged StartTLS packets
    // ttl := minTTL
    // for ; ttl < maxTTL; ttl++ {
    //     startTlsConn.SetTTL(ttl)
    //     fmt.Fprintf(startTlsConn, "STARTTLS\r\n") // send packet

    //     // TODO: listen for reply, wait for signal from ICMP listener
    //     time.Sleep(100 * time.Millisecond)
    // }

    //------------------------ Close the connection --------------------------//
    // time.Sleep(1 * time.Second)
    // defer fmt.Println("Closing Connection, test")
    // startTlsConn.SetTTL(60)
    // defer conn.Close()
    // defer startTlsConn.Close()
}
