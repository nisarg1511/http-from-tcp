package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	listner, err := net.Listen("tcp", "localhost:8080")

	if err != nil {
		log.Fatalf("Failed to start server:%v", err)
	}

	defer listner.Close()
	log.Printf("Server started successfully on port:8080")

	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		return
	}

	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")

	// Replace your old check with this:
	if len(parts) != 3 {
		badResponse := "HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\nInvalid HTTP Request format."
		conn.Write([]byte(badResponse))
		return
	}

	method := parts[0]
	resource := parts[1]
	version := parts[2]

	if version == "HTTP/1.1" {
		log.Printf("[Routing to PERSISTENT] %s %s %s", method, resource, version)
		handlePersistent(conn, reader, method, resource, version)
	} else {
		log.Printf("[Routing to NON-PERSISTENT] %s %s %s", method, resource, version)
		handleNonPersistent(conn, reader, method, resource, version)
	}

}
func handleNonPersistent(conn net.Conn, reader *bufio.Reader, method, resource, version string) {

	defer conn.Close()
	body := "We have recieved a " + method + " request for the resource:" + resource + "\n" + version + "\n"
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"Connection: close\r\n" + // Tell the browser to close the TCP connection
		"\r\n"
	for {
		hLine, err := reader.ReadString('\n')
		if err != nil || hLine == "\r\n" || hLine == "\n" {
			break
		}
	}
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD":
		response = response + body
	default:
		response = response + "Invalid Request!"
	}

	conn.Write([]byte(response))
}

func handlePersistent(conn net.Conn, reader *bufio.Reader, method, resource, version string) {

	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	for {

		for {
			headerLine, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if headerLine == "\r\n" || headerLine == "\n" {
				break
			}
		}

		body := "We have recieved a " + method + " request for the resource:" + resource + "\n" + version + "\n"
		response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
			"Connection: keep-alive\r\n" + // Tell the browser to close the TCP connection
			"\r\n"

		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD":
			response = response + body
		default:
			response = response + "Invalid Request!"
		}
		conn.Write([]byte(response))

		conn.SetDeadline(time.Now().Add(5 * time.Second))

		nextLine, err := reader.ReadString('\n')
		if err != nil {
			// Expected timeout error or EOF when the client safely closes the connection
			return
		}

		nextLine = strings.TrimSpace(nextLine)
		parts := strings.Split(nextLine, " ")
		if len(parts) != 3 {
			return
		}

		// Update variables for the next loop execution
		method = parts[0]
		resource = parts[1]
	}
}
