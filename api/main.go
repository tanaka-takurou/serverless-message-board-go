package main

import (
	"os"
	"log"
	"time"
	"context"
	"strconv"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type RoomData struct {
	Room_Id      int    `dynamodbav:"room_id"`
	Status       int    `dynamodbav:"status"`
	Messages     int    `dynamodbav:"messages"`
	Subject      string `dynamodbav:"subject"`
	Last_Message string `dynamodbav:"last_message"`
	Last_User    string `dynamodbav:"last_user"`
	Updated      string `dynamodbav:"updated"`
}

type MessageData struct {
	Message_Id int    `dynamodbav:"message_id"`
	Room_Id    int    `dynamodbav:"room_id"`
	Icon_Id    int    `dynamodbav:"icon_id"`
	Status     int    `dynamodbav:"status"`
	User       string `dynamodbav:"user"`
	Message    string `dynamodbav:"message"`
	Created    string `dynamodbav:"created"`
}

type UpdateRoomData struct {
	Last_User    string `dynamodbav:":u"`
	Last_Message string `dynamodbav:":m"`
	Messages     int    `dynamodbav:":c"`
	Updated      string `dynamodbav:":d"`
}

type TokenData struct {
	Token     string `dynamodbav:"token"`
	Created   string `dynamodbav:"created"`
}

type TokenResponse struct {
	Token     string `json:"token"`
}

type Response events.APIGatewayProxyResponse

var dynamodbClient *dynamodb.Client

const layout string = "2006-01-02 15:04"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var jsonBytes []byte
	var room_id int
	var message_id int
	var err error
	d := make(map[string]string)
	json.Unmarshal([]byte(request.Body), &d)
	if v, ok := d["action"]; ok {
		switch v {
		case "createroom" :
			log.Print("Create Room.")
			if _, ok := d["subject"]; ok {
				if v, ok := d["token"]; ok {
					if checkToken(ctx, os.Getenv("TOKEN_TABLE_NAME"), v) {
						err = putRoom(ctx, os.Getenv("ROOM_TABLE_NAME"), d["subject"])
						deleteToken(ctx, os.Getenv("TOKEN_TABLE_NAME"), v)
					}
				}
			}
		case "addmessage" :
			log.Print("Add Message.")
			room_id, err = strconv.Atoi(d["room_id"])
			icon_id, _ := strconv.Atoi(d["icon"])
			if err == nil {
				if v, ok := d["token"]; ok {
					if checkToken(ctx, os.Getenv("TOKEN_TABLE_NAME"), v) {
						err = putMessage(ctx, os.Getenv("MESSAGE_TABLE_NAME"), os.Getenv("ROOM_TABLE_NAME"), room_id, d["user"], d["message"], icon_id)
						deleteToken(ctx, os.Getenv("TOKEN_TABLE_NAME"), v)
					}
				}
			}
		case "updatemessage" :
			log.Print("Update Message.")
			if _, ok := d["message_id"]; ok {
				message_id, err = strconv.Atoi(d["message_id"])
				if err == nil {
					err = updateMessage(ctx, os.Getenv("MESSAGE_TABLE_NAME"), message_id, 1, "status")
				}
			}
		case "puttoken" :
			hash, err := bcrypt.GenerateFromPassword([]byte("salt1"), bcrypt.DefaultCost)
			if err == nil {
				err = putToken(ctx, os.Getenv("TOKEN_TABLE_NAME"), string(hash))
				if err == nil {
					jsonBytes, err = json.Marshal(TokenResponse{Token:string(hash)})
				}
			}
		}
	}
	if err != nil {
		log.Print(err)
		return Response{}, err
	}
	responseBody := ""
	if len(jsonBytes) > 0 {
		responseBody = string(jsonBytes)
	}
	return Response {
		StatusCode: 200,
		Body: responseBody,
	}, nil
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

func put(ctx context.Context, tableName string, av map[string]types.AttributeValue) error {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}
	_, err := dynamodbClient.PutItem(ctx, input)
	return err
}

