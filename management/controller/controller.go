package controller

import (
	"log"
	"sort"
	"io/ioutil"
	"strconv"
	"net/http"
	"html/template"
	"encoding/json"
)

type PageData struct {
	Title string
	RoomId int
	MessagePage int
	RoomPage int
	PageList []int
	MessageList []MessageData
	RoomList []RoomData
	TokenList []TokenData
}

type RoomData struct {
	Room_Id      int    `json:"room_id"`
	Status       int    `json:"status"`
	Messages     int    `json:"messages"`
	Subject      string `json:"subject"`
	Last_Message string `json:"last_message"`
	Last_User    string `json:"last_user"`
	Updated      string `json:"updated"`
}

type MessageData struct {
	Message_Id int    `json:"message_id"`
	Room_Id    int    `json:"room_id"`
	Icon_Id    int    `json:"icon_id"`
	Status     int    `json:"status"`
	User       string `json:"user"`
	Message    string `json:"message"`
	Created    string `json:"created"`
}

type TokenData struct {
	Token     string `json:"token"`
	Created   string `json:"created"`
}

type ConstantData struct {
	Title string `json:"title"`
}

const messageTableName string = "message"
const roomTableName    string = "room"
const tokenTableName   string = "token"

func HttpHandler(w http.ResponseWriter, request *http.Request){
	var room_id int
	var s_room_id string
	var message_id int
	var dat PageData
	var err error
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Fatal(err)
	}
	d := make(map[string]string)
	json.Unmarshal(body, &d)
	q := request.URL.Query()
	if q != nil && q["room_id"] != nil {
		s_room_id = q["room_id"][0]
	}
	funcMap := template.FuncMap{
		"safehtml": func(text string) template.HTML { return template.HTML(text) },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
	}
	templates := template.New("tmp")
	if v, ok := d["action"]; ok {
		switch v {
		case "createroom" :
			log.Print("Create Room.")
			if v, ok := d["subject"]; ok {
				err = putRoom(v)
			}
		case "addmessage" :
			log.Print("Add Message.")
			room_id, err = strconv.Atoi(d["room_id"])
			icon_id, _ := strconv.Atoi(d["icon"])
			if err == nil {
				err = putMessage(room_id, d["user"], d["message"], icon_id)
			}
		case "updateroom" :
			log.Print("Update Room.")
			room_id, err = strconv.Atoi(d["room_id"])
			if err == nil {
				status_, err_ := strconv.Atoi(d["status"])
				if err_ == nil {
					err = updateRoomStatus(room_id, status_, "status")
				}
			}
		case "updatemessage" :
			log.Print("Update Message.")
			if _, ok := d["message_id"]; ok {
				message_id, err = strconv.Atoi(d["message_id"])
				if err == nil {
					status_, err_ := strconv.Atoi(d["status"])
					if err_ == nil {
						err = updateMessage(message_id, status_, "status")
					}
				}
			}
		case "deletetoken" :
			log.Print("Delete Token.")
			if v, ok := d["token"]; ok {
				err = deleteToken(v)
			}
		}
	}
	if err != nil {
		log.Print(err)
		panic(err)
	}
	if len(s_room_id) > 0 {
		room_id, err = strconv.Atoi(s_room_id)
		if err == nil {
			dat.RoomId = room_id
			dat.MessagePage = 1
			dat.RoomPage = 0
			dat.MessageList, err = getMessageList(room_id)
			dat.RoomList = []RoomData{}
			dat.TokenList = []TokenData{}
			sort.Slice(dat.MessageList, func(i, j int) bool { return dat.MessageList[i].Created < dat.MessageList[j].Created })
		}
		templates = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/message.html"))
	} else {
		dat.RoomId = 0
		dat.MessagePage = 0
		dat.RoomPage = 1
		dat.MessageList = []MessageData{}
		dat.RoomList, err = getRoomList()
		dat.TokenList, _ = getTokenList()
		sort.Slice(dat.RoomList, func(i, j int) bool { return dat.RoomList[i].Updated < dat.RoomList[j].Updated })
		templates = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/room.html", "templates/token.html"))
	}
	if err != nil {
		log.Print(err)
		panic(err)
	}
	dat.Title = "Sample Management Page"
	err = templates.ExecuteTemplate(w, "base", dat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
