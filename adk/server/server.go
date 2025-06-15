package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	gin "github.com/gin-gonic/gin"
	uuid "github.com/google/uuid"
	adk "github.com/inference-gateway/a2a/adk"
	config "github.com/inference-gateway/a2a/adk/server/config"
	middlewares "github.com/inference-gateway/a2a/adk/server/middlewares"
	otel "github.com/inference-gateway/a2a/adk/server/otel"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	envconfig "github.com/sethvargo/go-envconfig"
	zap "go.uber.org/zap"
)

// A2AServer defines the interface for an A2A-compatible server
// This interface allows for easy testing and different implementations
type A2AServer interface {
	// Start starts the A2A server on the configured port
	Start(ctx context.Context) error

	// Stop gracefully stops the A2A server
	Stop(ctx context.Context) error

	// GetAgentCard returns the agent's capabilities and metadata
	GetAgentCard() adk.AgentCard

	// ProcessTask processes a task with the given message
	ProcessTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error)

	// StartTaskProcessor starts the background task processor
	StartTaskProcessor(ctx context.Context)

	// SetTaskHandler sets the task handler for processing tasks
	SetTaskHandler(handler TaskHandler)

	// GetTaskHandler returns the configured task handler
	GetTaskHandler() TaskHandler

	// SetAgent sets the OpenAI-compatible agent for processing tasks
	SetAgent(agent OpenAICompatibleAgent)

	// GetAgent returns the configured OpenAI-compatible agent
	GetAgent() OpenAICompatibleAgent

	// SetAgentName sets the agent's name dynamically
	SetAgentName(name string)

	// SetAgentDescription sets the agent's description dynamically
	SetAgentDescription(description string)

	// SetAgentURL sets the agent's URL dynamically
	SetAgentURL(url string)

	// SetAgentVersion sets the agent's version dynamically
	SetAgentVersion(version string)
}

// TaskResultProcessor defines how to process tool call results for task completion
type TaskResultProcessor interface {
	// ProcessToolResult processes a tool call result and returns a completion message if the task should be completed
	// Returns nil if the task should continue processing
	ProcessToolResult(toolCallResult string) *adk.Message
}

// JRPCErrorCode represents JSON-RPC error codes
type JRPCErrorCode int

const (
	ErrParseError     JRPCErrorCode = -32700
	ErrInvalidRequest JRPCErrorCode = -32600
	ErrMethodNotFound JRPCErrorCode = -32601
	ErrInvalidParams  JRPCErrorCode = -32602
	ErrInternalError  JRPCErrorCode = -32603
	ErrServerError    JRPCErrorCode = -32000
)

// QueuedTask represents a task in the processing queue
type QueuedTask struct {
	Task      *adk.Task
	RequestID interface{}
}

type A2AServerImpl struct {
	cfg            *config.Config
	logger         *zap.Logger
	taskHandler    TaskHandler
	taskManager    TaskManager
	messageHandler MessageHandler
	responseSender ResponseSender
	otel           otel.OpenTelemetry

	// Server state
	httpServer    *http.Server
	metricsServer *http.Server
	taskQueue     chan *QueuedTask

	// Optional processors
	taskResultProcessor TaskResultProcessor
	agent               OpenAICompatibleAgent
}

var _ A2AServer = (*A2AServerImpl)(nil)

// NewA2AServer creates a new A2A server with the provided configuration and logger
func NewA2AServer(cfg *config.Config, logger *zap.Logger, otel otel.OpenTelemetry) *A2AServerImpl {
	server := &A2AServerImpl{
		cfg:       cfg,
		logger:    logger,
		otel:      otel,
		taskQueue: make(chan *QueuedTask, cfg.QueueConfig.MaxSize),
	}

	maxConversationHistory := 20
	if cfg.AgentConfig != nil {
		maxConversationHistory = cfg.AgentConfig.MaxConversationHistory
	}

	server.taskManager = NewDefaultTaskManager(logger, maxConversationHistory)
	server.messageHandler = NewDefaultMessageHandler(logger, server.taskManager)
	server.responseSender = NewDefaultResponseSender(logger)
	server.taskHandler = NewDefaultTaskHandler(logger)

	return server
}

