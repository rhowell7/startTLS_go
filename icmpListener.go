package main

import (
    "golang.org/x/net/icmp"
    "log"
    "net"
    "fmt"
    "golang.org/x/net/ipv4"
    // go get -u golang.org/x/net/ipv4
)

func main() {
	//-------------------------- Open the listener ---------------------------//
    conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
    if err != nil {
        log.Fatal(err)
    }

    //----------------------- Listen for ICMP Packets ------------------------//
    for {
		rb := make([]byte, 1500)

		n, peer, err := conn.ReadFrom(rb)
		if err != nil {
	        if err, ok := err.(net.Error); ok && err.Timeout() {
	            fmt.Printf("%v\t*\n", 3)
	            continue
	        }
	        log.Fatal(err)
	    }
	    rm, err := icmp.ParseMessage(1, rb[:n])
	    if err != nil {
	        log.Fatal(err)
	    }

	    switch rm.Type {
	    case ipv4.ICMPTypeTimeExceeded:
	        fmt.Printf("\npeer: %v\n", peer)
	        body := rm.Body.(*icmp.TimeExceeded)
	        text := string(body.Data)
	        if len(text) > 52 {
	        	fmt.Printf("ICMP response: %s\n", text[52:])
	        }
	    case ipv4.ICMPTypeEchoReply:
	        names, _ := net.LookupAddr(peer.String())
	        fmt.Printf("\t%v %+v \n\t%+v\n", peer, names, rm)
	        return
	    default:
	        log.Printf("unknown ICMP message: %+v\n", rm)
	    }

    }
    _ = conn.Close()
}