package main

import (
	"io"
	"log"
	"sort"
	"bytes"
	"context"
	"strconv"
	"io/ioutil"
	"html/template"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type PageData struct {
	Title string
	Api string
	RoomId int
	MessagePage int
	RoomPage int
	PageList []int
	MessageList []MessageData
	RoomList []RoomData
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
	Api       string `json:"api"`
	Title     string `json:"title"`
	Threshold int    `json:"threshold"`
}

type Response events.APIGatewayProxyResponse

const messageTableName string = "message"
const roomTableName    string = "room"
const tokenTableName   string = "token"
const layout           string = "2006-01-02 15:04"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	templates := template.New("tmp")
	var room_id int
	var dat PageData
	var err error
	q := request.QueryStringParameters
	s_room_id := q["room_id"]
	funcMap := template.FuncMap{
		"safehtml": func(text string) template.HTML { return template.HTML(text) },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
	}
	buf := new(bytes.Buffer)
	fw := io.Writer(buf)
	jsonString, _ := ioutil.ReadFile("constant/constant.json")
	constant := new(ConstantData)
	json.Unmarshal(jsonString, constant)
	dat.Api = constant.Api
	if err != nil {
		log.Print(err)
		panic(err)
	}
	if len(s_room_id) > 0 {
		room_id, err = strconv.Atoi(s_room_id)
		if err == nil {
			dat.Title = getRoomSubject(room_id)
			dat.RoomId = room_id
			dat.MessagePage = 1
			dat.RoomPage = 0
			dat.MessageList, err = getMessageList(room_id, constant.Threshold)
			dat.RoomList = []RoomData{}
			sort.Slice(dat.MessageList, func(i, j int) bool { return dat.MessageList[i].Created < dat.MessageList[j].Created })
		}
		templates = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/message.html"))
	} else {
		dat.Title = constant.Title
		dat.RoomId = 0
		dat.MessagePage = 0
		dat.RoomPage = 1
		dat.MessageList = []MessageData{}
		dat.RoomList, err = getRoomList()
		sort.Slice(dat.RoomList, func(i, j int) bool { return dat.RoomList[i].Updated > dat.RoomList[j].Updated })
		templates = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/room.html"))
	}
	if err != nil {
		log.Print(err)
		panic(err)
	}
	if e := templates.ExecuteTemplate(fw, "base", dat); e != nil {
		log.Fatal(e)
	} else {
		log.Print(request.RequestContext.Identity.SourceIP)
	}
	res := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(buf.Bytes()),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}
	return res, nil
}

func get(tableName string, key map[string]*dynamodb.AttributeValue, att string)(*dynamodb.GetItemOutput, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: key,
		AttributesToGet: []*string{
			aws.String(att),
		},
		ConsistentRead: aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	return svc.GetItem(input)
}

func scan(tableName string, filt expression.ConditionBuilder)(*dynamodb.ScanOutput, error)  {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	expr, err := expression.NewBuilder().WithFilter(filt).Build()
	if err != nil {
		return nil, err
	}
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(tableName),
	}
	return svc.Scan(params)
}

func getMessageList(room_id int, threshold int)([]MessageData, error)  {
	var messageList []MessageData
	result, err := scan(messageTableName, expression.Name("room_id").Equal(expression.Value(room_id)))
	if err != nil {
		return nil, err
	}
	for _, i := range result.Items {
		item := MessageData{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}
		if item.Status >= threshold {
			continue
		}
		messageList = append(messageList, item)
	}
	return messageList, nil
}

func getRoomList()([]RoomData, error)  {
	var roomList []RoomData
	result, err := scan(roomTableName, expression.Equal(expression.Name("status"), expression.Value(0)))
	if err != nil {
		return nil, err
	}
	for _, i := range result.Items {
		item := RoomData{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}
		roomList = append(roomList, item)
	}
	return roomList, nil
}

func getRoomSubject(roomId int) string {
	item := struct {Room_Id int `json:"room_id"`}{roomId}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return ""
	}
	res, err := get(roomTableName, av, "subject")
	if err == nil && res.Item != nil{
		result := struct {Subject string `json:"subject"`}{""}
		err = dynamodbattribute.UnmarshalMap(res.Item, &result)
		if err != nil {
			return ""
		}
		return result.Subject
	}
	return ""
}

func main() {
	lambda.Start(HandleRequest)
}
