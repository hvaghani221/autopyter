package code

import (
	"bytes"
	"errors"
	"log"
	"sync"

	"github.com/hvaghani221/autopyter/internal/kernel"
)

type State struct {
	PreviousCode   bytes.Buffer
	CurrentState   []*ExecutionState
	mu             sync.RWMutex
	preloadKernels *kernel.PreloadedKernels
	// kernel         *kernel.Kernel
	lastId int64
}

type ExecutionState struct {
	Code        string                   `json:"code"`
	ID          int64                    `json:"id"`
	Result      *kernel.ResultMessage    `json:"result,omitempty"`
	Exception   *kernel.ExceptionMessage `json:"exception,omitempty"`
	Error       error                    `json:"error,omitempty"`
	Kernel      *kernel.Kernel           `json:"-"`
	waitChannel chan struct{}
}

func (es *ExecutionState) WaitForResult() bool {
	_, ok := <-es.waitChannel
	return ok
}

func NewState() *State {
	// k, err := kernel.CreateKernel()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	preloadedkernels, err := kernel.NewPreloaded()
	if err != nil {
		log.Fatal(err)
	}
	s := &State{
		PreviousCode:   bytes.Buffer{},
		CurrentState:   []*ExecutionState{},
		mu:             sync.RWMutex{},
		preloadKernels: preloadedkernels,
		// kernel:         k,
	}

	return s
}

func (s *State) Select(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id > len(s.CurrentState) {
		return errors.New("state index out of range")
	}
	if _, err := s.PreviousCode.WriteString(s.CurrentState[id].Code); err != nil {
		return err
	}
	s.CurrentState = s.CurrentState[:0]

	if err := s.preloadKernels.Reset(s.PreviousCode.String()); err != nil {
		return err
	}

	return nil
}

func (s *State) Execute(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// kernel := s.kernel
	kernel, err := s.preloadKernels.Get()
	if err != nil {
		return err
	}

	log.Println("executing code on kernel", kernel.ID)

	waitChan := make(chan struct{})
	state := &ExecutionState{
		Code:        code,
		ID:          s.lastId,
		Kernel:      kernel,
		waitChannel: waitChan,
	}
	s.lastId++
	go func() {
		res, exc, err := kernel.ExecuteCode(code)
		state.Result = res
		state.Exception = exc
		state.Error = err
		close(waitChan)
	}()

	s.CurrentState = append(s.CurrentState, state)
	return nil
}

func (s *State) ListStates() []ExecutionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]ExecutionState, 0, len(s.CurrentState))
	for _, state := range s.CurrentState {
		res = append(res, ExecutionState{
			Code: state.Code,
			ID:   state.ID,
		})
	}
	return res
}

func (s *State) Close() {
	s.preloadKernels.Close()
	for _, state := range s.CurrentState {
		state.Kernel.Close()
	}
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

			state.Kernel.Close()
			return true
		}
	}
	return false
}
