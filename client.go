package main

import (
	"github.com/gorilla/websocket"
	"time"
)

// clientはチャットを行っている1人のユーザーを表す
type client struct {
	// socketはこのclientのためのWebScoket
	socket *websocket.Conn
	// sendはメッセージが送られるチャネル
	send chan *message
	// roomはこのクライアントが参加しているチャットルーム
	room *room
	// userDataはユーザーに関する情報を保持する
	userData map[string]interface{}
}

func (c *client) read() {
	for {
		var msg *message
		if err := c.socket.ReadJSON(&msg); err == nil {
			msg.When = time.Now()
			msg.Name = c.userData["name"].(string)
			if avatarURL, ok := c.userData["avatar_url"]; ok {
				msg.AvatarURL = avatarURL.(string)
			}
			c.room.forward <- msg
		} else {
			break
		}
	}
	c.socket.Close()
}
func (c *client) write() {
	for msg := range c.send {
		if err := c.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	c.socket.Close()
}
