# serverless-message-board kit
Simple kit for serverless message board using AWS Lambda.


## Dependence
- aws-lambda-go


## Requirements
- AWS (Lambda, API Gateway, DynamoDB)
- aws-cli
- golang environment


## DynamoDB Setting
- make 3 table
```
Table Name: room
Partition key Name: room_id
Partition key Type: Number

Table Name: message
Partition key Name: message_id
Partition key Type: Number

Table Name: token
Partition key Name: token
Partition key Type: String
```


## Usage

### Edit View
##### HTML
- Edit templates/index.html

##### CSS
- Edit static/css/main.css

##### Javascript
- Edit static/js/main.js

##### Image
- Add image file into static/img/
- Edit templates/index.html like as 'enter.jpg'.

### Deploy
Open scripts/deploy.sh and edit 'your_function_name'.

Open api/scripts/deploy.sh and edit 'your_api_function_name'.

Open constant/constant.json and edit 'your_api_url'.


Then run this command.

```
$ sh scripts/deploy.sh
$ cd api
$ sh scripts/deploy.sh
```
