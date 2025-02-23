package chrome

import (
	"chromium-websocket-proxy/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"net/http"
	"regexp"
)

type Meta struct {
	debugUrl          string
	browserID         uuid.UUID
	firstPageTargetID target.ID
}

func (crm *Chrome) fetchAndSetMeta() error {
	r, err := regexp.Compile("[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	if err != nil {
		return err
	}

	// get info from version endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/version", crm.port))
	if err != nil {
		return err
	}

	var result map[string]interface{}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// set debugUrl from response
	crm.meta.debugUrl = result["webSocketDebuggerUrl"].(string)

	// parse browser id from debugUrl
	browserID := r.FindString(crm.meta.debugUrl)
	if len(browserID) == 0 {
		return errors.New("unable to parse browser id")
	}

	crm.ea.ctx = context.WithValue(crm.ea.ctx, logger.BrowserIdTrackingKey, browserID)
	crm.meta.browserID = uuid.MustParse(browserID)

	// get first page target for tracking browser
	targets, err := chromedp.Targets(crm.ctx)
	if err != nil {
		return err
	}

	if len(targets) < 1 {
		return errors.New("no targets created")
	}
	crm.meta.firstPageTargetID = targets[len(targets)-1].TargetID
	return nil
}
