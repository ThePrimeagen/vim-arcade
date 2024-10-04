package e2etests

import (
	"context"
	"testing"

	servermanagement "vim-arcade.theprimeagen.com/pkg/server-management"
)

func TestMatchMakingCreateServer(t *testing.T) {
    ctx := context.Background()
    state := createEnvironment(ctx, getDBPath("no_server"), servermanagement.ServerParams{
        MaxLoad: 0.9,
    })

}

//func TestMatchMakingCreateServer(t *testing.T) {
//    mock := mocks.NewGameServer(t)
//    port, err := dummy.GetFreePort()
//    require.NoError(t, err, "unable to get a free port")
//    factory := NewTestingClientFactory("", uint16(port))
//
//    // TODO
//    // I should probably make it so that match making receives an interface
//    // instead of a running the server itself....
//    // just a thought?
//    mm := matchmaking.NewMatchMakingServer(matchmaking.MatchMakingServerParams{
//        Port: port,
//        GameServer: mock,
//    })
//
//    ctx := context.Background()
//    go mm.Run(ctx)
//    mm.WaitForReady(ctx)
//
//    // i am a bit confused on this part
//    // what am i passing these arguments in for?
//    // that i don't understand... are they part of the expectations?
//    // i would assume anything with a context would be virtually untestable
//    mock.EXPECT().GetBestServer().Return("", servermanagement.NoBestServer).Once()
//    mock.EXPECT().CreateNewServer(ctx).Return("testing:42069", nil).Once()
//    mock.EXPECT().WaitForReady(ctx, "testing:42069").
//        RunAndReturn(func(ctx context.Context, hostAndPort string) error {
//            // just want to make sure this is going
//            return nil
//        }).Once()
//    mock.EXPECT().GetConnectionString("testing:42069").Return("testing:42069", nil).Once()
//
//    // ok i am going to want to create a singular client...
//    // i think..
//    // i am curious about the logging.
//    wait := sync.WaitGroup{}
//    wait.Add(1)
//    client := factory.New(t, &wait)
//    wait.Wait()
//    slog.Info("i am done here...")
//
//    require.Equal(t, "testing:42069", string(client.mmData))
//}


