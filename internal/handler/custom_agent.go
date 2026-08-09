package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/im"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// CustomAgentHandler defines the HTTP handler for custom agent operations
type CustomAgentHandler struct {
	service      interfaces.CustomAgentService
	imService    *im.Service
	disabledRepo interfaces.TenantDisabledSharedAgentRepository
	// userService is only used by the list endpoint to batch-fill creator_name; see
	// KnowledgeBaseHandler.userService。
	userService interfaces.UserService
}

// NewCustomAgentHandler creates a new custom agent handler instance
func NewCustomAgentHandler(
	service interfaces.CustomAgentService,
	imService *im.Service,
	disabledRepo interfaces.TenantDisabledSharedAgentRepository,
	userService interfaces.UserService,
) *CustomAgentHandler {
	return &CustomAgentHandler{
		service:      service,
		imService:    imService,
		disabledRepo: disabledRepo,
		userService:  userService,
	}
}

// CreateAgentRequest defines the request body for creating an agent
type CreateAgentRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	Avatar      string                  `json:"avatar"`
	Config      types.CustomAgentConfig `json:"config"`
}

// UpdateAgentRequest defines the request body for updating an agent
type UpdateAgentRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Avatar      string                  `json:"avatar"`
	Config      types.CustomAgentConfig `json:"config"`
}

// CreateAgent godoc
// @Summary      Create agent
// @Description  Create a new custom agent
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        request  body      CreateAgentRequest  true  "Agent info"
// @Success      201      {object}  map[string]interface{}  "Created agent"
// @Failure      400      {object}  errors.AppError         "Invalid request parameters"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents [post]
func (h *CustomAgentHandler) CreateAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start creating custom agent")

	// Parse request body
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, req.Config); err != nil {
		c.Error(err)
		return
	}

	// Build agent object
	agent := &types.CustomAgent{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
	}
	agent.EnsureDefaults()
	if err := agent.Config.QuestionSuggestions.Validate(); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Creating custom agent, name: %s, agent_mode: %s",
		secutils.SanitizeForLog(req.Name), req.Config.AgentMode)

	// Create agent using the service
	createdAgent, err := h.service.CreateAgent(ctx, agent)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		if err == service.ErrAgentNameRequired {
			c.Error(errors.NewBadRequestError(err.Error()))
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Custom agent created successfully, ID: %s, name: %s",
		secutils.SanitizeForLog(createdAgent.ID), secutils.SanitizeForLog(createdAgent.Name))
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdAgent,
	})
}

// GetAgent godoc
// @Summary      Get agent details
// @Description  Get agent details by ID
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Agent ID"
// @Success      200  {object}  map[string]interface{}  "Agent details"
// @Failure      400  {object}  errors.AppError         "Invalid request parameters"
// @Failure      404  {object}  errors.AppError         "Agent does not exist"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [get]
func (h *CustomAgentHandler) GetAgent(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	agent, err := h.service.GetAgentByID(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		if err == service.ErrAgentNotFound {
			c.Error(errors.NewNotFoundError("Agent not found"))
			return
		}
		if appErr, ok := err.(*errors.AppError); ok {
			c.Error(appErr)
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    agent,
	})
}

// ListAgents godoc
// @Summary      Get agent list
// @Description  Get all agents in the current workspace (including built-in agents)
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Agent list"
// @Failure      500  {object}  errors.AppError         "Internal server error"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents [get]
func (h *CustomAgentHandler) ListAgents(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all agents for this tenant
	agents, err := h.service.ListAgents(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Optional creator filter — see the matching block in
	// KnowledgeBaseHandler.ListKnowledgeBases for rationale. Built-in
	// agents (IsBuiltin=true, CreatedBy="") are tenant-level fixtures
	// rather than user creations; we always keep them regardless of the
	// filter so the conversation dropdown never silently loses
	// quick-answer / smart-reasoning when a user picks "Created by me".
	creatorFilter := strings.ToLower(strings.TrimSpace(c.Query("creator")))
	if creatorFilter == "mine" || creatorFilter == "others" {
		callerUserID, _ := c.Get(types.UserIDContextKey.String())
		callerUserIDStr, _ := callerUserID.(string)
		filtered := make([]*types.CustomAgent, 0, len(agents))
		for _, ag := range agents {
			if ag.IsBuiltin {
				filtered = append(filtered, ag)
				continue
			}
			if ag.CreatedBy == "" {
				continue
			}
			if creatorFilter == "mine" && ag.CreatedBy == callerUserIDStr {
				filtered = append(filtered, ag)
			} else if creatorFilter == "others" && ag.CreatedBy != callerUserIDStr {
				filtered = append(filtered, ag)
			}
		}
		agents = filtered
	}

	// Per-tenant "disabled by me" for own agents (only affects this tenant's conversation dropdown)
	tenantIDVal, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Workspace ID not found in context")
		c.Error(errors.NewUnauthorizedError("Missing workspace context"))
		return
	}
	tenantID, ok := tenantIDVal.(uint64)
	if !ok {
		logger.Errorf(ctx, "Tenant ID has unexpected type %T in context", tenantIDVal)
		c.Error(errors.NewInternalServerError("Invalid workspace context type"))
		return
	}
	disabledOwnIDs, err := h.disabledRepo.ListDisabledOwnAgentIDs(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": tenantID,
		})
		c.Error(errors.NewInternalServerError("Failed to list disabled agent IDs: " + err.Error()))
		return
	}

	// Batch-fill creator_name, same purpose as the KB list: let the frontend distinguish "created by me" from "other members of the same space".
	// Built-in agents (IsBuiltin=true, CreatedBy="") have no creator_name; the frontend renders the builtin
	// branch separately.
	enrichAgentCreatorNames(ctx, h.userService, agents)

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"data":                   agents,
		"disabled_own_agent_ids": disabledOwnIDs,
	})
}

