#!/bin/bash
if [ -z "$API_FUNCTION_NAME" ]; then
API_FUNCTION_NAME='your_api_function_name'
fi
if [ -z "$ROOM_TABLE_NAME" ]; then
ROOM_TABLE_NAME='sample_room'
fi
if [ -z "$MESSAGE_TABLE_NAME" ]; then
MESSAGE_TABLE_NAME='sample_message'
fi
if [ -z "$TOKEN_TABLE_NAME" ]; then
TOKEN_TABLE_NAME='sample_token'
fi
API_ROLE_NAME='your-api-lambda-role'
REGION='ap-northeast-1'
aws iam create-role --role-name $API_ROLE_NAME --path /service-role/ --assume-role-policy-document file://`pwd`/`dirname $0`/policy.json
API_ROLE_ARN=`aws iam get-role --role-name $API_ROLE_NAME | jq -r  .'Role.Arn'`
aws iam attach-role-policy --role-name $API_ROLE_NAME --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
aws iam attach-role-policy --role-name $API_ROLE_NAME --policy-arn "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
cd `dirname $0`/../
echo "{\"roomTableName\":\"$ROOM_TABLE_NAME\", \"messageTableName\":\"$MESSAGE_TABLE_NAME\", \"tokenTableName\":\"$TOKEN_TABLE_NAME\"}" > constant/constant.json

echo 'Create API Lambda-Function...'
rm function.zip
rm main
zip -r9 function.zip constant
GOOS=linux go build main.go
zip -g function.zip main
aws lambda create-function \
	--function-name $API_FUNCTION_NAME \
	--runtime go1.x \
	--role $API_ROLE_ARN \
	--handler main \
	--zip-file fileb://`pwd`/function.zip \
	--region $REGION > tmp.txt
