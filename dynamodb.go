import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
    "github.com/aws/aws-sdk-go/service/dynamodb/expression"

    "fmt"
    "log"
)

type Item struct {
    id int
    question string
    answer string
    option_1 string
    option_2 string
    option_3 string
    option_4 string
}

sess := session.Must(session.NewSessionWithOptions(session.Optons{
    SharedConfigState: session.SharedConfigEnable,
}))

db_client := dynamodb.New(sess)


func read() {
    id_filter := expression.Name("question_id").Equal(Expression.Value(year))
    questions_format := expression.NameList(
        expression.Name("question_id"),
        expression.Name("question"),
        expression.Name("answer"),
        expression.Name("option_1"),
        expression.Name("option_2"),
        expression.Name("option_3"),
        expression.Name("option_4"))

    expr, err := expression.NewBuilder().WithFilter(id_filter).WithProjection(question_format).Build()
    if err != nil {
    log.Fatalf("Got error building expression: %s", err)
    }

    // Build the query input parameters
    params := &dynamodb.ScanInput{
        ExpressionAttributeNames:  expr.Names(),
        ExpressionAttributeValues: expr.Values(),
        FilterExpression:          expr.Filter(),
        ProjectionExpression:      expr.Projection(),
        TableName:                 aws.String(tableName),
    }

    // Make the DynamoDB Query API call
    result, err := db_client.Scan(params)
    if err != nil {
        log.Fatalf("Query API call failed: %s", err)
    }

    fmt.Println(result[0].question)
}


func write(){
    item := Item{
        id: 0,
        question: "World's second largest Country",
        answer: "Canada",
        option_1: "USA",
        option_2: "Canada",
        option_3: "Australia",
        option_4: "Russia"
    }

    av, err := dynamodbattribute.MarshalMap(item)
    if err != nil {
        log.Fatalf("Got error marshalling new movie item: %s", err)
    }

    tableName := "questions"
    input := &dynamodb.PutItemInput{
        Item:      av,
        TableName: aws.String(tableName),
    }

    _, err = db_client.PutItem(input)
    if err != nil {
        log.Fatalf("Got error calling PutItem: %s", err)
    }
}





