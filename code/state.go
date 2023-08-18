package code

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/hvaghani221/autopyter/internal/kernel"
)

type State struct {
	PreviousState  []*ExecutionState
	CurrentState   []*ExecutionState
	mu             sync.RWMutex
	preloadKernels *kernel.PreloadedKernels
	// kernel         *kernel.Kernel
	lastId int64
}

type ExecutionState struct {
	Code       string                    `json:"code"`
	ID         int64                     `json:"id"`
	Results    []kernel.ResultMessage    `json:"result,omitempty"`
	Exceptions []kernel.ExceptionMessage `json:"exception,omitempty"`
	Error      error                     `json:"error,omitempty"`
	KernelID   string
	// Kernel      *kernel.Kernel           `json:"-"`
	waitChannel chan struct{}
}

func (es *ExecutionState) GetID() int64 {
	return es.ID
}

func (es *ExecutionState) WaitForResult() bool {
	_, ok := <-es.waitChannel
	return ok
}

func NewState() *State {
	preloadedkernels, err := kernel.NewPreloaded()
	if err != nil {
		log.Fatal(err)
	}
	s := &State{
		PreviousState:  []*ExecutionState{},
		CurrentState:   []*ExecutionState{},
		mu:             sync.RWMutex{},
		preloadKernels: preloadedkernels,
	}

	return s
}

func (s *State) Select(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var executionState *ExecutionState

	for _, state := range s.CurrentState {
		if state.ID == id {
			executionState = state
			break
		}
	}
	if executionState == nil {
		return errors.New("state not found")
	}

	s.PreviousState = append(s.PreviousState, executionState)

	s.CurrentState = s.CurrentState[:0]

	if err := s.preloadKernels.Reset(s.getPreviousCode()); err != nil {
		return err
	}
	return nil
}

func (s *State) getPreviousCode() string {
	if len(s.PreviousState) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, state := range s.PreviousState {
		builder.WriteString(state.Code)
		builder.WriteString("\n")
	}
	return builder.String()
}

func (s *State) Execute(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// kernel := s.kernel
	kernel, err := s.preloadKernels.Get()
	if err != nil {
		return err
	}

	// log.Println("executing code on kernel", kernel.ID)

	waitChan := make(chan struct{})
	state := &ExecutionState{
		Code: code,
		ID:   s.lastId,
		// Kernel:      kernel,
		waitChannel: waitChan,
		KernelID:    kernel.ID,
	}
	s.lastId++
	go func() {
		res, exc, err := kernel.ExecuteCode(code)
		state.Results = res
		state.Exceptions = exc
		state.Error = err
		close(waitChan)
		kernel.Close()
	}()

	s.CurrentState = append(s.CurrentState, state)
	return nil
}

func (s *State) ListStates(current bool) []*ExecutionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*ExecutionState
	if current {
		list = s.CurrentState
	} else {
		list = s.PreviousState
	}

	res := make([]*ExecutionState, 0, len(list))
	for _, state := range list {
		res = append(res, &ExecutionState{
			Code:     state.Code,
			ID:       state.ID,
			KernelID: state.KernelID,
		})
	}
	return res
}

func (s *State) Close() {
	s.preloadKernels.Close()
}

func (s *State) GetState(id int64) *ExecutionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, state := range s.CurrentState {
		if state.ID == id {
			return state
		}
	}
	return nil
}

func (s *State) RemoveState(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, state := range s.CurrentState {
		if state.ID == id {
			s.CurrentState = append(s.CurrentState[:i], s.CurrentState[i+1:]...)
			return true
		}
	}
	return false
}

func (s *State) RemovePreviousState(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, state := range s.PreviousState {
		if state.ID == id {
			s.PreviousState = append(s.PreviousState[:i], s.PreviousState[i+1:]...)
			go func() {
				if err := s.preloadKernels.Reset(s.getPreviousCode()); err != nil {
					log.Println(err)
				}
			}()
			return true
		}
	}

	return false
}

func (s *State) ResetPreviousState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.PreviousState) == 0 && len(s.CurrentState) == 0 {
		return
	}

	s.PreviousState = s.PreviousState[:0]
	s.CurrentState = s.CurrentState[:0]
	go func() {
		if err := s.preloadKernels.Reset(s.getPreviousCode()); err != nil {
			log.Println(err)
		}
	}()
}
