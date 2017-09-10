# startTLS_go #

## Brief explanation of the program ##

The goal is to find certain IP Addresses that are blocking email encryption. This program runs a sort of traceroute to a mail server and asks each hop along the way to encrypt the connection. The bad-guys who block the encryption use a simple find-and-replace that changes "`STARTTLS`" into "`XXXXXXXX`", so once we see the `X`'s, we infer that the previous hop is blocking the `STARTTLS` packet. A trick that helps us out: once the TTL expires on a packet, the last hop holding the packet is *supposed* to return it to the original sender (us), encapsulated in a "time-expired" ICMP packet. So we catch those ICMP packets, parse them, and return them to their proper goroutines. Once we reach the target IP (the traceroute is finished), that goroutine closes the connection and grabs a new IP address to try.

We use 3 main channels for communicating:
`input_chan` reads in IP addresses from a file, workers take them out
`icmp_chan` holds ICMPPacket objects that have already been parsed, and delivers them to their proper worker
`output_chan` holds Output objects that contain the JSON-style data we want to know about each completed traceroute: number of hops, if it was censored, and which IP Address is blocking the encryption

The program flow goes roughly:
1. Fill the `input_chan` with IP Addresses
2. Create the ICMP Listener
3. Spawn a bunch of workers (goroutines). Each worker will:
    1. Grab an IP from the `input_chan`
    2. Open an SMTP connection to that IP
    3. Run the traceroute with the STARTTLS packet. Each hop needs to:
        - Wait for the ICMP Dispatcher to return the response packet
        - Break when we've reached the target_ip
        - Increment the TTL
        - Send the next packet in the trace
    4. Put the JSON data into results.txt
    5. Break when there are no more IP addresses in the `input_chan`

## How to use ##
Build it: `go build`

Run default: `sudo ./smtp`

Run with options: `sudo ./smtp --input-file ipAddresses.txt --output-file results.txt --workers 10`
