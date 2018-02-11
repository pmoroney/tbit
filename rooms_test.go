package main

import "testing"

func TestJoinAndLeave(t *testing.T) {
	room := NewRoom("testRoom")
	conn := &Conn{id: 123}
	room.Join(conn)
	if len(room.Conns) != 1 {
		t.Fatalf("expected one connection in the room, got %d", len(room.Conns))
	}
	if room.Conns[123] != conn {
		t.Fatalf("conn id 123 is not in the list of connections for the room")
	}
	room.Leave(conn)
	if len(room.Conns) != 0 {
		t.Fatalf("expected zero connections in the room, got %d", len(room.Conns))
	}
}
