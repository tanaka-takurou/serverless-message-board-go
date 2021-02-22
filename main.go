package main

import (
	"io"
	"os"
	"log"
	"sort"
	"bytes"
	"embed"
	"context"
	"strconv"
	"html/template"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type PageData struct {
	Title string
	ApiPath string
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

type Response events.APIGatewayProxyResponse

//go:embed templates
var templateFS embed.FS

var dynamodbClient *dynamodb.Client

const layout string = "2006-01-02 15:04"
const title  string = "Sample Message Room"

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
	dat.ApiPath = os.Getenv("API_PATH")
	if err != nil {
		log.Print(err)
		panic(err)
	}
	if len(s_room_id) > 0 {
		room_id, err = strconv.Atoi(s_room_id)
		if err == nil {
			dat.Title = getRoomSubject(ctx, os.Getenv("ROOM_TABLE_NAME"), room_id)
			dat.RoomId = room_id
			dat.MessagePage = 1
			dat.RoomPage = 0
			threshold, _ := strconv.Atoi(os.Getenv("THRESHOLD"))
			dat.MessageList, err = getMessageList(ctx, os.Getenv("MESSAGE_TABLE_NAME"), room_id, threshold)
			dat.RoomList = []RoomData{}
			sort.Slice(dat.MessageList, func(i, j int) bool { return dat.MessageList[i].Created < dat.MessageList[j].Created })
		}
		templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/message.html"))
	} else {
		dat.Title = title
		dat.RoomId = 0
		dat.MessagePage = 0
		dat.RoomPage = 1
		dat.MessageList = []MessageData{}
		dat.RoomList, err = getRoomList(ctx, os.Getenv("ROOM_TABLE_NAME"))
		sort.Slice(dat.RoomList, func(i, j int) bool { return dat.RoomList[i].Updated > dat.RoomList[j].Updated })
		templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/index.html", "templates/view.html", "templates/header.html", "templates/footer.html", "templates/pager.html", "templates/room.html"))
	}
	if err != nil {
		log.Print(err)
		panic(err)
	}
	if e := templates.ExecuteTemplate(fw, "base", dat); e != nil {
		log.Fatal(e)
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

func get(ctx context.Context, tableName string, key map[string]types.AttributeValue, att string)(*dynamodb.GetItemOutput, error) {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: key,
		AttributesToGet: []string{att},
		ConsistentRead: aws.Bool(true),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}
	res, err := dynamodbClient.GetItem(ctx, input)
	return res, err
}

func scan(ctx context.Context, tableName string, filt expression.ConditionBuilder, proj expression.ProjectionBuilder)(*dynamodb.ScanOutput, error)  {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
	if err != nil {
		return nil, err
	}
	input := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(tableName),
	}
	res, err := dynamodbClient.Scan(ctx, input)
	return res, err
}

func getMessageList(ctx context.Context, tableName string, room_id int, threshold int)([]MessageData, error)  {
	var messageList []MessageData
	filt := expression.Equal(expression.Name("room_id"), expression.Value(room_id))
	proj := expression.NamesList(expression.Name("message_id"), expression.Name("room_id"), expression.Name("icon_id"), expression.Name("status"), expression.Name("user"), expression.Name("message"), expression.Name("created"))
	result, err := scan(ctx, tableName, filt, proj)
	if err != nil {
		return nil, err
	}
	for _, i := range result.Items {
		item := MessageData{}
		err = attributevalue.UnmarshalMap(i, &item)
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

func getRoomList(ctx context.Context, tableName string)([]RoomData, error)  {
	var roomList []RoomData
	filt := expression.Equal(expression.Name("status"), expression.Value(0))
	proj := expression.NamesList(expression.Name("room_id"), expression.Name("status"), expression.Name("messages"), expression.Name("subject"), expression.Name("last_message"), expression.Name("last_user"), expression.Name("updated"))
	result, err := scan(ctx, tableName, filt, proj)
	if err != nil {
		return nil, err
	}
	for _, i := range result.Items {
		item := RoomData{}
		err = attributevalue.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}
		roomList = append(roomList, item)
	}
	return roomList, nil
}

func getRoomSubject(ctx context.Context, tableName string, roomId int) string {
	item := struct {Room_Id int `json:"room_id"`}{roomId}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return ""
	}
	res, err := get(ctx, tableName, av, "subject")
	if err == nil && res.Item != nil{
		result := struct {Subject string `json:"subject"`}{""}
		err = attributevalue.UnmarshalMap(res.Item, &result)
		if err != nil {
			return ""
		}
		return result.Subject
	}
	return ""
}

func getConfig(ctx context.Context) aws.Config {
	var err error
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("REGION")))
	if err != nil {
		log.Print(err)
	}
	return cfg
}

func main() {
	lambda.Start(HandleRequest)
}
