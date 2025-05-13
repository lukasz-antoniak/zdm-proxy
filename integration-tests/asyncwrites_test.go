package integration_tests

import (
	"context"
	"github.com/datastax/go-cassandra-native-protocol/client"
	"github.com/datastax/go-cassandra-native-protocol/frame"
	"github.com/datastax/go-cassandra-native-protocol/message"
	"github.com/datastax/go-cassandra-native-protocol/primitive"
	"github.com/datastax/zdm-proxy/integration-tests/setup"
	"github.com/datastax/zdm-proxy/integration-tests/simulacron"
	"github.com/datastax/zdm-proxy/proxy/pkg/config"
	"github.com/gocql/gocql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAsyncWriteError(t *testing.T) {
	c := setup.NewTestConfig("", "")
	c.WriteMode = config.WriteModeDualAsyncOnSecondary
	testSetup, err := setup.NewSimulacronTestSetupWithConfig(t, c)
	require.Nil(t, err)
	defer testSetup.Cleanup()

	query := "INSERT INTO ks.table (name, id) VALUES (?, ?)"

	err = testSetup.Origin.Prime(simulacron.WhenQuery(
		query,
		simulacron.NewWhenQueryOptions()).
		ThenSuccess())
	require.Nil(t, err)

	err = testSetup.Target.Prime(simulacron.WhenQuery(
		query,
		simulacron.NewWhenQueryOptions()).
		ThenWriteTimeout(gocql.LocalOne, 0, 0, simulacron.Simple))
	require.Nil(t, err)

	client := client.NewCqlClient("127.0.0.1:14002", nil)
	cqlClientConn, err := client.ConnectAndInit(context.Background(), primitive.ProtocolVersion4, 0)
	require.Nil(t, err)
	defer cqlClientConn.Close()

	queryMsg := &message.Query{
		Query:   query,
		Options: nil,
	}

	rsp, err := cqlClientConn.SendAndReceive(frame.NewFrame(primitive.ProtocolVersion4, 0, queryMsg))
	require.Nil(t, err)
	require.Equal(t, primitive.OpCodeResult, rsp.Header.OpCode)
	_, ok := rsp.Body.Message.(*message.RowsResult)
	require.True(t, ok)
}
