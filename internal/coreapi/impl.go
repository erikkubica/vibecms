package coreapi

import (
	"vibecms/internal/cms"
	"vibecms/internal/email"
	"vibecms/internal/events"
	"vibecms/internal/secrets"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// coreImpl implements the CoreAPI interface, delegating to existing services.
// Remaining methods are provided in other impl_*.go files.
type coreImpl struct {
	db           *gorm.DB
	eventBus     *events.EventBus
	contentSvc   *cms.ContentService
	menuSvc      *cms.MenuService
	mediaSvc     *cms.MediaService
	nodeTypeSvc  *cms.NodeTypeService
	emailDisp    *email.Dispatcher
	app          *fiber.App
	secrets      *secrets.Service // may be nil (encryption disabled in dev)
	filters      map[string][]filterEntry
	nextFilterID uint64 // monotonically increasing — assigned to each filterEntry as a stable handle for Unsubscribe.
}

type filterEntry struct {
	id       uint64 // opaque registration handle — used by the returned UnsubscribeFunc to locate this entry.
	priority int
	handler  FilterHandler
}

// NewCoreImpl constructs a CoreAPI backed by the given services. Pass a
// non-nil *secrets.Service to enable transparent at-rest encryption for
// secret-shaped settings; nil leaves reads/writes plaintext (dev/test).
func NewCoreImpl(
	db *gorm.DB,
	eventBus *events.EventBus,
	contentSvc *cms.ContentService,
	menuSvc *cms.MenuService,
	mediaSvc *cms.MediaService,
	nodeTypeSvc *cms.NodeTypeService,
	emailDisp *email.Dispatcher,
	app *fiber.App,
	secretsSvc *secrets.Service,
) CoreAPI {
	return &coreImpl{
		db:          db,
		eventBus:    eventBus,
		contentSvc:  contentSvc,
		menuSvc:     menuSvc,
		mediaSvc:    mediaSvc,
		nodeTypeSvc: nodeTypeSvc,
		emailDisp:   emailDisp,
		app:         app,
		secrets:     secretsSvc,
		filters:     make(map[string][]filterEntry),
	}
}

// Compile-time interface satisfaction check.
var _ CoreAPI = (*coreImpl)(nil)
