package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"

	"github.com/google/go-dap"
)

// StdioReadWriteCloser is a ReadWriteCloser for reading from stdin and writing to stdout
// If Log is true, all incoming and outgoing data is logged
type StdioReadWriteCloser struct {
}

func (s StdioReadWriteCloser) Read(p []byte) (n int, err error) {
	return os.Stdin.Read(p)
}

func (s StdioReadWriteCloser) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

// Close closes stdin and stdout
func (s StdioReadWriteCloser) Close() error {
	err := os.Stdin.Close()
	if err != nil {
		return err
	}
	return os.Stdout.Close()
}

func StartSession(conn io.ReadWriteCloser) {
	debugSession := Session{
		rw:        bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		sendQueue: make(chan dap.Message),
		stopDebug: make(chan struct{}),
		debuglog:  true,
	}
	debugSession.Handler = NewHandler()
	debugSession.Handler.SetSession(&debugSession)

	go debugSession.sendFromQueue()

	for {
		err := debugSession.handleRequest()
		// TODO(polina): check for connection vs decoding error?
		if err != nil {
			if err == io.EOF {
				log.Println("No more data to read:", err)
				break
			}
			// There maybe more messages to process, but
			// we will start with the strict behavior of only accepting
			// expected inputs.
			log.Fatal("Server error: ", err)
		}
	}

	close(debugSession.stopDebug)
	debugSession.sendWg.Wait()
	close(debugSession.sendQueue)
	conn.Close()
}

func (ds *Session) send(message dap.Message) {
	ds.sendQueue <- message
}

func (ds *Session) sendFromQueue() {
	for message := range ds.sendQueue {
		dap.WriteProtocolMessage(ds.rw.Writer, message)
		log.Printf("Message sent\n\t%#v\n", message)
		ds.rw.Flush()
	}
}

func (ds *Session) handleRequest() error {
	log.Println("Reading request...")
	request, err := dap.ReadProtocolMessage(ds.rw.Reader)
	if err != nil {
		return err
	}
	log.Printf("Received request\n\t%#v\n", request)
	log.Printf("Breakpoints: %v", ds.breakPoints)
	ds.sendWg.Add(1)
	go func() {
		ds.dispatchRequest(request)
		ds.sendWg.Done()
	}()
	return nil
}

// dispatchRequest launches a new goroutine to process each request
// and send back events and responses.

func (ds *Session) dispatchRequest(request dap.Message) {
	switch request := request.(type) {
	case *dap.InitializeRequest:
		ds.Handler.OnInitializeRequest(request)
	case *dap.LaunchRequest:
		ds.Handler.OnLaunchRequest(request)
	case *dap.AttachRequest:
		ds.Handler.OnAttachRequest(request)
	case *dap.DisconnectRequest:
		ds.Handler.OnDisconnectRequest(request)
	case *dap.TerminateRequest:
		ds.Handler.OnTerminateRequest(request)
	case *dap.RestartRequest:
		ds.Handler.OnRestartRequest(request)
	case *dap.SetBreakpointsRequest:
		ds.Handler.OnSetBreakpointsRequest(request)
	case *dap.SetFunctionBreakpointsRequest:
		ds.Handler.OnSetFunctionBreakpointsRequest(request)
	case *dap.SetExceptionBreakpointsRequest:
		ds.Handler.OnSetExceptionBreakpointsRequest(request)
	case *dap.ConfigurationDoneRequest:
		ds.Handler.OnConfigurationDoneRequest(request)
	case *dap.ContinueRequest:
		ds.Handler.OnContinueRequest(request)
	case *dap.NextRequest:
		ds.Handler.OnNextRequest(request)
	case *dap.StepInRequest:
		ds.Handler.OnStepInRequest(request)
	case *dap.StepOutRequest:
		ds.Handler.OnStepOutRequest(request)
	case *dap.StepBackRequest:
		ds.Handler.OnStepBackRequest(request)
	case *dap.ReverseContinueRequest:
		ds.Handler.OnReverseContinueRequest(request)
	case *dap.RestartFrameRequest:
		ds.Handler.OnRestartFrameRequest(request)
	case *dap.GotoRequest:
		ds.Handler.OnGotoRequest(request)
	case *dap.PauseRequest:
		ds.Handler.OnPauseRequest(request)
	case *dap.StackTraceRequest:
		log.Printf("Trying to handle stack trace request")
		ds.Handler.OnStackTraceRequest(request)
	case *dap.ScopesRequest:
		ds.Handler.OnScopesRequest(request)
	case *dap.VariablesRequest:
		ds.Handler.OnVariablesRequest(request)
	case *dap.SetVariableRequest:
		ds.Handler.OnSetVariableRequest(request)
	case *dap.SetExpressionRequest:
		ds.Handler.OnSetExpressionRequest(request)
	case *dap.SourceRequest:
		ds.Handler.OnSourceRequest(request)
	case *dap.ThreadsRequest:
		ds.Handler.OnThreadsRequest(request)
	case *dap.TerminateThreadsRequest:
		ds.Handler.OnTerminateThreadsRequest(request)
	case *dap.EvaluateRequest:
		ds.Handler.OnEvaluateRequest(request)
	case *dap.StepInTargetsRequest:
		ds.Handler.OnStepInTargetsRequest(request)
	case *dap.GotoTargetsRequest:
		ds.Handler.OnGotoTargetsRequest(request)
	case *dap.CompletionsRequest:
		ds.Handler.OnCompletionsRequest(request)
	case *dap.ExceptionInfoRequest:
		ds.Handler.OnExceptionInfoRequest(request)
	case *dap.LoadedSourcesRequest:
		ds.Handler.OnLoadedSourcesRequest(request)
	case *dap.DataBreakpointInfoRequest:
		ds.Handler.OnDataBreakpointInfoRequest(request)
	case *dap.SetDataBreakpointsRequest:
		ds.Handler.OnSetDataBreakpointsRequest(request)
	case *dap.ReadMemoryRequest:
		ds.Handler.OnReadMemoryRequest(request)
	case *dap.DisassembleRequest:
		ds.Handler.OnDisassembleRequest(request)
	case *dap.CancelRequest:
		ds.Handler.OnCancelRequest(request)
	case *dap.BreakpointLocationsRequest:
		ds.Handler.OnBreakpointLocationsRequest(request)
	default:
		log.Fatalf("Unable to process %#v", request)
	}
}

