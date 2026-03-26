package cms

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// themeResponse is the API representation of a theme, hiding the git_token.
type themeResponse struct {
	ID          int       `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Author      string    `json:"author"`
	Source      string    `json:"source"`
	GitURL      *string   `json:"git_url"`
	GitBranch   string    `json:"git_branch"`
	HasGitToken bool      `json:"has_git_token"`
	IsActive    bool      `json:"is_active"`
	Path        string    `json:"path"`
	Thumbnail   *string   `json:"thumbnail"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// toThemeResponse converts a Theme model to a themeResponse, hiding the git token.
func toThemeResponse(t *models.Theme) themeResponse {
	return themeResponse{
		ID:          t.ID,
		Slug:        t.Slug,
		Name:        t.Name,
		Description: t.Description,
		Version:     t.Version,
		Author:      t.Author,
		Source:      t.Source,
		GitURL:      t.GitURL,
		GitBranch:   t.GitBranch,
		HasGitToken: t.GitToken != nil && *t.GitToken != "",
		IsActive:    t.IsActive,
		Path:        t.Path,
		Thumbnail:   t.Thumbnail,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ThemeHandler provides HTTP handlers for theme management.
type ThemeHandler struct {
	db      *gorm.DB
	mgmtSvc *ThemeMgmtService
}

// NewThemeHandler creates a new ThemeHandler.
func NewThemeHandler(db *gorm.DB, mgmtSvc *ThemeMgmtService) *ThemeHandler {
	return &ThemeHandler{db: db, mgmtSvc: mgmtSvc}
}

// RegisterRoutes registers all admin API theme routes on the provided router group.
// All routes require the manage_settings capability.
func (h *ThemeHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/themes", auth.CapabilityRequired("manage_settings"))
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/upload", h.Upload)
	g.Post("/git", h.InstallFromGit)
	g.Post("/:id/activate", h.Activate)
	g.Post("/:id/deactivate", h.Deactivate)
	g.Post("/:id/pull", h.Pull)
	g.Delete("/:id", h.Delete)
	g.Post("/:id/git-config", h.UpdateGitConfig)
}

// RegisterWebhook registers the public theme deploy webhook endpoint.
func (h *ThemeHandler) RegisterWebhook(app *fiber.App) {
	app.Post("/api/v1/theme-deploy", h.webhookDeploy)
}

// List handles GET /themes — returns all themes with git_token hidden.
func (h *ThemeHandler) List(c *fiber.Ctx) error {
	var themes []models.Theme
	if err := h.db.Order("name ASC").Find(&themes).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list themes")
	}

	resp := make([]themeResponse, len(themes))
	for i := range themes {
		resp[i] = toThemeResponse(&themes[i])
	}
	return api.Success(c, resp)
}

// Get handles GET /themes/:id — returns a single theme with git_token hidden.
func (h *ThemeHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	var theme models.Theme
	if err := h.db.First(&theme, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch theme")
	}

	return api.Success(c, toThemeResponse(&theme))
}

// Upload handles POST /themes/upload — multipart zip upload.
func (h *ThemeHandler) Upload(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "FILE_REQUIRED", "A file upload is required in the 'file' form field")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "FILE_OPEN_FAILED", "Failed to open uploaded file")
	}
	defer file.Close()

	theme, err := h.mgmtSvc.InstallFromZip(file, fileHeader.Filename)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INSTALL_FAILED", "Failed to install theme from zip: "+err.Error())
	}

	return api.Created(c, toThemeResponse(theme))
}

// gitInstallRequest represents the JSON body for git-based theme installation.
type gitInstallRequest struct {
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
	GitToken  string `json:"git_token"`
}

// InstallFromGit handles POST /themes/git — clone from a git repository.
func (h *ThemeHandler) InstallFromGit(c *fiber.Ctx) error {
	var req gitInstallRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.GitURL == "" {
		return api.ValidationError(c, map[string]string{"git_url": "git_url is required"})
	}
	if req.GitBranch == "" {
		req.GitBranch = "main"
	}

	theme, err := h.mgmtSvc.InstallFromGit(req.GitURL, req.GitBranch, req.GitToken)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INSTALL_FAILED", "Failed to install theme from git: "+err.Error())
	}

	return api.Created(c, toThemeResponse(theme))
}

// Activate handles POST /themes/:id/activate.
func (h *ThemeHandler) Activate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	if err := h.mgmtSvc.Activate(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "ACTIVATE_FAILED", "Failed to activate theme")
	}

	return api.Success(c, fiber.Map{"message": "Theme activated"})
}

// Deactivate handles POST /themes/:id/deactivate.
func (h *ThemeHandler) Deactivate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	if err := h.mgmtSvc.Deactivate(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DEACTIVATE_FAILED", "Failed to deactivate theme")
	}

	return api.Success(c, fiber.Map{"message": "Theme deactivated"})
}