func putToken(ctx context.Context, tokenTableName string, token string) error {
	t := time.Now()
	item := TokenData {
		Token: token,
		Created: t.Format(layout),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(ctx, tokenTableName, av)
	if err != nil {
		return err
	}
	return nil
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

func update(ctx context.Context, tableName string, an map[string]string, av map[string]types.AttributeValue, key map[string]types.AttributeValue, updateExpression string) error {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: an,
		ExpressionAttributeValues: av,
		TableName: aws.String(tableName),
		Key: key,
		ReturnValues:     types.ReturnValueUpdatedNew,
		UpdateExpression: aws.String(updateExpression),
	}

	_, err := dynamodbClient.UpdateItem(ctx, input)
	return err
}

func updateRoom(ctx context.Context, roomTableName string, room_id int, user string, message string, updated string) error {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	an := map[string]string{
		"#u": "last_user",
		"#m": "last_message",
		"#d": "updated",
		"#c": "messages",
	}

	av_struct := UpdateRoomData{
		Last_User:    user,
		Last_Message: message,
		Messages:     1,
		Updated:      updated,
	}
	av, err := attributevalue.MarshalMap(av_struct)
	if err != nil {
		return err
	}

	key_struct := struct {Room int `dynamodbav:"room_id"`}{room_id}
	key, err := attributevalue.MarshalMap(key_struct)
	if err != nil {
		return err
	}
	updateExpression := "set #u = :u, #m = :m, #d = :d, #c = #c + :c"

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: an,
		ExpressionAttributeValues: av,
		TableName: aws.String(roomTableName),
		Key: key,
		ReturnValues:     types.ReturnValueUpdatedNew,
		UpdateExpression: aws.String(updateExpression),
	}

	_, err = dynamodbClient.UpdateItem(ctx, input)
	return err
}

func getMessageCount(ctx context.Context, messageTableName string, room_id int)(int32, error)  {
	filt := expression.Equal(expression.Name("room_id"), expression.Value(room_id))
	proj := expression.NamesList(expression.Name("message_id"), expression.Name("room_id"), expression.Name("icon_id"), expression.Name("status"), expression.Name("user"), expression.Name("message"), expression.Name("created"))
	result, err := scan(ctx, messageTableName, filt, proj)
	if err != nil {
		return int32(0), err
	}
	return result.ScannedCount, nil
}

func getRoomCount(ctx context.Context, roomTableName string)(int32, error)  {
	filt := expression.NotEqual(expression.Name("status"), expression.Value(-1))
	proj := expression.NamesList(expression.Name("room_id"), expression.Name("status"), expression.Name("messages"), expression.Name("subject"), expression.Name("last_message"), expression.Name("last_user"), expression.Name("updated"))
	result, err := scan(ctx, roomTableName, filt, proj)
	if err != nil {
		return int32(0), err
	}
	return result.ScannedCount, nil
}

func putMessage(ctx context.Context, messageTableName string, roomTableName string, room_id int, user string, message string, icon_id int) error {
	t := time.Now()
	count, err := getMessageCount(ctx, messageTableName, room_id)
	if err != nil {
		return err
	}
	item := MessageData {
		Message_Id: int(count) + 1,
		Room_Id: room_id,
		Icon_Id: icon_id,
		Status: 0,
		User: user,
		Message: message,
		Created: t.Format(layout),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(ctx, messageTableName, av)
	if err != nil {
		return err
	}
	_ = updateRoom(ctx, roomTableName, room_id, user, message, t.Format(layout))
	return nil
}

func putRoom(ctx context.Context, roomTableName string, subject string) error {
	t := time.Now()
	count, err := getRoomCount(ctx, roomTableName)
	if err != nil {
		return err
	}
	item := RoomData {
		Room_Id: int(count) + 1,
		Status: 0,
		Messages: 0,
		Subject: subject,
		Last_Message: "none",
		Last_User: "none",
		Updated: t.Format(layout),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	err = put(ctx, roomTableName, av)
	if err != nil {
		return err
	}
	return nil
}

func updateMessage(ctx context.Context, messageTableName string, message_id int, value int, name string) error {
	an := map[string]string{
		"#s": name,
	}

	av_struct := struct {New int `dynamodbav:":new"`}{value}
	av, err := attributevalue.MarshalMap(av_struct)
	if err != nil {
		return err
	}

	key_struct := struct {Message int `dynamodbav:"message_id"`}{message_id}
	key, err := attributevalue.MarshalMap(key_struct)
	if err != nil {
		return err
	}

	updateExpression := "set #s = #s + :new"
	err = update(ctx, messageTableName, an, av, key, updateExpression)
	if err != nil {
		return err
	}
	return nil
}

func checkToken(ctx context.Context, tokenTableName string, token string) bool {
	item := struct {Token string `dynamodbav:"token"`}{token}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return false
	}
	res, err := get(ctx, tokenTableName, av, "token")
	if err == nil && res.Item != nil{
		return true
	}
	return false
}

func delete(ctx context.Context, tableName string, key map[string]types.AttributeValue) error {
	if dynamodbClient == nil {
		dynamodbClient = dynamodb.NewFromConfig(getConfig(ctx))
	}
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: key,
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	return err
}

func deleteToken(ctx context.Context, tokenTableName string, token string) error {
	item := struct {Token string `dynamodbav:"token"`}{token}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Print(err)
		return err
	}
	err = delete(ctx, tokenTableName, av)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
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
