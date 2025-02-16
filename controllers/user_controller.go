package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"main.go/models"
	"main.go/services"
)

type IUserController interface {
	HandleWebSocket(ctx *gin.Context)
}

type UserController struct {
	service       services.IUserService
	memoryservice services.IUserMemoryService
}

// メモリサービス
type MemoryService struct {
	// memory   map[string]*models.Client
	memory   sync.Map
	duration time.Duration // 一定時間後に削除する時間
}

// メモリの情報を更新する関数
func (ms *MemoryService) UpdateMemory(clientID string, client *models.Client, c *UserController) {
	ms.memory.Store(clientID, client)

	// クライアント情報とともに時刻を記録
	key := client.Key
	// ゴルーチンで削除処理を開始
	go ms.scheduleDeletion(clientID, c, &key)
}

// メモリからクライアント情報を削除する関数
func (ms *MemoryService) scheduleDeletion(clientID string, c *UserController, key *string) {
	// 一定時間後に削除
	time.Sleep(ms.duration)
	value, ok := ms.memory.Load(clientID)
	if !ok {
		return
	}
	client := value.(*models.Client)
	if client.Key == *key {
		c.memoryservice.DeleteMemory(clientID)
		ms.memory.Delete(clientID)
	}
	// clients, err := c.memoryservice.FindMemories(clientID)
	// if err != nil {
	// 	return
	// }
	// for _, v := range *clients {
	// 	fmt.Println(v.Key, *key, "scheduleDeletion()", v.Key == *key)
	// 	if v.Key == *key {
	// 		c.memoryservice.DeleteMemory(clientID)
	// 	}
	// }
	// クライアント情報が存在すれば削除
	// delete(ms.memory, clientID)
	// fmt.Println("Client removed after timeout:", clientID)
}

func NewUserController(service services.IUserService, memoryservice services.IUserMemoryService) IUserController {
	return &UserController{
		service:       service,
		memoryservice: memoryservice,
	}
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients = make(map[string]*models.Client) // クライアントリスト
	rooms   = make(map[int]*models.Room, 0)
	mu      sync.Mutex // 排他制御
)

