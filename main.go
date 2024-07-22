package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/go-sql-driver/mysql"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var db *sql.DB

type DBSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Dbname   string `json:"dbname"`
}

func init() {
	secretName := os.Getenv("DB_SECRET_NAME")
	region := os.Getenv("AWS_REGION")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatalf("failed to create session, %v", err)
	}

	svc := secretsmanager.New(sess)

	result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		log.Fatalf("failed to get secret value, %v", err)
	}

	var secret DBSecret
	err = json.Unmarshal([]byte(*result.SecretString), &secret)
	if err != nil {
		log.Fatalf("failed to unmarshal secret, %v", err)
	}

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		secret.Username, secret.Password, secret.Host, secret.Port, secret.Dbname)
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatalf("failed to open database, %v", err)
	}
}

type Question struct {
	ID            int      `json:"id"`
	Question      string   `json:"question"`
	Choices       []string `json:"choices"`
	Answer        string   `json:"answer"`
	CreatedByTeam string   `json:"created_by_team"`
}

func getNextQuestionHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	query := "SELECT id, question, choices, answer, created_by_team FROM questions ORDER BY RAND() LIMIT 1"
	row := db.QueryRow(query)

	var question Question
	var choices string

	err := row.Scan(&question.ID, &question.Question, &choices, &question.Answer, &question.CreatedByTeam)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	err = json.Unmarshal([]byte(choices), &question.Choices)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	response, err := json.Marshal(question)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(response),
	}, nil
}

func main() {
	lambda.Start(getNextQuestionHandler)
}