// enrichAgentCreatorNames batch-resolves agent.CreatedBy into display names. Failures are swallowed,
// so the list itself remains usable. Behavior aligned with enrichKBCreatorNames.
func enrichAgentCreatorNames(ctx context.Context, userSvc interfaces.UserService, agents []*types.CustomAgent) {
	if userSvc == nil || len(agents) == 0 {
		return
	}
	idSet := make(map[string]struct{}, len(agents))
	for _, ag := range agents {
		if ag.IsBuiltin || ag.CreatedBy == "" {
			continue
		}
		idSet[ag.CreatedBy] = struct{}{}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	users, err := userSvc.GetUsersByIDs(ctx, ids)
	if err != nil {
		logger.Warnf(ctx, "Failed to resolve agent creator names: %v", err)
		return
	}
	for _, ag := range agents {
		if ag.IsBuiltin || ag.CreatedBy == "" {
			continue
		}
		u, ok := users[ag.CreatedBy]
		if !ok || u == nil {
			continue
		}
		ag.CreatorName = pickUserDisplayName(u)
	}
}

// UpdateAgent godoc
// @Summary      Update agent
// @Description  Update an agent's name, description and configuration
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "Agent ID"
// @Param        request  body      UpdateAgentRequest  true  "Update request"
// @Success      200      {object}  map[string]interface{}  "Updated agent"
// @Failure      400      {object}  errors.AppError         "Invalid request parameters"
// @Failure      403      {object}  errors.AppError         "Cannot modify built-in agents"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [put]
func (h *CustomAgentHandler) UpdateAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	// Parse request body
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, req.Config); err != nil {
		c.Error(err)
		return
	}

	// Build agent object
	agent := &types.CustomAgent{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
	}
	agent.EnsureDefaults()
	if err := agent.Config.QuestionSuggestions.Validate(); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Updating custom agent, ID: %s, name: %s",
		secutils.SanitizeForLog(id), secutils.SanitizeForLog(req.Name))

	// Update the agent
	updatedAgent, err := h.service.UpdateAgent(ctx, agent)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		case service.ErrCannotModifyBuiltin:
			c.Error(errors.NewForbiddenError("Cannot modify built-in agent"))
		case service.ErrAgentNameRequired:
			c.Error(errors.NewBadRequestError(err.Error()))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent updated successfully, ID: %s", secutils.SanitizeForLog(id))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedAgent,
	})
}

// DeleteAgent godoc
// @Summary      Delete agent
// @Description  Delete the given agent
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Agent ID"
// @Success      200  {object}  map[string]interface{}  "Deleted successfully"
// @Failure      400  {object}  errors.AppError         "Invalid request parameters"
// @Failure      403  {object}  errors.AppError         "Cannot delete built-in agents"
// @Failure      404  {object}  errors.AppError         "Agent does not exist"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [delete]
func (h *CustomAgentHandler) DeleteAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Deleting custom agent, ID: %s", secutils.SanitizeForLog(id))

	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	if err := h.imService.DeleteChannelsByAgent(id, tenantID); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		c.Error(errors.NewInternalServerError("Failed to delete agent IM channels"))
		return
	}

	// Delete the agent
	err := h.service.DeleteAgent(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		case service.ErrCannotDeleteBuiltin:
			c.Error(errors.NewForbiddenError("Cannot delete built-in agent"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent deleted successfully, ID: %s", secutils.SanitizeForLog(id))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent deleted successfully",
	})
}

// CopyAgent godoc
// @Summary      Duplicate agent
// @Description  Duplicate the given agent
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Agent ID"
// @Success      201  {object}  map[string]interface{}  "Duplicated successfully"
// @Failure      400  {object}  errors.AppError         "Invalid request parameters"
// @Failure      404  {object}  errors.AppError         "Agent does not exist"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/copy [post]
func (h *CustomAgentHandler) CopyAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start copying custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Copying custom agent, ID: %s", secutils.SanitizeForLog(id))
	sourceAgent, err := h.service.GetAgentByID(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, sourceAgent.Config); err != nil {
		c.Error(err)
		return
	}

	// Copy the agent
	copiedAgent, err := h.service.CopyAgent(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent copied successfully, source ID: %s, new ID: %s",
		secutils.SanitizeForLog(id), secutils.SanitizeForLog(copiedAgent.ID))
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    copiedAgent,
	})
}

