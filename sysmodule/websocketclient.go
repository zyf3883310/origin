package sysmodule

import (
	"fmt"
	"gorilla/websocket"
	"log"
	"net/http"
	"net/url"

	"time"
)

type IWebsocketClient interface {
	Init(slf IWebsocketClient, strurl string, bproxy bool, timeoutsec time.Duration) error
	Start() error
	WriteMessage(msg []byte) error
	OnDisconnect() error
	OnConnected() error
	OnReadMessage(msg []byte) error
}

type WebsocketClient struct {
	WsDailer   *websocket.Dialer
	conn       *websocket.Conn
	url        string
	state      int //0未连接状态   1连接状态
	bwritemsg  chan []byte
	slf        IWebsocketClient
	timeoutsec time.Duration
}

func (ws *WebsocketClient) Init(slf IWebsocketClient, strurl string, bproxy bool, timeoutsec time.Duration) error {

	ws.timeoutsec = timeoutsec
	ws.slf = slf
	if bproxy == true {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse("http://127.0.0.1:1080")
		}

		if timeoutsec > 0 {
			tosec := timeoutsec * time.Second
			ws.WsDailer = &websocket.Dialer{Proxy: proxy, HandshakeTimeout: tosec}
		} else {
			ws.WsDailer = &websocket.Dialer{Proxy: proxy}
		}
	} else {
		if timeoutsec > 0 {
			tosec := timeoutsec * time.Second
			ws.WsDailer = &websocket.Dialer{HandshakeTimeout: tosec}
		} else {
			ws.WsDailer = &websocket.Dialer{}
		}
	}

	ws.url = strurl
	ws.bwritemsg = make(chan []byte, 1000)

	return nil
}

func (ws *WebsocketClient) OnRun() error {
	for {
		if ws.state == 0 {
			time.Sleep(1 * time.Second)
			ws.StartConnect()
		} else {
			ws.conn.SetReadDeadline(time.Now().Add(ws.timeoutsec * time.Second))
			_, message, err := ws.conn.ReadMessage()

			if err != nil {
				log.Printf("到服务器的连接断开 %+v\n", err)
				ws.conn.Close()
				ws.state = 0
				ws.slf.OnDisconnect()
				continue
			}

			ws.slf.OnReadMessage(message)
		}
	}

	return nil
}

func (ws *WebsocketClient) StartConnect() error {

	var err error
	ws.conn, _, err = ws.WsDailer.Dial(ws.url, nil)
	fmt.Printf("connecting %s, %+v\n", ws.url, err)
	if err != nil {
		return err
	}

	ws.state = 1
	ws.slf.OnConnected()

	return nil
}

func (ws *WebsocketClient) Start() error {
	ws.state = 0
	go ws.OnRun()
	go ws.writeMsg()
	return nil
}

//触发
func (ws *WebsocketClient) writeMsg() error {
	timerC := time.NewTicker(time.Second * 5).C
	for {
		if ws.state == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		select {
		case <-timerC:
			if ws.state == 1 {
				ws.WriteMessage([]byte(`ping`))
			}
		case msg := <-ws.bwritemsg:
			if ws.state == 1 {
				ws.conn.SetWriteDeadline(time.Now().Add(ws.timeoutsec * time.Second))
				err := ws.conn.WriteMessage(websocket.TextMessage, msg)

				if err != nil {
					fmt.Print(err)
					ws.state = 0
					ws.conn.Close()
					ws.slf.OnDisconnect()
				}
			}
		}
	}

	return nil
}

func (ws *WebsocketClient) WriteMessage(msg []byte) error {
	ws.bwritemsg <- msg
	return nil
}

func (ws *WebsocketClient) OnDisconnect() error {

	return nil
}

func (ws *WebsocketClient) OnConnected() error {

	return nil
}

//触发
func (ws *WebsocketClient) OnReadMessage(msg []byte) error {

	return nil
}
