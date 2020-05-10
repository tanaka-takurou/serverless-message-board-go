package main

import (
	"log"
	"time"
	"context"
	"strconv"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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

type TokenResponse struct {
	Token     string `json:"token"`
}

type Response events.APIGatewayProxyResponse

const messageTableName string = "sample_message"
const roomTableName string = "sample_room"
const tokenTableName string = "sample_token"
const layout = "2006-01-02 15:04"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var jsonBytes []byte
	var room_id int
	var message_id int
	var err error
	hash, err := bcrypt.GenerateFromPassword([]byte("salt1"), bcrypt.DefaultCost)
	d := make(map[string]string)
	json.Unmarshal([]byte(request.Body), &d)
	if err == nil {
		err = putToken(string(hash))
		if err == nil {
			jsonBytes, err = json.Marshal(TokenResponse{Token:string(hash)})
		}
	}
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
		case "updatemessage" :
			log.Print("Update Message.")
			if _, ok := d["message_id"]; ok {
				message_id, err = strconv.Atoi(d["message_id"])
				if err == nil {
					err = updateMessage(message_id, 1, "status")
				}
			}
		}
	}
	if err != nil {
		return Response{}, err
	} else {
		log.Print(request.RequestContext.Identity.SourceIP)
	}
	return Response {
		StatusCode: 200,
		Body: string(jsonBytes),
	}, nil
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

func put(tableName string, av map[string]*dynamodb.AttributeValue) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}
	_, err := svc.PutItem(input)
	return err
}

func putToken(token string) error {
	t := time.Now()
	item := TokenData {
		Token: token,
		Created: t.Format(layout),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(tokenTableName, av)
	if err != nil {
		return err
	}
	return nil
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

func update(tableName string, an map[string]*string, av map[string]*dynamodb.AttributeValue, key map[string]*dynamodb.AttributeValue, updateExpression string) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: an,
		ExpressionAttributeValues: av,
		TableName: aws.String(tableName),
		Key: key,
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String(updateExpression),
	}

	_, err := svc.UpdateItem(input)
	return err
}

func updateRoom(room_id int, user string, message string, updated string) error {
	an := map[string]*string{
		"#u": aws.String("last_user"),
		"#m": aws.String("last_message"),
		"#d": aws.String("updated"),
		"#c": aws.String("messages"),
	}
	av := map[string]*dynamodb.AttributeValue{
		":u": {
			S: aws.String(user),
		},
		":m": {
			S: aws.String(message),
		},
		":d": {
			S: aws.String(updated),
		},
		":c": {
			N: aws.String("1"),
		},
	}
	key := map[string]*dynamodb.AttributeValue{
		"room_id": {
			N: aws.String(strconv.Itoa(room_id)),
		},
	}
	updateExpression := "set #u = :u, #m = :m, #d = :d, #c = #c + :c"

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: an,
		ExpressionAttributeValues: av,
		TableName: aws.String(roomTableName),
		Key: key,
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String(updateExpression),
	}

	_, err := svc.UpdateItem(input)
	if err != nil {
		return err
	}
	return nil
}

func getMessageCount(room_id int)(*int64, error)  {
	result, err := scan(messageTableName, expression.Name("room_id").Equal(expression.Value(room_id)))
	if err != nil {
		return nil, err
	}
	return result.ScannedCount, nil
}

func getRoomCount()(*int64, error)  {
	result, err := scan(roomTableName, expression.NotEqual(expression.Name("status"), expression.Value(-1)))
	if err != nil {
		return nil, err
	}
	return result.ScannedCount, nil
}

func putMessage(room_id int, user string, message string, icon_id int) error {
	t := time.Now()
	count, err := getMessageCount(room_id)
	if err != nil {
		return err
	}
	item := MessageData {
		Message_Id: int(*count) + 1,
		Room_Id: room_id,
		Icon_Id: icon_id,
		Status: 0,
		User: user,
		Message: message,
		Created: t.Format(layout),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(messageTableName, av)
	if err != nil {
		return err
	}
	_ = updateRoom(room_id, user, message, t.Format(layout))
	return nil
}

func putRoom(subject string) error {
	t := time.Now()
	count, err := getRoomCount()
	if err != nil {
		return err
	}
	item := RoomData {
		Room_Id: int(*count) + 1,
		Status: 0,
		Messages: 0,
		Subject: subject,
		Last_Message: "none",
		Last_User: "none",
		Updated: t.Format(layout),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(roomTableName, av)
	if err != nil {
		return err
	}
	return nil
}

func updateMessage(message_id int, value int, name string) error {
	an := map[string]*string{
		"#s": aws.String(name),
	}
	av := map[string]*dynamodb.AttributeValue{
		":new": {
			N: aws.String(strconv.Itoa(value)),
		},
	}
	key := map[string]*dynamodb.AttributeValue{
		"message_id": {
			N: aws.String(strconv.Itoa(message_id)),
		},
	}
	updateExpression := "set #s = #s + :new"
	err := update(messageTableName, an, av, key, updateExpression)
	if err != nil {
		return err
	}
	return nil
}

func checkToken(token string) bool {
	item := struct {Token string `json:"token"`}{token}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return false
	}
	res, err := get(tokenTableName, av, "token")
	if err == nil && res.Item != nil{
		return true
	}
	return false
}

func main() {
	lambda.Start(HandleRequest)
}
