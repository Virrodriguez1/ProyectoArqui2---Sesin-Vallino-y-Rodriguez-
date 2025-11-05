package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"backend/search-api/services"
)

// PropertyMessage representa un mensaje sobre una propiedad
type PropertyMessage struct {
	Action    string `json:"action"`     // "create", "update", "delete"
	PropertyID string `json:"property_id"`
}

// RabbitMQConsumer consume mensajes de RabbitMQ para actualizar el índice de búsqueda
type RabbitMQConsumer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
	service    services.SearchService
}

// NewRabbitMQConsumer crea una nueva instancia de RabbitMQConsumer
func NewRabbitMQConsumer(rabbitURL, queueName string, service services.SearchService) (*RabbitMQConsumer, error) {
	log.Printf("Connecting to RabbitMQ at %s", rabbitURL)

	// Conectar con RabbitMQ
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	log.Printf("Successfully connected to RabbitMQ")

	// Crear channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	log.Printf("Channel created successfully")

	// Declarar la queue "properties_queue"
	queueNameFinal := queueName
	if queueNameFinal == "" {
		queueNameFinal = "properties_queue"
	}

	_, err = ch.QueueDeclare(
		queueNameFinal, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	log.Printf("Queue '%s' declared successfully", queueNameFinal)

	return &RabbitMQConsumer{
		connection: conn,
		channel:    ch,
		queueName:  queueNameFinal,
		service:    service,
	}, nil
}

// Start inicia el consumo de mensajes de RabbitMQ
func (c *RabbitMQConsumer) Start() error {
	log.Printf("Starting RabbitMQ consumer for queue '%s'", c.queueName)

	// Configurar QoS para procesar un mensaje a la vez
	err := c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Consumir mensajes de la queue
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack (manejamos manualmente)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Consumer registered, waiting for messages...")

	// Procesar mensajes
	go func() {
		for msg := range msgs {
			c.processMessage(msg)
		}
	}()

	return nil
}

// processMessage procesa un mensaje individual
func (c *RabbitMQConsumer) processMessage(msg amqp.Delivery) {
	log.Printf("Received message: %s", string(msg.Body))

	// Deserializar JSON a PropertyMessage
	var propertyMsg PropertyMessage
	if err := json.Unmarshal(msg.Body, &propertyMsg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		// Rechazar mensaje sin requeue si el formato es inválido
		msg.Nack(false, false)
		return
	}

	log.Printf("Processing message: Action=%s, PropertyID=%s", propertyMsg.Action, propertyMsg.PropertyID)

	// Validar mensaje
	if propertyMsg.PropertyID == "" {
		log.Printf("Error: PropertyID is empty in message")
		msg.Nack(false, false)
		return
	}

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Procesar según el Action
	var err error
	switch propertyMsg.Action {
	case "create":
		err = c.handleCreate(ctx, propertyMsg.PropertyID)
	case "update":
		err = c.handleUpdate(ctx, propertyMsg.PropertyID)
	case "delete":
		err = c.handleDelete(ctx, propertyMsg.PropertyID)
	default:
		log.Printf("Unknown action: %s", propertyMsg.Action)
		msg.Nack(false, false)
		return
	}

	// Manejar resultado
	if err != nil {
		log.Printf("Error processing message (Action=%s, PropertyID=%s): %v", propertyMsg.Action, propertyMsg.PropertyID, err)
		// Rechazar con requeue para reintentar
		msg.Nack(false, true)
		return
	}

	log.Printf("Successfully processed message: Action=%s, PropertyID=%s", propertyMsg.Action, propertyMsg.PropertyID)

	// ACK del mensaje
	if err := msg.Ack(false); err != nil {
		log.Printf("Error acknowledging message: %v", err)
	}
}

// handleCreate maneja la acción "create"
func (c *RabbitMQConsumer) handleCreate(ctx context.Context, propertyID string) error {
	log.Printf("Handling CREATE action for PropertyID=%s", propertyID)

	// 1. Obtener propiedad desde la API
	property, err := c.service.FetchPropertyFromAPI(propertyID)
	if err != nil {
		return fmt.Errorf("failed to fetch property from API: %w", err)
	}

	log.Printf("Fetched property from API: ID=%s, Title=%s", property.ID, property.Title)

	// 2. Indexar en Solr
	if err := c.service.IndexProperty(ctx, *property); err != nil {
		return fmt.Errorf("failed to index property: %w", err)
	}

	log.Printf("Successfully indexed property: ID=%s", propertyID)
	return nil
}

// handleUpdate maneja la acción "update"
func (c *RabbitMQConsumer) handleUpdate(ctx context.Context, propertyID string) error {
	log.Printf("Handling UPDATE action for PropertyID=%s", propertyID)

	// 1. Obtener propiedad desde la API
	property, err := c.service.FetchPropertyFromAPI(propertyID)
	if err != nil {
		return fmt.Errorf("failed to fetch property from API: %w", err)
	}

	log.Printf("Fetched property from API: ID=%s, Title=%s", property.ID, property.Title)

	// 2. Actualizar en Solr
	if err := c.service.UpdateProperty(ctx, *property); err != nil {
		return fmt.Errorf("failed to update property: %w", err)
	}

	log.Printf("Successfully updated property: ID=%s", propertyID)
	return nil
}

// handleDelete maneja la acción "delete"
func (c *RabbitMQConsumer) handleDelete(ctx context.Context, propertyID string) error {
	log.Printf("Handling DELETE action for PropertyID=%s", propertyID)

	// Eliminar de Solr
	if err := c.service.DeleteProperty(ctx, propertyID); err != nil {
		return fmt.Errorf("failed to delete property: %w", err)
	}

	log.Printf("Successfully deleted property: ID=%s", propertyID)
	return nil
}

// Close cierra las conexiones de RabbitMQ
func (c *RabbitMQConsumer) Close() error {
	log.Printf("Closing RabbitMQ consumer connections")

	var errs []error

	// Cerrar channel
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing channel: %w", err))
		} else {
			log.Printf("Channel closed successfully")
		}
	}

	// Cerrar connection
	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing connection: %w", err))
		} else {
			log.Printf("Connection closed successfully")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing RabbitMQ consumer: %v", errs)
	}

	log.Printf("RabbitMQ consumer closed successfully")
	return nil
}