// WebSocket 接続を処理するハンドラ
func (c *UserController) HandleWebSocket(ctx *gin.Context) {

	// HTTP 接続を WebSocket にアップグレード
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		fmt.Println("WebSocket 接続エラー:", err)
		return
	}
	defer conn.Close()

	fmt.Println("クライアントが接続しました")
	clientID := uuid.New().String()
	// 新しいクライアントを登録
	mu.Lock()
	clients[clientID] = &models.Client{ID: clientID, Conn: conn, RoomID: -1, Key: "0"}
	err = c.memoryservice.CreateMemory(clients[clientID])
	if err != nil {
		mu.Unlock()
		return
	}
	// fmt.Println(clients[clientID])
	// fmt.Println(clients)
	mu.Unlock()
	clientDisconnectTimer := time.NewTimer(3 * time.Second)
	numControll()
	defer func() {
		mu.Lock()
		// fmt.Println("削除を実行")
		// if room, ok := rooms[clients[clientID].RoomID]; ok {
		// 	delete(rooms, room.RoomID)
		// }
		memoryService := MemoryService{
			// memory:   make(map[string]*models.Client),
			duration: 5 * time.Second, // 3秒後に削除
		}
		client, ok := clients[clientID]
		if !ok {
			// クライアントが見つからなかった場合、適切なエラーハンドリング
			fmt.Println("Client not found:", clientID)
			ok := c.memoryservice.DeleteMemory(clientID)
			if ok != nil {
				fmt.Println(ok)
			}
		} else {
			go memoryService.UpdateMemory(clientID, client, c)
			delete(clients, clientID)
		}
		// if room, ok := rooms[client.RoomID]; ok {
		// 	// roomが見つかった場合の処理
		// 	//対戦相手にコネクションが失われたと送信
		// 	for i := range room.Players {
		// 		if room.Players[i].ID == clientID {
		// 			room.Players = append(room.Players[:i], room.Players[i+1:]...)
		// 		}
		// 	}
		// } else {
		// 	// roomが見つからなかった場合の処理
		// 	// fmt.Println("Error: Room not found,", client.RoomID)
		// }
		mu.Unlock()
		numControll()
		fmt.Println("切断完了:", clientID)
	}()
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("接続が切断されました:", err)
			// numControll()
			clientDisconnectTimer.Stop()
			break
		}
		fmt.Printf("受信メッセージ: %s\n", msg)
		var receivedMsg models.Message
		//shouldbindingするのか
		if err := json.Unmarshal(msg, &receivedMsg); err != nil {
			fmt.Println("JSON のパースに失敗:", err)
			continue
		}
		if receivedMsg.Type == "join" {
			mu.Lock()
			roomnum := rand.Intn(1000000)
			for {
				if rooms[roomnum] != nil {
					roomnum = rand.Intn(1000000)
				} else {
					break
				}
			}

			key := uuid.New().String()
			clients[clientID] = &models.Client{
				ID:     clientID,
				Conn:   conn,
				RoomID: roomnum,
				Key:    key,
			}
			c.memoryservice.UpdateMemory(clientID, clients[clientID])
			message := fmt.Sprintf(`{"type":"roomnum", "message": %d}`, roomnum)
			conn.WriteMessage(messageType, []byte(message))
			mu.Unlock()
			//この時点でルームを作成し、ホストにしてもよい
		} else if receivedMsg.Type == "join for" {
			room := rooms[receivedMsg.RoomID]
			if receivedMsg.RoomID <= 0 || (room != nil && room.Start) {
				//nullもしたほうがいいかも
				message := `{"type":"error", "message": "有効な数値を入力してください"}`
				conn.WriteMessage(messageType, []byte(message))
			} else {
				model, err := c.memoryservice.FindMessageMemory(receivedMsg)
				// fmt.Println(model)
				if err != nil {
					if err.Error() == "room not found" {
						clients[clientID] = &models.Client{ID: clientID, Conn: conn, RoomID: -1, Key: "0"}
						c.memoryservice.UpdateMemory(clientID, clients[clientID])
						message := `{"type":"error", "message": "ルームが見つかりませんでした"}`
						conn.WriteMessage(messageType, []byte(message))
						return
					} else {
						fmt.Println("join for error")
						return
					}
				} else {
					mu.Lock()
					//model.IDをhostとし、clientIDをメンバーとする
					roomnum := model.RoomID
					// c.memoryservice.UpdateMemory(model.ID, clients[model.ID])
					clients[clientID] = &models.Client{ID: clientID, Conn: conn, RoomID: roomnum, Key: model.Key}
					err := c.memoryservice.UpdateMemory(clientID, clients[clientID])
					if err != nil {
						fmt.Println("user not found", "join for")
						mu.Unlock()
						return
					}
					// BattleReady(roomnum, model.ID, clientID)
					if model.Conn == nil || conn == nil {
						mu.Unlock()
						return
					} else {
						message := fmt.Sprintf(`{"type":"ready","message":"%s","RoomNum":"%d"}`, model.ID, roomnum)
						model.Conn.WriteMessage(messageType, []byte(message))
						message = fmt.Sprintf(`{"type":"ready","message":"%s","RoomNum":"%d"}`, clientID, roomnum)
						conn.WriteMessage(messageType, []byte(message))
						BattleReady(roomnum)
						// delete(clients, model.ID)
						// delete(clients, clientID)
						mu.Unlock()
					}
				}
			}
		} else if receivedMsg.Type == "reconnect" {
			// 再接続時に、記録されている情報を基に復元
			// fmt.Println("reconnect")
			// room := rooms[receivedMsg.RoomID]
			mu.Lock()
			// exists := 0
			// for i := range room.Players {
			// 	if room.Players[i].ID == receivedMsg.Message {
			// 		exists = i
			// 	}
			// }
			// mu.Unlock()
			// if exists != 0 {
			// 	mu.Lock()
			// 	clients[clientID] = room.Players[exists]
			// 	err = c.memoryservice.DeleteMemory(receivedMsg.Message)
			// 	if err != nil {
			// 		continue
			// 	}
			// 	err = c.memoryservice.CreateMemory(room.Players[exists])
			// 	// fmt.Println(err, "err")
			// 	if err != nil {
			// 		mu.Unlock()
			// 		return
			// 	}
			// 	mu.Unlock()
			// 	fmt.Println("再々接続成功")
			// } else {
			// fmt.Println(receivedMsg.RoomID)
			_, err := c.memoryservice.FindMemory(receivedMsg.Message)
			if err != nil {
				// メモリ内に情報がない場合、エラーメッセージ
				message := `{"type":"error", "message": "再接続情報が見つかりません"}`
				conn.WriteMessage(messageType, []byte(message))
				mu.Unlock()
				return
			} else {
				// mu.Lock()
				key := uuid.New().String()
				reClient := &models.Client{
					ID:     clientID,
					Conn:   conn,
					RoomID: receivedMsg.RoomID,
					Key:    key,
				}
				// fmt.Println(reClient, "recreate")
				err = c.memoryservice.DeleteMemory(receivedMsg.Message)
				if err != nil {
					continue
				}
				err = c.memoryservice.CreateMemory(reClient)
				// fmt.Println(err, "err")
				if err != nil {
					mu.Unlock()
					return
				}
				// 以前の状態を復元し、再接続
				clients[clientID] = reClient
				fmt.Println(clients)
				fmt.Println("再接続成功")
				message := fmt.Sprintf(`{"type":"reset","message":"%s","RoomNum":"%d"}`, clientID, receivedMsg.RoomID)
				conn.WriteMessage(messageType, []byte(message))
				room := rooms[reClient.RoomID]
				//二度以上の接続かチェック（reconnectのたびに追加していないか）
				exists := false
				if len(room.Audience) != 0 {
					for i := range room.Audience {
						if room.Audience[i].ID == receivedMsg.Message {
							room.Audience[i] = reClient
							exists = true
						}
					}
				}
				if len(room.Players) == 0 {
					room.Players = append(room.Players, reClient)
				} else {
					for i := range room.Players {
						if room.Players[i].ID == receivedMsg.Message {
							room.Players[i] = reClient
							exists = true
						}
					}
					if !exists {
						room.Players = append(room.Players, reClient)
					}
				}
				// fmt.Println(room.Players, "room.Players")
				// delete(clients, clientID)
				// c.memoryservice.DeleteMemory(clientID)
				// clientID = receivedMsg.Message
				if !room.Start && len(room.Players) >= 2 {
					go Battle(room.Players[0].ID, room.Players[1].ID)
					room.Start = true
					fmt.Println("start")
				}
				mu.Unlock()
				// }
				//一回のみ実行したい
				numControll()
				message = `{"type":"reconnect", "message": "再接続成功"}`
				conn.WriteMessage(messageType, []byte(message))
			}
		} else if receivedMsg.Type == "battle" {
			// fmt.Println("battle start")
			// fmt.Println(clientID)
			// Battle(clientID)

			//一回のみ実行したい
			return
		} else if receivedMsg.Type == "update" {
			// fmt.Println("update")
			mu.Lock()
			if receivedMsg.Message == "" {
				fmt.Println("間違ったコネクションです")
				mu.Unlock()
				return
			}
			room, ok := rooms[receivedMsg.RoomID]
			if !ok {
				fmt.Println(ok, "roomnum error")
				mu.Unlock()
				return
			}
			if room == nil {
				mu.Unlock()
				return
			}
			mu.Unlock()
			if room.Start {
				mu.Lock()
				fmt.Println(room.Players, len(room.Players))
				if len(room.Players) == 1 {
					if room.Players[0] != nil && clientID == room.Players[0].ID {
						if receivedMsg.GameState.Player1X < 385 {
							room.GameState.Player1X = receivedMsg.GameState.Player1X - 25
						}
						room.GameState.Player1Y = receivedMsg.GameState.Player1Y - 35
						// fmt.Println(room.GameState.Player1Y)
					}
				} else if len(room.Players) == 2 {
					if room.Players[0] != nil && clientID == room.Players[0].ID {
						if receivedMsg.GameState.Player1X < 385 {
							room.GameState.Player1X = receivedMsg.GameState.Player1X - 25
						}
						room.GameState.Player1Y = receivedMsg.GameState.Player1Y - 35
						// fmt.Println(room.GameState.Player1Y)
					}
					if room.Players[1] != nil && clientID == room.Players[1].ID {
						if receivedMsg.GameState.Player2X > 385 {
							room.GameState.Player2X = receivedMsg.GameState.Player2X - 25
						}
						room.GameState.Player2Y = receivedMsg.GameState.Player2Y - 35
						// fmt.Println(room.GameState.Player2Y)
					}
				}
				mu.Unlock()
				updateGameState(room)
			}
		} else if receivedMsg.Type == "audience" {
			mu.Lock()
			if receivedMsg.RoomID < 0 {
				mu.Unlock()
				return
			}
			room := rooms[receivedMsg.RoomID]
			if room == nil {
				message := `{"type":"error", "message": "ルームが存在しません"}`
				conn.WriteMessage(messageType, []byte(message))
				mu.Unlock()
				return
			}
			audience := &models.Client{
				ID:     clientID,
				Conn:   conn,
				RoomID: receivedMsg.RoomID,
			}
			if len(room.Audience) == 0 {
				room.Audience = append(room.Audience, audience)
			} else {
				exists := false
				for i := range room.Audience {
					if room.Audience[i].ID == audience.ID {
						room.Audience[i] = audience
						exists = true
					}
				}
				if !exists {
					room.Audience = append(room.Audience, audience)
				}
			}
			message := fmt.Sprintf(`{"type":"ready","message":"%s","RoomNum":"%d"}`, clientID, audience.RoomID)
			mu.Unlock()
			conn.WriteMessage(messageType, []byte(message))
		} else {
			fmt.Println("client error")
			return
		}
	}
}

