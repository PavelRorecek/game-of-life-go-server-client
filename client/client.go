package client

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"nhooyr.io/websocket"
	"pavelrorecek.com/ebitenginetest/shared"
)

type Game shared.Game

func (g *GameWrapper) Update() error {
	return nil
}

func (w *GameWrapper) Draw2(pix []byte) {
	for i, v := range w.game.Area {
		if v {
			pix[4*i] = 0xff
			pix[4*i+1] = 0xff
			pix[4*i+2] = 0xff
			pix[4*i+3] = 0xff
		} else {
			pix[4*i] = 0
			pix[4*i+1] = 0
			pix[4*i+2] = 0
			pix[4*i+3] = 0
		}
	}
}

func (g *GameWrapper) Draw(screen *ebiten.Image) {
	if g.pix == nil {
		g.pix = make([]byte, shared.WorldWidth*shared.WorldHeight*4)
	}
	g.Draw2(g.pix)
	screen.WritePixels(g.pix)
	// log.Println("Drawing:", g.pix)
}

func (g *GameWrapper) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return shared.WorldWidth, shared.WorldHeight
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func StartClient() {
	url := "ws://localhost:8000"
	ctx := context.Background()
	c, resp, err := websocket.Dial(ctx, url, nil)
	check(err)

	log.Println("Connection response:", resp)

	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)

	const MaxMsgSize int = 4 * 1024

	gameWrap := GameWrapper{}
	go func() {
		msg := make([]byte, MaxMsgSize)
		for {
			n, err := conn.Read(msg)

			if err != nil {
				log.Println("Error reading", err)
				return
			}

			var res shared.Game
			var e error = json.Unmarshal(msg[:n], &res)

			log.Println("Client message(bytes)", string(msg[:n]))
			// log.Println("Client message(string)", string(msg[:n]))
			if e == nil {
				log.Println("Client message(world)", res)
				gameWrap.game = res
			} else {
				log.Println("Client message(world) err", e)
			}
		}
	}()

	go func() {
		counter := byte(0)
		for {
			time.Sleep(1 * time.Second)
			n, err := conn.Write([]byte{counter})
			if err != nil {
				log.Println("Error sending", err)
				return
			}
			log.Println("Sent n bytes", n)
			counter++
		}
	}()

	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Hello, World!")
	if err := ebiten.RunGame(&gameWrap); err != nil {
		log.Fatal(err)
	}
}

type GameWrapper struct {
	game shared.Game
	pix  []byte
}
