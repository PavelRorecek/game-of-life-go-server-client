package server

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"nhooyr.io/websocket"
	"pavelrorecek.com/ebitenginetest/shared"
)

type Game shared.Game

func NewGame(width, height int, maxInitLiveCells int) *Game {
	w := &Game{
		Area:   make([]bool, width*height),
		Width:  width,
		Height: height,
	}
	w.init(maxInitLiveCells)
	return w
}

// init inits world with a random state.
func (w *Game) init(maxLiveCells int) {
	for i := 0; i < maxLiveCells; i++ {
		x := rand.Intn(w.Width)
		y := rand.Intn(w.Height)
		w.Area[y*w.Width+x] = true
	}
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// neighbourCount calculates the Moore neighborhood of (x, y).
func neighbourCount(a []bool, width, height, x, y int) int {
	c := 0
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			if i == 0 && j == 0 {
				continue
			}
			x2 := x + i
			y2 := y + j
			if x2 < 0 || y2 < 0 || width <= x2 || height <= y2 {
				continue
			}
			if a[y2*width+x2] {
				c++
			}
		}
	}
	return c
}

// Update game state by one tick.
func (w *Game) Update() {
	Width := w.Width
	Height := w.Height
	next := make([]bool, Width*Height)
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			pop := neighbourCount(w.Area, Width, Height, x, y)
			switch {
			case pop < 2:
				// rule 1. Any live cell with fewer than two live neighbours
				// dies, as if caused by under-population.
				next[y*Width+x] = false

			case (pop == 2 || pop == 3) && w.Area[y*Width+x]:
				// rule 2. Any live cell with two or three live neighbours
				// lives on to the next generation.
				next[y*Width+x] = true

			case pop > 3:
				// rule 3. Any live cell with more than three live neighbours
				// dies, as if by over-population.
				next[y*Width+x] = false

			case pop == 3:
				// rule 4. Any dead cell with exactly three live neighbours
				// becomes a live cell, as if by reproduction.
				next[y*Width+x] = true
			}
		}
	}
	w.Area = next
}

func StartServer() {
	listener, err := net.Listen("tcp", ":8000")

	if err != nil {
		panic(err)
	}

	s := &http.Server{
		Handler:      websocketServer{},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Starting Server", listener.Addr())

	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(listener)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	select {
	case err := <-errc:
		log.Println("Failed to serve: ", err)
	case sig := <-sigs:
		log.Println("Terminating: ", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.Shutdown(ctx)
	if err != nil {
		log.Println("Error shutting down server:", err)
	}
}

type websocketServer struct {
}

func (s websocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println("Error accepting websocket", err)
		return
	}
	ctx := context.Background()
	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	go ServeNetConn(conn)
}

func ServeNetConn(conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Println("Error closing net.Conn", err)
		}
	}()

	timeoutSeconds := time.Minute
	timeout := make(chan uint8, 1)
	const StopTimeout uint8 = 0
	const ContTimeout uint8 = 1
	const MaxMsgSize int = 4 * 1024

	go func() {
		msg := make([]byte, MaxMsgSize)
		for {
			n, err := conn.Read(msg)

			if err != nil {
				log.Println("Error reading", err)
				timeout <- StopTimeout
				return
			}

			timeout <- ContTimeout

			log.Println("Server message", msg[:n])
		}
	}()

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		world := NewGame(shared.WorldWidth, shared.WorldHeight, 20)

		for range ticker.C {
			world.Update()

			regenerate := false
			for _, item := range world.Area {
				if item == true {
					regenerate = true
					break
				}
			}
			if regenerate {
				world = NewGame(shared.WorldWidth, shared.WorldHeight, 20)
			}

			b, err := json.Marshal(world)

			if err != nil {
				log.Print("Error sending world:", err)
			}

			// log.Println(string(b))

			conn.Write(b)
		}
	}()

ExitTimeout:
	for {
		select {
		case res := <-timeout:
			if res == StopTimeout {
				log.Println("manually stopping timeout manager")
				break ExitTimeout
			}
		case <-time.After(timeoutSeconds):
			log.Println("User timed out")
			break ExitTimeout
		}
	}
}