//pingの確認
// WebSocketのセットアップ時にPing/Pongを処理
// conn.SetPingHandler(func(appData string) error {
//     return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
// })
//タイマー式
// func removeInactiveClients() {
//     ticker := time.NewTicker(10 * time.Second)
//     defer ticker.Stop()

//     for range ticker.C {
//         mu.Lock()
//         for id, client := range clients {
//             if time.Since(client.LastActive) > 30*time.Second {
//                 delete(clients, id)
//                 fmt.Println("タイムアウトで削除:", id)
//             }
//         }
//         mu.Unlock()
//     }
// }

func BattleReady(roomnum int) {
	newRoom := &models.Room{
		RoomID: roomnum,
		// Players: []*models.Client{},
		Players: make([]*models.Client, 0),
		GameState: &models.GameState{
			PuckX:      385.0,
			PuckY:      185.0,
			PuckSpeedX: (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1), // 絶対値2~5 のランダム速度
			PuckSpeedY: (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1),
			Player1X:   50,
			Player1Y:   185,
			Player2X:   700,
			Player2Y:   185,
			Score1:     0,
			Score2:     0,
			Time:       60 * 120, //2min 120 * 60
		},
		Audience: make([]*models.Client, 0),
		Start:    false,
	}
	rooms[roomnum] = newRoom
}

