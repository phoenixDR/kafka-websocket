package websocket_test

import (
	"github.com/golang/mock/gomock"
	"github.com/phoenixDR/kafka-websocket/internal/mocks"
	"github.com/phoenixDR/kafka-websocket/internal/websocket"
	"testing"
	"time"
)

func TestRegisterClient(t *testing.T) {
	type testCase struct {
		Name               string
		Client             *websocket.Client
		ExpectedSubscribed bool
	}

	tests := []testCase{
		{
			Name:               "Register valid client",
			Client:             &websocket.Client{},
			ExpectedSubscribed: true,
		},
		{
			Name:               "Register nil client",
			Client:             nil,
			ExpectedSubscribed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			s := &websocket.Server{
				Clients: make(map[*websocket.Client]bool),
			}

			s.RegisterClient(tc.Client)

			if tc.Client != nil {
				if _, exists := s.Clients[tc.Client]; exists != tc.ExpectedSubscribed {
					t.Errorf("Test %s failed, expected subscribed status: %v, got: %v", tc.Name, tc.ExpectedSubscribed, exists)
				}
			} else if len(s.Clients) != 0 {
				t.Errorf("Test %s failed, nil client should not be registered", tc.Name)
			}
		})
	}
}

func TestUnregisterClient(t *testing.T) {
	type testCase struct {
		name     string
		client   *websocket.Client
		expected bool
		mockConn *mocks.MockWsConnection
	}

	tests := []testCase{
		{
			name: "Unregister valid client",
			client: &websocket.Client{
				Conn: mocks.NewMockWsConnection(gomock.NewController(t)),
			},
			expected: false,
		},
		{
			name: "Unregister non-existent client",
			client: &websocket.Client{
				Conn: mocks.NewMockWsConnection(gomock.NewController(t)),
			},
			expected: false,
		},
		{
			name:     "Unregister nil client",
			client:   nil,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tc.client != nil {
				mockConn := mocks.NewMockWsConnection(ctrl)
				tc.client.Conn = mockConn
				mockConn.EXPECT().Close().Return(nil).AnyTimes()
			}

			s := websocket.Server{
				Clients: make(map[*websocket.Client]bool),
			}

			if tc.name == "Unregister valid client" {
				s.Clients[tc.client] = true
			}

			s.UnregisterClient(tc.client)

			_, exists := s.Clients[tc.client]
			if exists != tc.expected {
				t.Errorf("Test %s failed: expected %v, got %v", tc.name, !tc.expected, exists)
			}
		})
	}
}

func TestRun(t *testing.T) {
	type testCase struct {
		name     string
		action   func(*websocket.Server, *websocket.Client)
		validate func(*testing.T, *websocket.Server, *websocket.Client)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mocks.NewMockWsConnection(ctrl)
	mockConn.EXPECT().Close().Return(nil).AnyTimes()

	client := &websocket.Client{Conn: mockConn}

	tests := []testCase{
		{
			name: "Handle registration",
			action: func(s *websocket.Server, c *websocket.Client) {
				s.Register <- c
			},
			validate: func(t *testing.T, s *websocket.Server, c *websocket.Client) {
				if _, exists := s.Clients[c]; !exists {
					t.Errorf("Client was not registered")
				}
			},
		},
		{
			name: "Handle unregistration",
			action: func(s *websocket.Server, c *websocket.Client) {
				s.Clients[c] = true
				s.Unregister <- c
			},
			validate: func(t *testing.T, s *websocket.Server, c *websocket.Client) {
				if _, exists := s.Clients[c]; exists {
					t.Errorf("Client was not unregistered")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := &websocket.Server{
				Clients:    make(map[*websocket.Client]bool),
				Register:   make(chan *websocket.Client),
				Unregister: make(chan *websocket.Client),
				Broadcast:  make(chan string),
			}

			go server.Run()
			tc.action(server, client)
			time.Sleep(time.Second)
			tc.validate(t, server, client)

		})
	}
}

func TestBroadcastMessage(t *testing.T) {
	type testCase struct {
		name         string
		setupClients func(*gomock.Controller, *websocket.Server) map[*websocket.Client]bool
		message      string
	}

	tests := []testCase{
		{
			name: "Broadcast with no clients",
			setupClients: func(ctrl *gomock.Controller, s *websocket.Server) map[*websocket.Client]bool {
				return make(map[*websocket.Client]bool)
			},
			message: "test message",
		},
		{
			name: "Broadcast to multiple clients",
			setupClients: func(ctrl *gomock.Controller, s *websocket.Server) map[*websocket.Client]bool {
				client1 := &websocket.Client{
					Conn:       mocks.NewMockWsConnection(ctrl),
					Subscribed: map[string]bool{"test_topic": true},
				}
				client2 := &websocket.Client{
					Conn:       mocks.NewMockWsConnection(ctrl),
					Subscribed: map[string]bool{"test_topic": true},
				}
				s.Clients[client1] = true
				s.Clients[client2] = true

				return map[*websocket.Client]bool{
					client1: true,
					client2: true,
				}
			},
			message: "test message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := websocket.Server{
				Clients: make(map[*websocket.Client]bool),
			}

			expectedResults := tc.setupClients(ctrl, &s)

			for client, expected := range expectedResults {
				mockConn, ok := client.Conn.(*mocks.MockWsConnection)
				if !ok {
					t.Fatalf("Expected a MockWsConnection for client")
				}

				if expected {
					mockConn.EXPECT().SendMessage(tc.message).Return(nil).Times(1)
				}
			}

			s.BroadcastMessage(tc.message)
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestSendMessageToClient(t *testing.T) {
	type testCase struct {
		name       string
		client     *websocket.Client
		message    string
		shouldSend bool
	}

	tests := []testCase{
		{
			name:       "Send to valid client",
			client:     &websocket.Client{Conn: mocks.NewMockWsConnection(gomock.NewController(t))},
			message:    "test message",
			shouldSend: true,
		},
		{
			name:       "Send to nil client",
			client:     nil,
			message:    "test message",
			shouldSend: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := websocket.Server{}

			if tc.client != nil && tc.shouldSend {
				mockConn := tc.client.Conn.(*mocks.MockWsConnection)
				mockConn.EXPECT().SendMessage(tc.message).Return(nil)
			}

			s.SendMessageToClient(tc.client, tc.message)
		})
	}
}
