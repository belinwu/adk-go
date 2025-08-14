package graph

import (
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/session"
)

type eventLogEntry struct {
	EventParentID string
	Events        []*session.Event

	NodeRun *Node
}

type EventLog struct {
	entries []*eventLogEntry
}

func NewEventLog() *EventLog {
	return &EventLog{}
}

func (log *EventLog) AppendRun(n *Node) {
	log.entries = append(log.entries, &eventLogEntry{
		NodeRun: n,
	})
}

func (log *EventLog) AppendEvents(id string, e []*session.Event) {
	log.entries = append(log.entries, &eventLogEntry{
		EventParentID: id,
		Events:        e,
	})
}

type Executor struct {
	eventlog *EventLog

	nodes          map[string]*Node
	requiredInputs map[string][]string         // by node id  -> inputs
	data           map[string][]*session.Event // by ID -> events
	errCh          chan error
}

func NewExecutor() *Executor {
	return &Executor{
		eventlog:       NewEventLog(),
		nodes:          make(map[string]*Node),
		requiredInputs: make(map[string][]string),
		data:           make(map[string][]*session.Event),

		errCh: make(chan error),
	}
}

type Node struct {
	id     string
	Run    func(input []*session.Event) (output []*session.Event, err error)
	Inputs []string
	Output string
}

func (e *Executor) Wait() {
	<-e.errCh
}

func (e *Executor) Put(id string, data []*session.Event) {
	e.data[id] = data
	e.eventlog.AppendEvents(id, data)
}

func (e *Executor) pickNextRunnable() *Node {
	for nodeID, inputs := range e.requiredInputs {
		found := true
		for _, input := range inputs {
			_, ok := e.data[input]
			found = found && ok
		}
		if found {
			return e.nodes[nodeID]
		}
	}
	return nil
}

func (e *Executor) Read(id string) []*session.Event {
	return e.data[id]
}

func (e *Executor) Spawn(n *Node) {
	n.id = uuid.NewString()
	e.nodes[n.id] = n
	e.requiredInputs[n.id] = n.Inputs
}

func (e *Executor) removeNode(nodeID string) {
	delete(e.nodes, nodeID)
	delete(e.requiredInputs, nodeID)
}

func (e *Executor) Start() error {
	// TODO(jbd): Allow concurrency.
	go func() {
		for {
			node := e.pickNextRunnable()
			if node != nil {
				if err := e.run(node); err != nil {
					e.errCh <- err
					return
				}
			}

			// TODO(jbd): Replace the busy loop with something
			// more efficient.
			time.Sleep(100 * time.Millisecond)
			if len(e.nodes) == 0 {
				close(e.errCh)
				return
			}
		}
	}()
	return nil
}

func (e *Executor) run(n *Node) error {
	log.Printf("Running %v with inputs=%v, outputs=%v", n.id, n.Inputs, n.Output)
	var inputs []*session.Event
	for _, input := range n.Inputs {
		inputs = append(inputs, e.data[input]...)
	}
	events, err := n.Run(inputs)
	if err != nil {
		return err
	}
	e.eventlog.AppendRun(n)
	e.removeNode(n.id)
	e.Put(n.Output, events)
	return nil
}

func (e *Executor) EventLog() *EventLog {
	return e.eventlog
}

type ReplayExecutor struct {
	executor *Executor
	eventlog *EventLog
}

func NewReplayExecutor(log *EventLog) *ReplayExecutor {
	return &ReplayExecutor{
		executor: NewExecutor(),
		eventlog: log,
	}
}

func (e *ReplayExecutor) Start() error {
	log.Printf("%v entries in the log", len(e.eventlog.entries))

	for _, entry := range e.eventlog.entries {
		if n := entry.NodeRun; n != nil {
			log.Printf("Previously ran %v with inputs=%v, outputs=%v", n.id, n.Inputs, n.Output)
		}

		if entry.EventParentID != "" {
			log.Printf("Restoring %v", entry.EventParentID)
			e.executor.Put(entry.EventParentID, entry.Events)
		}
	}
	return nil
}
