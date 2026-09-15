package e2e

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NATSMessage is a message delivered on a subscribed subject.
type NATSMessage struct {
	Subject string
	Payload []byte
}

// NATSSubscriber is a minimal NATS client that speaks the wire protocol
// directly.
//
// The E2E module stays dependency-free on purpose: pulling a client library
// into the deployment repository just to assert on a handful of published
// events would add a supply-chain surface for no benefit. Only the subset the
// scenarios need is implemented — CONNECT, SUB, MSG and PING/PONG.
type NATSSubscriber struct {
	conn     net.Conn
	messages chan NATSMessage

	writeMu sync.Mutex
	closeMu sync.Mutex
	closed  bool
}

// SubscribeNATS connects to addr and subscribes to subject, which may contain
// the NATS wildcards `*` and `>`.
func SubscribeNATS(addr, subject string) (*NATSSubscriber, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial NATS at %s: %w", addr, err)
	}
	subscriber := &NATSSubscriber{conn: conn, messages: make(chan NATSMessage, 64)}

	reader := bufio.NewReader(conn)
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		conn.Close()
		return nil, err
	}
	info, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read NATS INFO: %w", err)
	}
	if !strings.HasPrefix(info, "INFO ") {
		conn.Close()
		return nil, fmt.Errorf("unexpected NATS greeting: %q", strings.TrimSpace(info))
	}

	if err := subscriber.send(`CONNECT {"verbose":false,"pedantic":false,"tls_required":false,"name":"bsystem-e2e","lang":"go"}`); err != nil {
		conn.Close()
		return nil, err
	}
	if err := subscriber.send("SUB " + subject + " 1"); err != nil {
		conn.Close()
		return nil, err
	}
	// PING/PONG round-trips confirm the subscription is registered before the
	// caller triggers the action it expects to produce an event.
	if err := subscriber.send("PING"); err != nil {
		conn.Close()
		return nil, err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("await NATS PONG: %w", err)
		}
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "PONG") {
			break
		}
		if strings.HasPrefix(trimmed, "-ERR") {
			conn.Close()
			return nil, errors.New("NATS rejected the subscription: " + trimmed)
		}
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}
	go subscriber.readLoop(reader)
	return subscriber, nil
}

func (s *NATSSubscriber) send(command string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	if _, err := s.conn.Write([]byte(command + "\r\n")); err != nil {
		return fmt.Errorf("write NATS command: %w", err)
	}
	return nil
}

func (s *NATSSubscriber) readLoop(reader *bufio.Reader) {
	defer close(s.messages)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.EqualFold(trimmed, "PING"):
			if err := s.send("PONG"); err != nil {
				return
			}
		case strings.HasPrefix(trimmed, "MSG "):
			message, err := readMessage(reader, trimmed)
			if err != nil {
				return
			}
			select {
			case s.messages <- message:
			default:
				// A full buffer means the scenario is not consuming; dropping
				// is preferable to blocking the reader forever.
			}
		}
	}
}

// readMessage parses `MSG <subject> <sid> [reply-to] <#bytes>` followed by the
// payload and its trailing CRLF.
func readMessage(reader *bufio.Reader, header string) (NATSMessage, error) {
	fields := strings.Fields(header)
	if len(fields) < 4 {
		return NATSMessage{}, fmt.Errorf("malformed MSG header: %q", header)
	}
	size, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil || size < 0 {
		return NATSMessage{}, fmt.Errorf("malformed MSG payload size in %q", header)
	}
	// +2 consumes the CRLF that terminates the payload.
	payload := make([]byte, size+2)
	if _, err := readFull(reader, payload); err != nil {
		return NATSMessage{}, err
	}
	return NATSMessage{Subject: fields[1], Payload: payload[:size]}, nil
}

func readFull(reader *bufio.Reader, buffer []byte) (int, error) {
	read := 0
	for read < len(buffer) {
		n, err := reader.Read(buffer[read:])
		read += n
		if err != nil {
			return read, err
		}
	}
	return read, nil
}

// Next waits for the next message, or reports why none arrived.
func (s *NATSSubscriber) Next(timeout time.Duration) (NATSMessage, error) {
	select {
	case message, ok := <-s.messages:
		if !ok {
			return NATSMessage{}, errors.New("NATS connection closed before a message arrived")
		}
		return message, nil
	case <-time.After(timeout):
		return NATSMessage{}, fmt.Errorf("no NATS message within %s", timeout)
	}
}

// Close releases the connection.
func (s *NATSSubscriber) Close() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	_ = s.conn.Close()
}
