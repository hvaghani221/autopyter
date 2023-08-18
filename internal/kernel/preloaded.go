package kernel

import (
	"log"
	"sync"
)

type PreloadedKernels struct {
	code    string
	kernels []*Kernel
	mu      sync.Mutex
}

func NewPreloaded() (*PreloadedKernels, error) {
	kernels := &PreloadedKernels{}
	if err := kernels.Reset(""); err != nil {
		return nil, err
	}
	return kernels, nil
}

func (k *PreloadedKernels) Get() (*Kernel, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if len(k.kernels) == 0 {
		return k.createPreloadedKernel()
	}
	kernel := k.kernels[len(k.kernels)-1]
	k.kernels = k.kernels[:len(k.kernels)-1]

	if len(k.kernels) == 0 {
		go func() {
			preloadedkernel, err := k.createPreloadedKernel()
			if err != nil {
				log.Println("Error creating preloaded kernel:", err)
				return
			}
			k.mu.Lock()
			defer k.mu.Unlock()
			k.kernels = append(k.kernels, preloadedkernel)
		}()
	}

	return kernel, nil
}

func (k *PreloadedKernels) Reset(code string) error {
	k.mu.Lock()
	k.code = code
	for _, kernel := range k.kernels {
		go kernel.Close()
	}
	k.kernels = k.kernels[:0]
	k.mu.Unlock()

	for i := 0; i < 4; i++ {
		go func() {
			preloadedkernel, err := k.createPreloadedKernel()
			if err != nil {
				log.Println("Error creating preloaded kernel:", err)
				return
			}
			k.mu.Lock()
			defer k.mu.Unlock()
			k.kernels = append(k.kernels, preloadedkernel)
		}()
	}
	return nil
}

func (k *PreloadedKernels) createPreloadedKernel() (*Kernel, error) {
	preloadedkernel, err := CreateKernel()
	if err != nil {
		return nil, err
	}
	if msg, exception, err := preloadedkernel.ExecuteCode(k.code); err != nil {
		log.Println(msg, exception, err)
		return nil, err
	}
	return preloadedkernel, nil
}

func (k *PreloadedKernels) Close() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, kernel := range k.kernels {
		kernel.Close()
	}
}