func Battle(clientID string, clientID2 string) {
	time.Sleep(5 * time.Second)
	mu.Lock()
	player, ok := clients[clientID]
	if !ok {
		fmt.Println("player is nil in Battle")
		mu.Unlock()
		return
	}
	player2, ok := clients[clientID2]
	if !ok {
		fmt.Println("player is nil in Battle")
		mu.Unlock()
		return
	}
	conn := player.Conn
	if conn == nil {
		fmt.Println("player.Conn is nil in Battle")
		mu.Unlock()
		return
	}
	conn2 := player2.Conn
	if conn2 == nil {
		fmt.Println("player.Conn is nil in Battle")
		mu.Unlock()
		return
	}

	roomnum := player.RoomID
	// fmt.Println(roomnum)
	room, ok := rooms[roomnum]
	if !ok {
		fmt.Println(room, ok, "ルームが見つかりません")
		message := `{"type":"error", "message": "ルームが見つかりません"}`
		conn.WriteMessage(websocket.TextMessage, []byte(message))
		mu.Unlock()
		return
	}
	roomnum = player2.RoomID
	// fmt.Println(roomnum)
	room, ok = rooms[roomnum]
	if !ok {
		fmt.Println(room, ok, "ルームが見つかりません")
		message := `{"type":"error", "message": "ルームが見つかりません"}`
		conn.WriteMessage(websocket.TextMessage, []byte(message))
		mu.Unlock()
		return
	}
	mu.Unlock()
	go gameLoop(room)
}

func gameLoop(room *models.Room) {
	if room == nil {
		return
	}
	ticker := time.NewTicker(16 * time.Millisecond) // 約60FPS
	defer ticker.Stop()

	for {
		<-ticker.C
		mu.Lock()
		room.GameState.Time--
		mu.Unlock()
		if room.GameState.Time <= 0 {
			fmt.Println("ゲーム終了:", room.RoomID)
			// for i := range room.Players {
			// 	delete(clients, room.Players[i].ID)
			// }
			message := `{"type":"end", "message": "対戦終了"}`
			room.Players[0].Conn.WriteMessage(websocket.TextMessage, []byte(message))
			room.Players[1].Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if room.Audience != nil {
				for i := range room.Audience {
					room.Audience[i].Conn.WriteMessage(websocket.TextMessage, []byte(message))
				}
			}
			rooms[room.RoomID] = nil
			break
		}
		temporary(room)
		updateGameState(room)
		broadcastGameState(room)
	}
}

