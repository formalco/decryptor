package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	_ "decryptor/provider/awskms"
	_ "decryptor/provider/gcpkms"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
)

type Response struct {
	Message string `json:"message"`
}

var corsHeaders = map[string]string{
	"Access-Control-Allow-Origin":  "https://app.formal.ai",
	"Access-Control-Allow-Methods": "OPTIONS,POST",
}

func decrypt(ctx context.Context, body string) (plaintext string, status int, errBody string) {
	obj, err := parseJWE(body)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse JWE from request body")
		return "", http.StatusBadRequest, "request body is not a valid JWE"
	}
	plaintext, err = decryptJWE(ctx, obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to decrypt JWE")
		return "", http.StatusInternalServerError, "could not decrypt the JWE"
	}
	return plaintext, http.StatusOK, ""
}

func lambdaHandler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	plaintext, status, errBody := decrypt(ctx, event.Body)
	if status != http.StatusOK {
		return events.APIGatewayProxyResponse{StatusCode: status, Body: errBody, Headers: corsHeaders}, nil
	}
	body, err := json.Marshal(&Response{Message: plaintext})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal decrypt response")
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "internal error", Headers: corsHeaders}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(body), Headers: corsHeaders}, nil
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	for k, v := range corsHeaders {
		w.Header().Set(k, v)
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("could not read request body"))
		return
	}

	plaintext, status, errBody := decrypt(r.Context(), string(body))
	if status != http.StatusOK {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(errBody))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&Response{Message: plaintext}); err != nil {
		log.Error().Err(err).Msg("failed to write decrypt response")
	}
}

func main() {
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(lambdaHandler)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/decrypt", httpHandler)
	log.Info().Msgf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal().Err(err).Msg("http server failed")
	}
}