// NewA2AServerWithAgent creates a new A2A server with an optional OpenAI-compatible agent
func NewA2AServerWithAgent(cfg *config.Config, logger *zap.Logger, otel otel.OpenTelemetry, agent OpenAICompatibleAgent) *A2AServerImpl {
	server := NewA2AServer(cfg, logger, otel)

	if agent != nil {
		server.agent = agent
		server.taskHandler = NewAgentTaskHandler(logger, agent)
	}

	return server
}

// NewDefaultA2AServer creates a new default A2A server implementation
func NewDefaultA2AServer(cfg *config.Config) *A2AServerImpl {
	var finalCfg *config.Config
	var err error

	finalCfg, err = config.LoadWithLookuper(context.Background(), cfg, envconfig.OsLookuper())
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	var logger *zap.Logger
	if finalCfg.Debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var telemetryInstance otel.OpenTelemetry
	if finalCfg.TelemetryConfig != nil && finalCfg.TelemetryConfig.Enable {
		telemetryInstance, err = otel.NewOpenTelemetry(finalCfg, logger)
		if err != nil {
			logger.Fatal("failed to initialize telemetry", zap.Error(err))
		}
		logger.Info("telemetry enabled - metrics will be available on :9090/metrics")
	}

	server := NewA2AServer(finalCfg, logger, telemetryInstance)

	return server
}

// NewA2AServerEnvironmentAware creates a new A2A server with environment-aware configuration.
func NewA2AServerEnvironmentAware(cfg *config.Config, logger *zap.Logger, otel otel.OpenTelemetry) *A2AServerImpl {
	var err error
	cfg, err = config.LoadWithLookuper(context.Background(), cfg, envconfig.OsLookuper())
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	server := &A2AServerImpl{
		cfg:       cfg,
		logger:    logger,
		otel:      otel,
		taskQueue: make(chan *QueuedTask, cfg.QueueConfig.MaxSize),
	}

	// Safely get max conversation history from config
	maxConversationHistory := 20 // default value
	if cfg.AgentConfig != nil {
		maxConversationHistory = cfg.AgentConfig.MaxConversationHistory
	}

	server.taskManager = NewDefaultTaskManager(logger, maxConversationHistory)
	server.messageHandler = NewDefaultMessageHandler(logger, server.taskManager)
	server.responseSender = NewDefaultResponseSender(logger)
	server.taskHandler = NewDefaultTaskHandler(logger)

	return server
}

// SetTaskHandler allows injecting a custom task handler
func (s *A2AServerImpl) SetTaskHandler(handler TaskHandler) {
	s.taskHandler = handler
}

// GetTaskHandler returns the configured task handler
func (s *A2AServerImpl) GetTaskHandler() TaskHandler {
	return s.taskHandler
}

// SetAgent sets the OpenAI-compatible agent for processing tasks
func (s *A2AServerImpl) SetAgent(agent OpenAICompatibleAgent) {
	s.agent = agent
}

// GetAgent returns the configured OpenAI-compatible agent
func (s *A2AServerImpl) GetAgent() OpenAICompatibleAgent {
	return s.agent
}

// SetTaskResultProcessor sets the task result processor for custom business logic
func (s *A2AServerImpl) SetTaskResultProcessor(processor TaskResultProcessor) {
	s.taskResultProcessor = processor
}

// SetAgentName sets the agent's name dynamically
func (s *A2AServerImpl) SetAgentName(name string) {
	s.cfg.AgentName = name
}

// SetAgentDescription sets the agent's description dynamically
func (s *A2AServerImpl) SetAgentDescription(description string) {
	s.cfg.AgentDescription = description
}

// SetAgentURL sets the agent's URL dynamically
func (s *A2AServerImpl) SetAgentURL(url string) {
	s.cfg.AgentURL = url
}

// SetAgentVersion sets the agent's version dynamically
func (s *A2AServerImpl) SetAgentVersion(version string) {
	s.cfg.AgentVersion = version
}