func temporary(room *models.Room) {
	// for i := range room.Players {
	// 	if room.Players[i].Conn == nil {
	// 		for i := range rooms {
	// 			if room.Players[i].Conn != nil {
	// 				message := `{"type":"error", "message": "対戦相手との接続が切れました"}`
	// 				room.Players[i].Conn.WriteMessage(websocket.TextMessage, []byte(message))
	// 			}
	// 		}
	// 	}
	// }
	// パックの移動処理
	mu.Lock()
	room.GameState.PuckX += room.GameState.PuckSpeedX
	room.GameState.PuckY += room.GameState.PuckSpeedY

	// 壁の反射処理
	if room.GameState.PuckY <= 0 || room.GameState.PuckY >= 370 {
		room.GameState.PuckSpeedY *= -1
	}
	mu.Unlock()
}

func updateGameState(room *models.Room) {
	mu.Lock()

	// パックとパドルの反射処理
	d := math.Sqrt((room.GameState.Player1X-room.GameState.PuckX)*(room.GameState.Player1X-room.GameState.PuckX) + (room.GameState.Player1Y-room.GameState.PuckY)*(room.GameState.Player1Y-room.GameState.PuckY))
	if d < 45 { //パドルの半径とパックの半径の合計
		room.GameState.PuckSpeedX *= -1
		room.GameState.PuckSpeedY *= -1
		if room.GameState.PuckSpeedX > 0 && room.GameState.PuckSpeedX < 10 {
			room.GameState.PuckSpeedX += 1
		} else {
			room.GameState.PuckSpeedX -= 1
		}
		if room.GameState.PuckSpeedY > 0 && room.GameState.PuckSpeedY < 10 {
			room.GameState.PuckSpeedY += 1
		} else {
			room.GameState.PuckSpeedY -= 1
		}
	}
	d = math.Sqrt((room.GameState.Player2X-room.GameState.PuckX)*(room.GameState.Player2X-room.GameState.PuckX) + (room.GameState.Player2Y-room.GameState.PuckY)*(room.GameState.Player2Y-room.GameState.PuckY))
	if d < 45 {
		room.GameState.PuckSpeedX *= -1
		room.GameState.PuckSpeedY *= -1
		if room.GameState.PuckSpeedX > 0 && room.GameState.PuckSpeedX < 10 {
			room.GameState.PuckSpeedX += 1
		} else {
			room.GameState.PuckSpeedX -= 1
		}
		if room.GameState.PuckSpeedY > 0 && room.GameState.PuckSpeedY < 10 {
			room.GameState.PuckSpeedY += 1
		} else {
			room.GameState.PuckSpeedY -= 1
		}
	}

	// スコア処理
	if room.GameState.PuckX <= 0 {
		room.GameState.Score2++
		room.GameState.PuckX, room.GameState.PuckY = 385, 185
		room.GameState.PuckSpeedX = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
		room.GameState.PuckSpeedY = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
	}
	if room.GameState.PuckX >= 750 {
		room.GameState.Score1++
		room.GameState.PuckX, room.GameState.PuckY = 385, 185
		room.GameState.PuckSpeedX = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
		room.GameState.PuckSpeedY = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
	}
	mu.Unlock()
}

// クライアントにゲーム状態を送信
func broadcastGameState(room *models.Room) {
	mu.Lock()

	gameStateJSON, err := json.Marshal(room.GameState)
	if err != nil {
		fmt.Println("GameState JSON変換エラー:", err)
		mu.Unlock()
		return
	}
	message := fmt.Sprintf(`{"type":"battle","message":%s}`, gameStateJSON)
	// fmt.Println(message)
	for _, client := range room.Players {
		err = client.Conn.WriteJSON([]byte(message))
		if err != nil {
			// fmt.Println("送信エラー:", err)
		}
	}
	for _, client := range room.Audience {
		err = client.Conn.WriteJSON([]byte(message))
		if err != nil {
			// fmt.Println("送信エラー:", err)
		}
	}
	mu.Unlock()
}

func numControll() {
	mu.Lock()
	usernum := len(clients)
	// fmt.Println(clients)
	for client := range clients {
		message := fmt.Sprintf(`{"type":"usernum","usernum": %d}`, usernum)
		err := clients[client].Conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Println("メッセージ送信エラー:", err)
			continue
		}
		room := rooms[clients[client].RoomID]
		if room == nil {
			continue
		} else {
			message = fmt.Sprintf(`{"type":"audiences","message":%d}`, len(room.Audience))
			err = clients[client].Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				fmt.Println("メッセージ送信エラー:", err)
				continue
				// clients[client].Conn.Close()
				// delete(clients, client)
			}
		}
	}
	mu.Unlock()
	// fmt.Println(userNum)
}