// GetPlaceholders godoc
// @Summary      Get placeholder definitions
// @Description  Get all available prompt placeholder definitions, grouped by field type
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Placeholder definitions"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/placeholders [get]
func (h *CustomAgentHandler) GetPlaceholders(c *gin.Context) {
	// Return all placeholder definitions grouped by field type
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"all":                   types.AllPlaceholders(),
			"system_prompt":         types.PlaceholdersByField(types.PromptFieldSystemPrompt),
			"agent_system_prompt":   types.PlaceholdersByField(types.PromptFieldAgentSystemPrompt),
			"context_template":      types.PlaceholdersByField(types.PromptFieldContextTemplate),
			"rewrite_system_prompt": types.PlaceholdersByField(types.PromptFieldRewriteSystemPrompt),
			"rewrite_prompt":        types.PlaceholdersByField(types.PromptFieldRewritePrompt),
			"fallback_prompt":       types.PlaceholdersByField(types.PromptFieldFallbackPrompt),
		},
	})
}

// GetAgentTypePresets godoc
// @Summary      Get agent type preset list
// @Description  Return all available agent type presets under smart-reasoning (RAG/Wiki/Hybrid/Custom) for auto-filling system prompts, tools and KB compatibility in the editor
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Preset list"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/type-presets [get]
func (h *CustomAgentHandler) GetAgentTypePresets(c *gin.Context) {
	ctx := c.Request.Context()
	presets := types.ListAgentTypePresetsWithContext(ctx)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    presets,
	})
}

// GetSuggestedQuestions godoc
// @Summary      Get suggested questions
// @Description  Return suggested questions based on the agent's linked knowledge bases for quick asking
// @Tags         Agents
// @Accept       json
// @Produce      json
// @Param        id                  path      string  true   "Agent ID"
// @Param        knowledge_base_ids  query     string  false  "Knowledge base ID list (comma-separated), overrides the agent default config"
// @Param        knowledge_ids       query     string  false  "Knowledge ID list (comma-separated), scoped to specific documents"
// @Param        tag_scopes          query     string  false  "Tag scope with knowledge base ownership (JSON)"
// @Param        limit               query     int     false  "Max number of results (falls back to the agent's configured opener question count, max 30)"
// @Success      200                 {object}  map[string]interface{}  "Suggested question list"
// @Failure      400                 {object}  errors.AppError         "Invalid request parameters"
// @Failure      404                 {object}  errors.AppError         "Agent does not exist"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/suggested-questions [get]
func (h *CustomAgentHandler) GetSuggestedQuestions(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	// Parse optional query parameters
	var kbIDs []string
	if kbIDsStr := strings.TrimSpace(c.Query("knowledge_base_ids")); kbIDsStr != "" {
		for _, id := range strings.Split(kbIDsStr, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				kbIDs = append(kbIDs, trimmed)
			}
		}
	}

	var knowledgeIDs []string
	if kIDsStr := strings.TrimSpace(c.Query("knowledge_ids")); kIDsStr != "" {
		for _, id := range strings.Split(kIDsStr, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				knowledgeIDs = append(knowledgeIDs, trimmed)
			}
		}
	}

	var tagScopes []types.TagScope
	if raw := strings.TrimSpace(c.Query("tag_scopes")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &tagScopes); err != nil {
			c.Error(errors.NewBadRequestError("tag_scopes must be valid JSON"))
			return
		}
	}

	// limit == 0 signals "unspecified" so the service falls back to the agent's
	// configured starter count. A provided value is passed through unchanged and
	// bounded by the service's safety cap.
	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logger.Infof(ctx, "Getting suggested questions for agent %s, kbIDs: %v, tagScopes: %d, limit: %d",
		secutils.SanitizeForLog(id), kbIDs, len(tagScopes), limit)

	questions, err := h.service.GetSuggestedQuestions(ctx, id, kbIDs, knowledgeIDs, tagScopes, limit)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		if err == service.ErrAgentNotFound {
			c.Error(errors.NewNotFoundError("Agent not found"))
			return
		}
		if appErr, ok := err.(*errors.AppError); ok {
			c.Error(appErr)
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"questions": questions,
		},
	})
}

func authorizeAgentKnowledgeScope(ctx context.Context, cfg types.CustomAgentConfig) error {
	scope, ok := types.TenantAPIKeyScopeFromContext(ctx)
	if !ok || !scope.IsKnowledgeBaseRestricted() {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.KBSelectionMode)) {
	case "none":
		return nil
	case "all":
		return errors.NewForbiddenError("API key scope does not allow agents that use all knowledge bases")
	case "selected":
		return types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, cfg.KnowledgeBases...)
	default:
		if len(cfg.KnowledgeBases) == 0 {
			return nil
		}
		return types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, cfg.KnowledgeBases...)
	}
}