// SetupRouter configures the HTTP router with A2A endpoints
func (s *A2AServerImpl) setupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.GET("/.well-known/agent.json", s.handleAgentInfo)

	var telemetryMiddleware gin.HandlerFunc
	if s.cfg.TelemetryConfig != nil && s.cfg.TelemetryConfig.Enable && s.otel != nil {
		telemetryMw, err := middlewares.NewTelemetryMiddleware(*s.cfg, s.otel, s.logger)
		if err != nil {
			s.logger.Error("failed to create telemetry middleware", zap.Error(err))
		} else {
			telemetryMiddleware = telemetryMw.Middleware()
		}
	}

	if s.cfg.AuthConfig == nil || !s.cfg.AuthConfig.Enable {
		if telemetryMiddleware != nil {
			r.POST("/a2a", telemetryMiddleware, s.handleA2ARequest)
		} else {
			r.POST("/a2a", s.handleA2ARequest)
		}
		s.logger.Warn("authentication is disabled, oidcAuthenticator will be nil")
		return r
	}
	oidcAuthenticator, err := middlewares.NewOIDCAuthenticatorMiddleware(s.logger, *s.cfg)
	if err != nil {
		s.logger.Error("failed to create OIDC authenticator", zap.Error(err))
		return r
	}

	s.logger.Info("oidcAuthenticator is valid, setting up authentication")
	if telemetryMiddleware != nil {
		r.POST("/a2a", telemetryMiddleware, oidcAuthenticator.Middleware(), s.handleA2ARequest)
	} else {
		r.POST("/a2a", oidcAuthenticator.Middleware(), s.handleA2ARequest)
	}

	return r
}

// Start starts the A2A server
func (s *A2AServerImpl) Start(ctx context.Context) error {
	router := s.setupRouter(s.cfg)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", s.cfg.Port),
		Handler:      router,
		ReadTimeout:  s.cfg.ServerConfig.ReadTimeout,
		WriteTimeout: s.cfg.ServerConfig.WriteTimeout,
		IdleTimeout:  s.cfg.ServerConfig.IdleTimeout,
	}

	s.logger.Info("starting A2A server", zap.String("port", s.cfg.Port))

	if s.cfg.TelemetryConfig != nil && s.cfg.TelemetryConfig.Enable && s.otel != nil {
		go func() {
			metricsRouter := gin.Default()
			metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))

			s.metricsServer = &http.Server{
				Addr:    ":9090",
				Handler: metricsRouter,
			}

			s.logger.Info("starting metrics server on port 9090")
			if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.Error("metrics server failed", zap.Error(err))
			}
		}()
	}

	go s.StartTaskProcessor(ctx)

	if s.cfg.TLSConfig != nil && s.cfg.TLSConfig.Enable {
		return s.httpServer.ListenAndServeTLS(s.cfg.TLSConfig.CertPath, s.cfg.TLSConfig.KeyPath)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the A2A server
func (s *A2AServerImpl) Stop(ctx context.Context) error {
	s.logger.Info("stopping A2A server")

	var err error

	if s.httpServer != nil {
		if shutdownErr := s.httpServer.Shutdown(ctx); shutdownErr != nil {
			s.logger.Error("error stopping HTTP server", zap.Error(shutdownErr))
			err = shutdownErr
		}
	}

	if s.metricsServer != nil {
		if shutdownErr := s.metricsServer.Shutdown(ctx); shutdownErr != nil {
			s.logger.Error("error stopping metrics server", zap.Error(shutdownErr))
			if err == nil {
				err = shutdownErr
			}
		}
	}

	if s.otel != nil {
		if shutdownErr := s.otel.ShutDown(ctx); shutdownErr != nil {
			s.logger.Error("error shutting down telemetry", zap.Error(shutdownErr))
			if err == nil {
				err = shutdownErr
			}
		}
	}

	defer func() {
		if syncErr := s.logger.Sync(); syncErr != nil {
			s.logger.Error("failed to sync logger on shutdown", zap.Error(syncErr))
		}
	}()

	return err
}

// GetAgentCard returns the agent's capabilities and metadata
func (s *A2AServerImpl) GetAgentCard() adk.AgentCard {
	capabilities := adk.AgentCapabilities{}

	if s.cfg.CapabilitiesConfig != nil {
		capabilities.Streaming = &s.cfg.CapabilitiesConfig.Streaming
		capabilities.PushNotifications = &s.cfg.CapabilitiesConfig.PushNotifications
		capabilities.StateTransitionHistory = &s.cfg.CapabilitiesConfig.StateTransitionHistory
	} else {
		defaultStreaming := true
		defaultPushNotifications := true
		defaultStateTransitionHistory := false
		capabilities.Streaming = &defaultStreaming
		capabilities.PushNotifications = &defaultPushNotifications
		capabilities.StateTransitionHistory = &defaultStateTransitionHistory
	}

	return adk.AgentCard{
		Name:               s.cfg.AgentName,
		Description:        s.cfg.AgentDescription,
		URL:                s.cfg.AgentURL,
		Version:            s.cfg.AgentVersion,
		Capabilities:       capabilities,
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills:             []adk.AgentSkill{},
	}
}

// ProcessTask processes a task with the given message
func (s *A2AServerImpl) ProcessTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error) {
	if s.agent != nil {
		s.logger.Info("processing task with openai-compatible agent",
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID))
		return s.agent.ProcessTask(ctx, task, message)
	}

	return s.taskHandler.HandleTask(ctx, task, message)
}

