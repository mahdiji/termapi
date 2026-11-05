package ui

import (
	"testing"

	"api/internal/collectionstore"
	"api/internal/model"
	requestinput "api/internal/ui/requestInput"
)

func TestManualAutochainingCaptureAndInject(t *testing.T) {
	state := model.NewSharedState()
	app := NewAppModel(state)

	// Simulasikan Request 1: Response login berhasil mengembalikan token
	loginResponse := requestinput.ResponseMsg{
		URL:     "http://localhost:8080/api/login",
		Method:  "POST",
		Code:    200,
		RawBody: `{"token": "secret_jwt_token_12345", "user": {"id": "user_777"}}`,
	}

	// Update app saat menerima ResponseMsg
	app.captureChainedVariables(loginResponse)

	// Verifikasi variabel berhasil ditangkap di SharedState
	if state.SessionVariables["token"] != "secret_jwt_token_12345" {
		t.Fatalf("Expected token 'secret_jwt_token_12345', got '%s'", state.SessionVariables["token"])
	}
	if state.SessionVariables["user.id"] != "user_777" {
		t.Fatalf("Expected user.id 'user_777', got '%s'", state.SessionVariables["user.id"])
	}

	// Simulasikan Request 2: User mengetik manual {{token}} pada header
	app.ReqState.Input.SetValue("http://localhost:8080/api/profile")
	app.ReqState.Headers.SetRows([]collectionstore.KVRow{{Key: "Authorization", Value: "Bearer {{token}}"}})
	app.ReqState.Body.SetSpec(collectionstore.BodySpec{Type: "json", Raw: `{"uid": "{{user.id}}"}`})

	// Jalankan SendRequest()
	cmd := app.ReqState.SendRequest()
	if cmd == nil {
		t.Fatal("Expected cmd from SendRequest")
	}
}
