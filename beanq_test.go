//go:build integration || ci
// +build integration ci

package beanq

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type BeanqSuite struct {
	suite.Suite
	config *BeanqConfig
	client *Client
	ctx    context.Context

	delayChannel           string
	delayTopic             string
	delayPayload           []byte
	delayExpectMsg         string
	delayExpectExecuteTime time.Time
	delayAddTime           time.Time

	normalChannel   string
	normalTopic     string
	normalMsg       string
	normalPayload   []byte
	normalExpectMsg string
}

func (s *BeanqSuite) SetupSuite() {

	cfg, err := NewConfig("./", "json", "env.testing")
	s.Require().Nil(err, fmt.Sprintf("NewConfig error:%+v", err))
	s.config = cfg
	s.delayChannel = "delay-channel"
	s.delayTopic = "delay-topic"
	s.delayExpectMsg = "testing"

	s.normalChannel = "normal-channel"
	s.normalTopic = "normal-topic"
	s.normalMsg = "testing"
	s.normalPayload = []byte("testing")
	s.normalExpectMsg = "testing"

	s.ctx = context.Background()

	s.client = New(s.config)

	s.delayPayload = []byte(`testing`)

	now := time.Now()
	s.delayAddTime = now
	executeTime := now.Add(10 * time.Second)

	// publish payload into the delay channel
	err = s.client.BQ().WithContext(s.ctx).PublishAtTime(s.delayChannel, s.delayTopic, s.delayPayload, executeTime)
	s.Require().NoError(err, "Publish delay error")

	// publish payload into the normal channel
	err = s.client.BQ().WithContext(s.ctx).Publish(s.normalChannel, s.normalTopic, s.normalPayload)
	s.Require().NoError(err, "Publish normal error")

}

func (s *BeanqSuite) TestConsume() {

	proc, _ := os.FindProcess(os.Getpid())
	go func() {
		// after 30 seconds, send SIGINT signal to stop the program
		time.Sleep(30 * time.Second)
		s.T().Log("send SIGINT signal")
		_ = proc.Signal(syscall.SIGINT)
	}()

	_, err := s.client.BQ().WithContext(s.ctx).SubscribeToDelay(s.delayChannel, s.delayTopic, s.delayHandler())
	s.Require().NoError(err, "Subscribe delayHandler error")

	_, err = s.client.BQ().WithContext(s.ctx).Subscribe(s.normalChannel, s.normalTopic, s.normalHandler())
	s.Require().NoError(err, "Subscribe normalHandler error")

	s.client.Wait(s.ctx)
}

func (s *BeanqSuite) TearDownTest() {
	//delay check
	s.Require().Equal(s.delayExpectMsg, "testing", "expectMsg is equal to payload")
	s.Require().GreaterOrEqual(s.delayExpectExecuteTime, s.delayAddTime, "executeTime is greater than addTime")
	//normal check
	s.Require().Equal(s.normalMsg, s.normalExpectMsg, "payload match")
}

func (s *BeanqSuite) delayHandler() DefaultHandle {
	return DefaultHandle{
		DoHandle: func(ctx context.Context, message *Message) error {
			s.delayExpectMsg = message.Payload
			s.delayExpectExecuteTime = message.ExecuteTime
			s.T().Logf("Delay Queue AddTime:%+v,ExecuteTime:%+v \n", s.delayAddTime, message.ExecuteTime)
			return nil
		},
		DoCancel: nil,
		DoError:  nil,
	}
}

func (s *BeanqSuite) normalHandler() DefaultHandle {
	return DefaultHandle{
		DoHandle: func(ctx context.Context, message *Message) error {
			s.normalExpectMsg = message.Payload
			return nil
		},
		DoCancel: nil,
		DoError:  nil,
	}
}

func TestDelaySuite(t *testing.T) {
	suite.Run(t, new(BeanqSuite))
}