// StartTaskProcessor starts the background task processing goroutine
func (s *A2AServerImpl) StartTaskProcessor(ctx context.Context) {
	s.logger.Info("starting task processor")

	go s.startTaskCleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("task processor shutting down")
			return
		case queuedTask := <-s.taskQueue:
			s.processQueuedTask(ctx, queuedTask)
		}
	}
}

// processQueuedTask processes a single queued task
func (s *A2AServerImpl) processQueuedTask(ctx context.Context, queuedTask *QueuedTask) {
	task := queuedTask.Task

	var message *adk.Message
	if task.Status.Message != nil {
		message = task.Status.Message
	} else {
		message = &adk.Message{
			Kind:      "message",
			MessageID: uuid.New().String(),
			Role:      "user",
			Parts:     []adk.Part{},
		}
	}

	s.logger.Info("processing task",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID))

	err := s.taskManager.UpdateTask(task.ID, adk.TaskStateWorking, nil)
	if err != nil {
		s.logger.Error("failed to update task state", zap.Error(err))
		return
	}

	updatedTask, err := s.taskHandler.HandleTask(ctx, task, message)
	if err != nil {
		s.logger.Error("failed to process task",
			zap.Error(err),
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID))
		updateErr := s.taskManager.UpdateTask(task.ID, adk.TaskStateFailed, &adk.Message{
			Kind:      "message",
			MessageID: uuid.New().String(),
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": err.Error(),
				},
			},
		})
		if updateErr != nil {
			s.logger.Error("failed to update task to failed state",
				zap.Error(updateErr),
				zap.String("task_id", task.ID),
				zap.String("context_id", task.ContextID))
		}
		return
	}

	if err := s.taskManager.UpdateTask(updatedTask.ID, updatedTask.Status.State, nil); err != nil {
		s.logger.Error("failed to update task status",
			zap.Error(err),
			zap.String("task_id", updatedTask.ID),
			zap.String("context_id", updatedTask.ContextID))
		return
	}
	s.logger.Info("task processed successfully",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID))
}

// startTaskCleanup starts the background task cleanup process
func (s *A2AServerImpl) startTaskCleanup(ctx context.Context) {
	// Apply safe defaults if queue config is nil
	cleanupInterval := 30 * time.Second
	if s.cfg.QueueConfig != nil {
		cleanupInterval = s.cfg.QueueConfig.CleanupInterval
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("task cleanup shutting down")
			return
		case <-ticker.C:
			s.taskManager.CleanupCompletedTasks()
		}
	}
}

// handleAgentInfo returns agent capabilities and metadata
func (s *A2AServerImpl) handleAgentInfo(c *gin.Context) {
	s.logger.Info("agent info requested")
	agentCard := s.GetAgentCard()
	c.JSON(http.StatusOK, agentCard)
}