// -----------------------------------------------------------------------
// Request Handlers
//
// Below is a dummy implementation of the request handlers.
// They take no action, but just return dummy responses.
// A real debug adaptor would call the debugger methods here
// and use their results to populate each response.

func newEvent(event string) *dap.Event {
	return &dap.Event{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "event",
		},
		Event: event,
	}
}

func newResponse(requestSeq int, command string) *dap.Response {
	return &dap.Response{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "response",
		},
		Command:    command,
		RequestSeq: requestSeq,
		Success:    true,
	}
}

func newErrorResponse(requestSeq int, command string, message string) *dap.ErrorResponse {
	er := &dap.ErrorResponse{}
	er.Response = *newResponse(requestSeq, command)
	er.Success = false
	er.Message = "unsupported"
	er.Body.Error.Format = message
	er.Body.Error.Id = 12345
	return er
}

type Session struct {
	// rw is used to read requests and write events/responses
	rw *bufio.ReadWriter

	// sendQueue is used to capture messages from multiple request
	// processing goroutines while writing them to the client connection
	// from a single goroutine via sendFromQueue. We must keep track of
	// the multiple channel senders with a wait group to make sure we do
	// not close this channel prematurely. Closing this channel will signal
	// the sendFromQueue goroutine that it can exit.
	sendQueue chan dap.Message
	sendWg    sync.WaitGroup

	// stopDebug is used to notify long-running handlers to stop processing.
	stopDebug chan struct{}

	Handler MonkeyHandler
	// bpSet is a counter of the remaining breakpoints that the debug
	// session is yet to stop at before the program terminates.
	bpSet       int
	bpSetMux    sync.Mutex
	breakPoints []dap.SourceBreakpoint
	source      dap.Source
	// rw is used to read requests and write events/responses

	debuglog bool
}

type Handler interface {
	SetSession(s *Session)
	OnInitializeRequest(arguments *dap.InitializeRequest)
	OnLaunchRequest(arguments *dap.LaunchRequest)
	OnDisconnectRequest(arguments *dap.DisconnectRequest)
	OnTerminateRequest(arguments *dap.TerminateRequest)
	OnSetBreakpointsRequest(arguments *dap.SetBreakpointsRequest)
	OnSetFunctionBreakpointsRequest(arguments *dap.SetFunctionBreakpointsRequest)
	OnSetExceptionBreakpointsRequest(arguments *dap.SetExceptionBreakpointsRequest)
	OnConfigurationDoneRequest(arguments *dap.ConfigurationDoneRequest)
	OnContinueRequest(arguments *dap.ContinueRequest)
	OnNextRequest(arguments *dap.NextRequest)
	OnStepInRequest(arguments *dap.StepInRequest)
	OnStepOutRequest(arguments *dap.StepOutRequest)
	OnStepBackRequest(arguments *dap.StepBackRequest)
	OnReverseContinueRequest(arguments *dap.ReverseContinueRequest)
	OnRestartFrameRequest(arguments *dap.RestartFrameRequest)
	OnGotoRequest(arguments *dap.GotoRequest)
	OnPauseRequest(arguments *dap.PauseRequest)
	OnStackTraceRequest(arguments *dap.StackTraceRequest)
	OnScopesRequest(arguments *dap.ScopesRequest)
	OnVariablesRequest(arguments *dap.VariablesRequest)
	OnSetVariableRequest(arguments *dap.SetVariableRequest)
	OnSetExpressionRequest(arguments *dap.SetExpressionRequest)
	OnSourceRequest(arguments *dap.SourceRequest)
	OnThreadsRequest(arguments *dap.ThreadsRequest)
	OnTerminateThreadsRequest(arguments *dap.TerminateThreadsRequest)
	OnEvaluateRequest(arguments *dap.EvaluateRequest)
	OnStepInTargetsRequest(arguments *dap.StepInTargetsRequest)
	OnGotoTargetsRequest(arguments *dap.GotoTargetsRequest)
	OnCompletionsRequest(arguments *dap.CompletionsRequest)
	OnExceptionInfoRequest(arguments *dap.ExceptionInfoRequest)
	OnLoadedSourcesRequest(arguments *dap.LoadedSourcesRequest)
	OnDataBreakpointInfoRequest(arguments *dap.DataBreakpointInfoRequest)
	OnSetDataBreakpointsRequest(arguments *dap.SetDataBreakpointsRequest)
	OnReadMemoryRequest(arguments *dap.ReadMemoryRequest)
	OnDisassembleRequest(arguments *dap.DisassembleRequest)
	OnCancelRequest(arguments *dap.CancelRequest)
	OnBreakpointLocationsRequest(arguments *dap.BreakpointLocationsRequest)
}
