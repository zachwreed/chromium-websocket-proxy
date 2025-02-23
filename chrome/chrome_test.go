package chrome

import (
	"chromium-websocket-proxy/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ChromeTestSuite struct {
	suite.Suite
}

// run before each test
func (suite *ChromeTestSuite) SetupTest() {

}

func (suite *ChromeTestSuite) TestNewChrome() {
	port := 9000
	sessionId := uuid.New()
	eventReceiver := make(chan EventData)

	crm := NewChrome(CreateChromePayload{
		Port:      port,
		SessionId: sessionId,
		Options: config.ChromeConfigOptions{
			Profile: "",
			Hash:    "",
		},
		EventReceiver: eventReceiver,
	})

	assert.Equal(suite.T(), crm.SessionId(), sessionId)
	assert.Equal(suite.T(), crm.Port(), port)
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(ChromeTestSuite))
}
