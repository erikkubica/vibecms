package coreapi

import (
	"vibecms/internal/cms"
	"vibecms/internal/email"
	"vibecms/internal/events"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// coreImpl implements the CoreAPI interface, delegating to existing services.
// Remaining methods are provided in other impl_*.go files.
type coreImpl struct {
	db         *gorm.DB
	eventBus   *events.EventBus
	contentSvc *cms.ContentService
	menuSvc    *cms.MenuService
	mediaSvc   *cms.MediaService
	emailDisp  *email.Dispatcher
	app        *fiber.App
	filters    map[string][]filterEntry
}

type filterEntry struct {
	priority int
	handler  FilterHandler
}

// NewCoreImpl constructs a CoreAPI backed by the given services.
func NewCoreImpl(
	db *gorm.DB,
	eventBus *events.EventBus,
	contentSvc *cms.ContentService,
	menuSvc *cms.MenuService,
	mediaSvc *cms.MediaService,
	emailDisp *email.Dispatcher,
	app *fiber.App,
) CoreAPI {
	return &coreImpl{
		db:         db,
		eventBus:   eventBus,
		contentSvc: contentSvc,
		menuSvc:    menuSvc,
		mediaSvc:   mediaSvc,
		emailDisp:  emailDisp,
		app:        app,
		filters:    make(map[string][]filterEntry),
	}
}

// Compile-time interface satisfaction check.
var _ CoreAPI = (*coreImpl)(nil)