// handleA2ARequest processes A2A protocol requests
func (s *A2AServerImpl) handleA2ARequest(c *gin.Context) {
	var req adk.JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("failed to parse json request", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrParseError), "parse error")
		return
	}

	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	if req.ID == nil {
		id := interface{}(uuid.New().String())
		req.ID = &id
	}

	s.logger.Info("received a2a request",
		zap.String("method", req.Method),
		zap.Any("id", req.ID))

	switch req.Method {
	case "message/send":
		s.handleMessageSend(c, req)
	case "message/stream":
		s.handleMessageStream(c, req)
	case "tasks/get":
		s.handleTaskGet(c, req)
	case "tasks/cancel":
		s.handleTaskCancel(c, req)
	default:
		s.logger.Warn("unknown method requested", zap.String("method", req.Method))
		s.responseSender.SendError(c, req.ID, int(ErrMethodNotFound), "method not found")
	}
}

// handleMessageSend processes message/send requests
func (s *A2AServerImpl) handleMessageSend(c *gin.Context, req adk.JSONRPCRequest) {
	var params adk.MessageSendParams
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.logger.Error("failed to marshal params", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid params")
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		s.logger.Error("failed to parse message/send request", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid request")
		return
	}

	task, err := s.messageHandler.HandleMessageSend(c.Request.Context(), params)
	if err != nil {
		s.logger.Error("failed to handle message send", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInternalError), err.Error())
		return
	}

	queuedTask := &QueuedTask{
		Task:      task,
		RequestID: req.ID,
	}

	select {
	case s.taskQueue <- queuedTask:
		s.logger.Info("task queued for processing",
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID))
	default:
		s.logger.Error("task queue is full")
		err := s.taskManager.UpdateTask(task.ID, adk.TaskStateFailed, &adk.Message{
			Kind:      "message",
			MessageID: uuid.New().String(),
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "Task queue is full. Please try again later.",
				},
			},
		})
		if err != nil {
			s.logger.Error("failed to update task to failed state due to full queue",
				zap.Error(err),
				zap.String("task_id", task.ID),
				zap.String("context_id", task.ContextID))
		}
	}

	s.responseSender.SendSuccess(c, req.ID, *task)
}

// handleMessageStream processes message/stream requests
func (s *A2AServerImpl) handleMessageStream(c *gin.Context, req adk.JSONRPCRequest) {
	var params adk.MessageSendParams
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.logger.Error("failed to marshal params", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid params")
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		s.logger.Error("failed to parse message/stream request", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid request")
		return
	}

	ctx := c.Request.Context()
	err = s.messageHandler.HandleMessageStream(ctx, params)
	if err != nil {
		s.logger.Error("failed to handle message stream", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInternalError), err.Error())
		return
	}

	s.responseSender.SendSuccess(c, req.ID, "streaming started successfully")
}

// handleTaskGet processes tasks/get requests
func (s *A2AServerImpl) handleTaskGet(c *gin.Context, req adk.JSONRPCRequest) {
	var params adk.TaskQueryParams
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.logger.Error("failed to marshal params", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid params")
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		s.logger.Error("failed to parse tasks/get request", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid request")
		return
	}

	s.logger.Info("retrieving task", zap.String("task_id", params.ID))

	task, exists := s.taskManager.GetTask(params.ID)
	if !exists {
		s.logger.Error("task not found", zap.String("task_id", params.ID))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "task not found")
		return
	}

	s.logger.Info("task retrieved successfully",
		zap.String("task_id", params.ID),
		zap.String("context_id", task.ContextID),
		zap.String("status", string(task.Status.State)))
	s.responseSender.SendSuccess(c, req.ID, *task)
}

// handleTaskCancel processes tasks/cancel requests
func (s *A2AServerImpl) handleTaskCancel(c *gin.Context, req adk.JSONRPCRequest) {
	var params adk.TaskIdParams
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.logger.Error("failed to marshal params", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid params")
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		s.logger.Error("failed to parse tasks/cancel request", zap.Error(err))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), "invalid request")
		return
	}

	s.logger.Info("canceling task", zap.String("task_id", params.ID))

	err = s.taskManager.CancelTask(params.ID)
	if err != nil {
		s.logger.Error("failed to cancel task",
			zap.Error(err),
			zap.String("task_id", params.ID))
		s.responseSender.SendError(c, req.ID, int(ErrInvalidParams), err.Error())
		return
	}

	task, _ := s.taskManager.GetTask(params.ID)
	s.responseSender.SendSuccess(c, req.ID, *task)
}
