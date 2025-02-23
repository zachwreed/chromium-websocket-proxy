package logger

import (
	"bufio"
	"chromium-websocket-proxy/config"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"sync"
	"testing"
)

const TestLogFile = "tmp/logger-logs.txt"

type ExpectedLog struct {
	Level     string `json:"level,omitempty"`
	Time      int    `json:"time,omitempty"`
	Message   string `json:"message,omitempty"`
	SessionId string `json:"sessionId,omitempty"`
	BrowserId string `json:"browserId,omitempty"`
}

type LoggerTestSuite struct {
	suite.Suite
}

// run before each test
func (suite *LoggerTestSuite) SetupTest() {
	once = sync.Once{}
	config.Once = sync.Once{}
}

// run after each test
func (suite *LoggerTestSuite) TearDownTest() {
	err := os.Remove(TestLogFile)
	if err != nil {
		fmt.Println(fmt.Sprintf("unable to delete file %s", TestLogFile))
	}
}

func getExpectedLogsFromLogFile() ([]ExpectedLog, error) {
	var els []ExpectedLog
	file, err := os.Open(TestLogFile)
	if err != nil {
		return els, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var v map[string]interface{}
		err = json.Unmarshal(scanner.Bytes(), &v)
		if err != nil {
			return els, err
		}
		jsonStr, err := json.Marshal(v)
		if err != nil {
			return els, err
		}
		var el ExpectedLog
		err = json.Unmarshal(jsonStr, &el)
		els = append(els, el)
	}
	return els, nil
}

func (suite *LoggerTestSuite) TestLogsRequiredKeys() {
	expectedMessage := "expectedMessage"

	suite.T().Setenv(config.LogFilePath, TestLogFile)
	suite.T().Setenv(config.LogLevel, "debug")

	logger := Get()

	logger.Info().Msg(expectedMessage)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 1)
	el := els[0]

	assert.Equal(suite.T(), "info", el.Level)
	assert.Equal(suite.T(), expectedMessage, el.Message)
	assert.Greater(suite.T(), el.Time, 0)
}

func (suite *LoggerTestSuite) TestDebugLevelDebugIsLogged() {
	expectedMessage := "expectedMessage"
	suite.T().Setenv(config.LogFilePath, TestLogFile)
	suite.T().Setenv(config.LogLevel, "debug")

	logger := Get()
	logger.Debug().Msg(expectedMessage)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 1)
	el := els[0]

	assert.Equal(suite.T(), "debug", el.Level)
	assert.Equal(suite.T(), expectedMessage, el.Message)
	assert.Greater(suite.T(), el.Time, 0)
}

func (suite *LoggerTestSuite) TestInfoLevelDebugIsNotLogged() {
	expectedMessage := "expectedMessage"
	suite.T().Setenv(config.LogFilePath, TestLogFile)
	suite.T().Setenv(config.LogLevel, "info")

	logger := Get()
	logger.Debug().Msg(expectedMessage)

	assert.NoFileExists(suite.T(), TestLogFile)
}

func (suite *LoggerTestSuite) TestWarnIsLogged() {
	expectedMessage := "expectedMessage"
	suite.T().Setenv(config.LogFilePath, TestLogFile)

	logger := Get()
	logger.Warn().Msg(expectedMessage)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 1)
	el := els[0]

	assert.Equal(suite.T(), "warn", el.Level)
	assert.Equal(suite.T(), expectedMessage, el.Message)
	assert.Greater(suite.T(), el.Time, 0)
}

func (suite *LoggerTestSuite) TestErrorIsLogged() {
	expectedMessage := "expectedMessage"
	suite.T().Setenv(config.LogFilePath, TestLogFile)

	logger := Get()
	logger.Error().Msg(expectedMessage)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 1)
	el := els[0]

	assert.Equal(suite.T(), "error", el.Level)
	assert.Equal(suite.T(), expectedMessage, el.Message)
	assert.Greater(suite.T(), el.Time, 0)
}

func (suite *LoggerTestSuite) TestChildContext() {
	expectedParentMessage := "expectedParentMessage"
	expectedChildMessage := "expectedChildMessage"
	expectedChildAdditionalVal := "expectedChildAdditionalVal"

	suite.T().Setenv(config.LogFilePath, TestLogFile)
	suite.T().Setenv(config.LogLevel, "debug")

	logger := Get()
	ctx := context.WithValue(context.Background(), SessionIdTrackingKey, expectedChildAdditionalVal)

	logger.Info().Ctx(ctx).Msg(expectedChildMessage)
	logger.Info().Msg(expectedParentMessage)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 2)
	childEl := els[0]
	parentEl := els[1]

	assert.Equal(suite.T(), "info", parentEl.Level)
	assert.Equal(suite.T(), "info", childEl.Level)
	assert.Equal(suite.T(), expectedParentMessage, parentEl.Message)
	assert.Equal(suite.T(), expectedChildMessage, childEl.Message)
	assert.Greater(suite.T(), parentEl.Time, 0)
	assert.Greater(suite.T(), childEl.Time, 0)

	assert.Empty(suite.T(), parentEl.SessionId)
	assert.Equal(suite.T(), childEl.SessionId, expectedChildAdditionalVal)
}

func (suite *LoggerTestSuite) TestMultipleContexts() {
	message1 := "message1"
	message2 := "message2"
	expectedSessionIdVal, _ := uuid.NewUUID()
	expectedBrowserIdVal, _ := uuid.NewUUID()

	suite.T().Setenv(config.LogFilePath, TestLogFile)
	suite.T().Setenv(config.LogLevel, "debug")

	logger := Get()
	sessionIdCtx := context.WithValue(context.Background(), SessionIdTrackingKey, expectedSessionIdVal.String())
	browserIdCtx := context.WithValue(context.Background(), BrowserIdTrackingKey, expectedBrowserIdVal)

	logger.Info().Ctx(sessionIdCtx).Msg(message1)
	logger.Info().Ctx(browserIdCtx).Msg(message2)

	els, err := getExpectedLogsFromLogFile()
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), els, 2)
	sessionIdEl := els[0]
	browserIdEl := els[1]

	assert.Equal(suite.T(), "info", sessionIdEl.Level)
	assert.Equal(suite.T(), "info", browserIdEl.Level)
	assert.Equal(suite.T(), message1, sessionIdEl.Message)
	assert.Equal(suite.T(), message2, browserIdEl.Message)
	assert.Greater(suite.T(), sessionIdEl.Time, 0)
	assert.Greater(suite.T(), browserIdEl.Time, 0)

	assert.Empty(suite.T(), sessionIdEl.BrowserId)
	assert.Equal(suite.T(), sessionIdEl.SessionId, expectedSessionIdVal.String())

	assert.Empty(suite.T(), browserIdEl.SessionId)
	assert.Equal(suite.T(), browserIdEl.BrowserId, expectedBrowserIdVal.String())
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}