// Pull handles POST /themes/:id/pull — git pull for a git-sourced theme.
func (h *ThemeHandler) Pull(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	theme, err := h.mgmtSvc.PullUpdate(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		if strings.Contains(err.Error(), "PULL_NOT_GIT") {
			return api.Error(c, fiber.StatusBadRequest, "PULL_NOT_GIT", "Theme is not git-sourced")
		}
		return api.Error(c, fiber.StatusInternalServerError, "PULL_FAILED", "Failed to pull theme update")
	}

	return api.Success(c, toThemeResponse(theme))
}

// Delete handles DELETE /themes/:id.
func (h *ThemeHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	if err := h.mgmtSvc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		if strings.Contains(err.Error(), "CANNOT_DELETE_ACTIVE") {
			return api.Error(c, fiber.StatusConflict, "CANNOT_DELETE_ACTIVE", "Cannot delete the active theme")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete theme")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// gitConfigRequest represents the JSON body for updating git config on a theme.
type gitConfigRequest struct {
	GitURL    *string `json:"git_url"`
	GitBranch *string `json:"git_branch"`
	GitToken  *string `json:"git_token"`
}

// UpdateGitConfig handles POST /themes/:id/git-config — update git settings on an existing theme.
func (h *ThemeHandler) UpdateGitConfig(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Theme ID must be a valid integer")
	}

	var req gitConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	var theme models.Theme
	if err := h.db.First(&theme, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Theme not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch theme")
	}

	updates := map[string]interface{}{}
	if req.GitURL != nil {
		updates["git_url"] = *req.GitURL
	}
	if req.GitBranch != nil {
		updates["git_branch"] = *req.GitBranch
	}
	if req.GitToken != nil {
		updates["git_token"] = *req.GitToken
	}

	if len(updates) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	if err := h.db.Model(&theme).Updates(updates).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update git config")
	}

	// Re-fetch to get updated values.
	h.db.First(&theme, id)
	return api.Success(c, toThemeResponse(&theme))
}

// webhookPayload is a minimal representation of GitHub/GitLab webhook payloads.
type webhookPayload struct {
	Repository *struct {
		CloneURL string `json:"clone_url"` // GitHub
	} `json:"repository"`
	Project *struct {
		GitHTTPURL string `json:"git_http_url"` // GitLab
	} `json:"project"`
}

// webhookDeploy handles POST /api/v1/theme-deploy — public webhook for auto-deploy.
func (h *ThemeHandler) webhookDeploy(c *fiber.Ctx) error {
	// 1. Validate webhook secret.
	var setting models.SiteSetting
	err := h.db.Where("`key` = ?", "theme_webhook_secret").First(&setting).Error
	if err != nil || setting.Value == nil || *setting.Value == "" {
		return api.Error(c, fiber.StatusForbidden, "WEBHOOK_DISABLED", "Webhook secret is not configured")
	}

	secret := c.Get("X-Webhook-Secret")
	if secret == "" {
		secret = c.Query("secret")
	}
	if secret != *setting.Value {
		return api.Error(c, fiber.StatusForbidden, "INVALID_SECRET", "Invalid webhook secret")
	}

	// 2. Extract repo URL from payload or use theme_slug query param.
	themeSlug := c.Query("theme_slug")
	var repoURL string

	var payload webhookPayload
	if err := json.Unmarshal(c.Body(), &payload); err == nil {
		if payload.Repository != nil && payload.Repository.CloneURL != "" {
			repoURL = payload.Repository.CloneURL
		} else if payload.Project != nil && payload.Project.GitHTTPURL != "" {
			repoURL = payload.Project.GitHTTPURL
		}
	}

	// 3. Find the matching theme.
	var theme models.Theme
	found := false

	if themeSlug != "" {
		if err := h.db.Where("slug = ?", themeSlug).First(&theme).Error; err == nil {
			found = true
		}
	}

	if !found && repoURL != "" {
		// Normalize URL: strip trailing .git for comparison.
		normalized := strings.TrimSuffix(repoURL, ".git")
		var themes []models.Theme
		h.db.Where("git_url IS NOT NULL").Find(&themes)
		for _, t := range themes {
			if t.GitURL != nil {
				candidate := strings.TrimSuffix(*t.GitURL, ".git")
				if strings.EqualFold(candidate, normalized) {
					theme = t
					found = true
					break
				}
			}
		}
	}

	if !found {
		return api.Error(c, fiber.StatusNotFound, "THEME_NOT_FOUND", "No theme matches the webhook payload")
	}

	// 4. Pull update.
	updated, err := h.mgmtSvc.PullUpdate(theme.ID)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "PULL_FAILED", "Failed to pull theme update: "+err.Error())
	}

	return api.Success(c, toThemeResponse(updated))
}
