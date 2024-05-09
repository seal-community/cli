package common

import (
	"fmt"
	"sync"
	"testing"
)

func TestChunks(t *testing.T) {
	input := []int{1, 2, 3}
	chunkSize := 1

	m := &sync.Mutex{}

	workerCalls := 0
	doneCalls := 0

	results := map[int]string{1: "111", 2: "222", 3: "333"}

	err := ConcurrentChunks(input, chunkSize,
		func(chunk []int, chunkIdx int) (string, error) {
			m.Lock() // since not safe
			workerCalls++
			m.Unlock()

			if len(chunk) != 1 {
				t.Fatalf("got different chunk length; got %d; exepected %d", len(chunk), len(input))
			}

			return results[chunkIdx], nil
		},
		func(result string, chunkIdx int) error {
			doneCalls++
			if results[chunkIdx] != result {
				t.Fatalf("idx %d got wrong result: %v; expected %v", chunkIdx, result, results[chunkIdx])
			}

			return nil
		},
	)

	if err != nil {
		t.Fatalf("got error from chunkify %v", err)
	}

	if workerCalls != 3 {
		t.Fatalf("got wrong worker calls from chunkify %d", workerCalls)
	}

	if doneCalls != 3 {
		t.Fatalf("got wrong worker calls from chunkify %d", doneCalls)
	}
}

func TestChunkOnly1(t *testing.T) {
	input := []int{1, 2, 3}
	chunkSize := 0 // should send as single chunk

	m := &sync.Mutex{}

	workerCalls := 0
	doneCalls := 0

	err := ConcurrentChunks(input, chunkSize,
		func(chunk []int, chunkIdx int) (string, error) {
			m.Lock() // since not safe
			workerCalls++
			m.Unlock()

			if len(chunk) != len(input) {
				t.Fatalf("got different chunk length; got %d; exepected %d", len(chunk), len(input))
			}

			return "", nil
		},
		func(result string, chunkIdx int) error {
			doneCalls++
			return nil
		},
	)

	if err != nil {
		t.Fatalf("got error from chunkify %v", err)
	}

	if workerCalls != 1 {
		t.Fatalf("got wrong worker calls from chunkify %d", workerCalls)
	}

	if doneCalls != 1 {
		t.Fatalf("got wrong worker calls from chunkify %d", doneCalls)
	}
}

func TestChunkLargerThanInput(t *testing.T) {
	input := []int{1, 2, 3}
	chunkSize := 6

	m := &sync.Mutex{}

	workerCalls := 0
	doneCalls := 0

	err := ConcurrentChunks(input, chunkSize,
		func(chunk []int, chunkIdx int) (string, error) {
			m.Lock() // since not safe
			workerCalls++
			m.Unlock()
			if len(chunk) != len(input) {
				t.Fatalf("got different chunk length; got %d; exepected %d", len(chunk), len(input))
			}
			return "", nil
		},
		func(result string, chunkIdx int) error {
			doneCalls++
			return nil
		},
	)

	if err != nil {
		t.Fatalf("got error from chunkify %v", err)
	}

	if workerCalls != 1 {
		t.Fatalf("got wrong worker calls from chunkify %d", workerCalls)
	}

	if doneCalls != 1 {
		t.Fatalf("got wrong worker calls from chunkify %d", doneCalls)
	}
}

func TestChunksErrorWorker(t *testing.T) {
	input := []int{1, 2, 3}
	chunkSize := 1

	doneCalls := 0

	results := map[int]string{1: "111", 2: "222", 3: "333"}
	returnedErr := fmt.Errorf("failed worker")
	err := ConcurrentChunks(input, chunkSize,
		func(chunk []int, chunkIdx int) (string, error) {
			if len(chunk) != 1 {
				t.Fatalf("got different chunk length; got %d; exepected %d", len(chunk), len(input))
			}

			return "", returnedErr
		},
		func(result string, chunkIdx int) error {
			doneCalls++
			if results[chunkIdx] != result {
				t.Fatalf("idx %d got wrong result: %v; expected %v", chunkIdx, result, results[chunkIdx])
			}

			return nil
		},
	)

	if err != returnedErr {
		t.Fatalf("did not get correct error from chunkify %v", err)
	}

	if doneCalls != 0 {
		t.Fatalf("got wrong worker calls from chunkify %d", doneCalls)
	}
}

func TestChunksErrorDone(t *testing.T) {
	input := []int{1, 2, 3}
	chunkSize := 1

	results := map[int]string{1: "111", 2: "222", 3: "333"}
	returnedErr := fmt.Errorf("failed done cb")

	err := ConcurrentChunks(input, chunkSize,
		func(chunk []int, chunkIdx int) (string, error) {
			return results[chunkIdx], nil
		},
		func(result string, chunkIdx int) error {
			return returnedErr
		},
	)

	if err != returnedErr {
		t.Fatalf("got error from chunkify %v", err)
	}
}
