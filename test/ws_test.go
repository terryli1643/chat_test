package test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestBroadcast(t *testing.T) {
	clear()

	var wg sync.WaitGroup
	wg.Add(1)

	c, err := conn()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	go func() {
		defer wg.Done()
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Error(err)
		}
		t.Log(string(msg))
		assert.Equal(t, "test 123", string(msg))

	}()

	c.WriteMessage(websocket.TextMessage, []byte("test 123"))
	wg.Wait()
}

// 此测试用例需要确保服务器没有历史消记录
func TestProfanityWords(t *testing.T) {
	clear()

	var wg sync.WaitGroup
	wg.Add(1)

	c, err := conn()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	go func() {
		defer wg.Done()
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Error(err)
		}
		t.Log(string(msg))
		assert.Equal(t, "test * 123", string(msg))

	}()

	c.WriteMessage(websocket.TextMessage, []byte("test shit 123"))
	wg.Wait()
}

func TestRead50MessageWhenConn(t *testing.T) {
	clear()

	var wg sync.WaitGroup
	wg.Add(1)

	c, err := conn()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	for i := 0; i < 50; i++ {
		c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("test %d", i)))
	}

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, msg, err := c.ReadMessage()
			if err != nil {
				t.Error(err)
			}
			t.Log(string(msg))
		}

	}()

	wg.Wait()
}

func TestPopularCmd(t *testing.T) {
	fillHistory()
	clear()

	var wg sync.WaitGroup
	wg.Add(1)

	c, err := conn()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	go func() {

		defer wg.Done()
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "/popular test", string(msg))

	}()

	c.WriteMessage(websocket.TextMessage, []byte("/popular"))
	wg.Wait()
}

func TestStatsCmd(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	c, err := conn()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	go func() {

		defer wg.Done()
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Error(err)
		}
		t.Log(string(msg))

	}()

	c.WriteMessage(websocket.TextMessage, []byte("/stats"))
	wg.Wait()
}

func conn() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:5000/ws", nil)
	return c, err
}

func clear() {
	c, err := conn()
	if err != nil {
		return
	}
	defer c.Close()

	c.WriteMessage(websocket.TextMessage, []byte("/clear"))
}

func fillHistory() {
	clear()
	c, err := conn()
	if err != nil {
		return
	}
	defer c.Close()
	c.WriteMessage(websocket.TextMessage, []byte("test 1"))
	c.WriteMessage(websocket.TextMessage, []byte("test 2"))
	c.WriteMessage(websocket.TextMessage, []byte("test 3"))
	c.WriteMessage(websocket.TextMessage, []byte("test 4"))
	c.WriteMessage(websocket.TextMessage, []byte("test 1"))
	c.WriteMessage(websocket.TextMessage, []byte("test 2"))
	c.WriteMessage(websocket.TextMessage, []byte("test 3"))
	c.WriteMessage(websocket.TextMessage, []byte("test 4"))
	c.WriteMessage(websocket.TextMessage, []byte("test 1"))
	c.WriteMessage(websocket.TextMessage, []byte("test 2"))
	c.WriteMessage(websocket.TextMessage, []byte("test 3"))
	c.WriteMessage(websocket.TextMessage, []byte("test 4"))
}
