package config

import (
	"chromium-websocket-proxy/test/testutils"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"strconv"
	"sync"
	"testing"
)

const TestLogFile = "tmp/config-logs.txt"

type ConfigTestSuite struct {
	suite.Suite
}

// run before each test
func (suite *ConfigTestSuite) SetupTest() {
	Once = sync.Once{}
}

func (suite *ConfigTestSuite) TestDefaultConfig() {
	c := Get()
	err := c.Validate()
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), c.GetChromePoolConfig().MaxBrowserInstances, MaxBrowserInstancesDefault)
	assert.Equal(suite.T(), c.GetChromePoolConfig().MinBrowserInstances, MinBrowserInstancesDefault)
	assert.Equal(suite.T(), c.GetChromePoolConfig().EnableAutoAssignDebugPort, EnableAutoAssignDebugPortDefault)
	assert.Equal(suite.T(), c.GetChromeConfig().Headless, ChromeHeadlessDefault)
	assert.Equal(suite.T(), c.GetServerConfig().Port, ServerPortDefault)
	assert.Equal(suite.T(), c.GetLoggerConfig().LogLevel, zerolog.InfoLevel)
}

func (suite *ConfigTestSuite) TestConfigWithOSEnv() {
	debugPorts := []int{9000, 9001}
	testutils.SetDebugPortsForConfig(suite.T(), ChromeDebugPorts, debugPorts)

	c := Get()
	err := c.Validate()
	assert.Nil(suite.T(), err)

	assert.Equal(suite.T(), c.GetChromePoolConfig().DebugPorts, debugPorts)
}

func (suite *ConfigTestSuite) TestConfigFailsValidationWithTooFewDebugPorts() {
	debugPorts := []int{9000, 9001}
	testutils.SetDebugPortsForConfig(suite.T(), ChromeDebugPorts, debugPorts)

	suite.T().Setenv(MaxBrowserInstances, strconv.FormatInt(10, 10))
	suite.T().Setenv(EnableAutoAssignDebugPort, strconv.FormatBool(false))

	c := Get()
	err := c.Validate()
	assert.ErrorContains(suite.T(), err, fmt.Sprintf("must contain %s number of ports", MaxBrowserInstances))
}

func (suite *ConfigTestSuite) TestConfigFailsValidationWithNoDebugPorts() {
	suite.T().Setenv(MaxBrowserInstances, strconv.FormatInt(10, 10))
	suite.T().Setenv(EnableAutoAssignDebugPort, strconv.FormatBool(false))

	c := Get()
	err := c.Validate()
	assert.ErrorContains(suite.T(), err, fmt.Sprintf("if %s is disabled, %s is required", EnableAutoAssignDebugPort, ChromeDebugPorts))
}

func (suite *ConfigTestSuite) TestLogLevelSet() {
	suite.T().Setenv(LogLevel, "debug")
	c := Get()
	err := c.Validate()
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), c.GetLoggerConfig().LogLevel, zerolog.DebugLevel)
}

func (suite *ConfigTestSuite) TestLogFilePath() {
	suite.T().Setenv(LogFilePath, TestLogFile)
	c := Get()
	err := c.Validate()
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), c.GetLoggerConfig().LogFilePath, TestLogFile)
}

func (suite *ConfigTestSuite) TestDefaultLogFilePath() {
	c := Get()
	err := c.Validate()
	assert.Nil(suite.T(), err)
	assert.Empty(suite.T(), c.GetLoggerConfig().LogFilePath)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
