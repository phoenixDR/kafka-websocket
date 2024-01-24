# Kafka communication via Websocket

## Overview

Create two microservices - a producer and a consumer - that interact with each other using a message exchange system (Kafka, Nats, or RabbitMQ). The consumer should be able to send data through a WebSocket.

### Requirements

#### Producer Microservice
- Sends events (messages) to the message exchange system (Kafka, Nats, RabbitMQ etc.).
- Provides generation of test events or messages for sending.

#### Consumer Microservice
- Receives events (messages) from the message exchange system.
- Provides the ability to connect via WebSocket.
- Sends the received data through a WebSocket to connected clients.

#### WebSocket
- The consumer must provide data through a WebSocket.
- The WebSocket must be able to handle connections from different clients.
- To receive messages, a client must subscribe to the message topic.
- A client can unsubscribe from the topic to stop receiving messages.

#### Docker Compose
- Ensure the ability to launch both microservices and the message exchange system (if necessary) using Docker Compose.

## Getting Started

These instructions will get copy of the project up and running on local machine for development and testing purposes.

### Prerequisites

Install Docker if needed. Install Postman for testing purposes.

### Installing

A step-by-step series of examples that tell how to get a development environment running.

#### Using Docker Compose

1. Clone the repository to local machine.
2. Navigate to the root directory of the project.
3. Run the following command to start kafka services:

   ```bash
   docker-compose up

This command builds and starts all the services defined in docker-compose.yaml file Kafka, Zookeeper, and Kafka-UI.

For proper start of the consumer and producer services run them in terminal in separate windows:

```bash
 go run ./cmd/consumer
```
and
```bash
 go run ./cmd/producer
```

#### Accessing Kafka-UI
Open a web browser and navigate to http://localhost:8082 to access Kafka-UI.

## Testing with Postman
To test the WebSocket server using Postman:

1. Open Postman and create a new WebSocket request.

2. Set the request URL to ws://localhost:8888/ws.

3. Add the Origin header with the value http://localhost or local IP address.

4. Connect to the WebSocket server.

5. Send a message in JSON format with action and topic keys. For example:

```bash
{"action": "subscribe", "topic": "crypto-prices"}
```

### Important Notes
1. Make sure Docker Desktop (or Docker environment) is running before executing docker-compose up --build.
2. The WebSocket server runs on port 8888 by default. Ensure this port is available on local machine or adjust the port in the Docker configuration if necessary.
3. The WebSocket server expects specific message formats for subscriptions. Follow the prescribed format for successful communication.