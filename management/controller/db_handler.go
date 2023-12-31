package controller

import (
	"time"
	"strconv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

const layout = "2006-01-02 15:04"

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

func delete(tableName string, key map[string]*dynamodb.AttributeValue) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: key,
	}

	_, err := svc.DeleteItem(input)
	return err
}

func getMessageList(room_id int)([]MessageData, error)  {
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
		messageList = append(messageList, item)
	}
	return messageList, nil
}

func getRoomList()([]RoomData, error)  {
	var roomList []RoomData
	result, err := scan(roomTableName, expression.NotEqual(expression.Name("status"), expression.Value(-1)))
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

func getTokenList()([]TokenData, error)  {
	var tokenList []TokenData
	result, err := scan(tokenTableName, expression.NotEqual(expression.Name("token"), expression.Value("")))
	if err != nil {
		return nil, err
	}
	for _, i := range result.Items {
		item := TokenData{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}
		tokenList = append(tokenList, item)
	}
	return tokenList, nil
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
	updateExpression := "set #s = :new"
	err := update(messageTableName, an, av, key, updateExpression)
	if err != nil {
		return err
	}
	return nil
}

func updateRoomStatus(room_id int, value int, name string) error {
	an := map[string]*string{
		"#s": aws.String(name),
	}
	av := map[string]*dynamodb.AttributeValue{
		":new": {
			N: aws.String(strconv.Itoa(value)),
		},
	}
	key := map[string]*dynamodb.AttributeValue{
		"room_id": {
			N: aws.String(strconv.Itoa(room_id)),
		},
	}
	updateExpression := "set #s = :new"
	err := update(roomTableName, an, av, key, updateExpression)
	if err != nil {
		return err
	}
	return nil
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

func deleteToken(token string) error {
	key := map[string]*dynamodb.AttributeValue{
		"token": {
			S: aws.String(token),
		},
	}
	err := delete(tokenTableName, key)
	if err != nil {
		return err
	}
	return nil
}
