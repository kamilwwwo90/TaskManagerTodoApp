package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Api struct {
	client *http.Client
}

type ReqTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

type UpdateObject struct {
	Change string `json:"change"`
	Value  any    `json:"value"`
}

func (a *Api) login(user User) error {
	targetURL := "http://localhost:8081/Login"

	jsonData, jsonErr := json.Marshal(user)

	if jsonErr != nil {
		return jsonErr
	}

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodPost, targetURL, bytes.NewBuffer(jsonData))

	if reqErr != nil {
		return reqErr
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("The server has returned an unexpected status")
	}

	fmt.Println("Succesfull log in")

	return nil
}

func (a *Api) signup(user User) error {
	targetURL := "http://localhost:8081/Signup"

	jsonData, jsonErr := json.Marshal(user)

	if jsonErr != nil {
		return jsonErr
	}

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodPost, targetURL, bytes.NewBuffer(jsonData))

	if reqErr != nil {
		return reqErr
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("The server has returned an unexpected status")
	}

	fmt.Println("Succesfull sign up")

	loginErr := a.login(user)

	if loginErr != nil {
		return loginErr
	}

	return nil
}

func (a *Api) loadTasks() ([]Task, error) {
	client := a.client

	targetURL := "http://localhost:8081/Tasks"

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)

	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := client.Do(req)

	if respErr != nil {
		return nil, respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("The server has returned an unexpected status")
	}

	var payload []Task

	jsonErr := json.NewDecoder(resp.Body).Decode(&payload)

	if jsonErr != nil {
		return nil, fmt.Errorf("could not get tasks")
	}

	return payload, nil
}

func (a *Api) getTask(id string) (Task, error) {
	targetURL := "http://localhost:8081/Tasks/" + url.PathEscape(id)

	req, reqError := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)

	if reqError != nil {
		return Task{}, reqError
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return Task{}, respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Task{}, fmt.Errorf("The server has returned an unexpected status")
	}

	var payload Task

	jsonErr := json.NewDecoder(resp.Body).Decode(&payload)

	if jsonErr != nil {
		return Task{}, jsonErr
	}

	return payload, nil
}

func (a *Api) addTask(task ReqTask) (Task, error) {
	targetURL := "http://localhost:8081/Tasks"

	jsonData, jsonErr := json.Marshal(task)

	if jsonErr != nil {
		return Task{}, jsonErr
	}

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodPost, targetURL, bytes.NewBuffer(jsonData))

	if reqErr != nil {
		return Task{}, reqErr
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return Task{}, respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return Task{}, fmt.Errorf("The server has returned an unexpected status")
	}

	var payload Task

	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)

	if decodeErr != nil {
		return Task{}, decodeErr
	}

	return payload, nil
}

func (a *Api) deleteTask(taskID string) error {
	targetURL := "http://localhost:8081/Tasks/" + url.PathEscape(taskID)

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodDelete, targetURL, nil)

	if reqErr != nil {
		return reqErr
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("The server has returned an unexpected status")
	}

	return nil
}

func (a *Api) updateTask(taskID string, change string, value any) error {
	targetURL := "http://localhost:8081/Tasks/" + url.PathEscape(taskID)

	update := UpdateObject{
		Change: change,
		Value:  value,
	}

	jsonData, jsonErr := json.Marshal(update)

	if jsonErr != nil {
		return jsonErr
	}

	req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodPatch, targetURL, bytes.NewBuffer(jsonData))

	if reqErr != nil {
		return reqErr
	}

	resp, respErr := a.client.Do(req)

	if respErr != nil {
		return respErr
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("The server has returned an unexpected status")
	}

	return nil
}
