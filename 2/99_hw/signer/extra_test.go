package main

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

/*
	Тест, предложенный одним из учащихся курса, Ilya Boltnev

	В чем его преимущество по сравнению с TestPipeline?
	1. Он проверяет то, что все функции действительно выполнились
	2. Он дает представление о влиянии time.Sleep в одном из звеньев конвейера на время работы

	возможно кому-то будет легче с ним
	при правильной реализации ваш код конечно же должен его проходить
*/

func TestByIlia(t *testing.T) {

	var recieved uint32
	freeFlowJobs := []job{
		job(func(in, out chan interface{}) {
			out <- uint32(1)
			out <- uint32(3)
			out <- uint32(4)
		}),
		job(func(in, out chan interface{}) {
			for val := range in {
				out <- val.(uint32) * 3
				time.Sleep(time.Millisecond * 100)
			}
		}),
		job(func(in, out chan interface{}) {
			for val := range in {
				fmt.Println("collected", val)
				atomic.AddUint32(&recieved, val.(uint32))
			}
		}),
	}

	start := time.Now()

	ExecutePipeline(freeFlowJobs...)

	end := time.Since(start)

	expectedTime := time.Millisecond * 350

	if end > expectedTime {
		t.Errorf("execition too long\nGot: %s\nExpected: <%s", end, expectedTime)
	}

	if recieved != (1+3+4)*3 {
		t.Errorf("f3 have not collected inputs, recieved = %d", recieved)
	}
}

// Тест проверяет, что два параллельных вызова SingleHash (из разных горутин)
// не вызывают DataSignerMd5 одновременно (что привело бы к перегреву на 1+ сек).
func TestParallelSingleHashNoOverheat(t *testing.T) {
	origOverheatLock := OverheatLock
	origOverheatUnlock := OverheatUnlock
	defer func() {
		OverheatLock = origOverheatLock
		OverheatUnlock = origOverheatUnlock
	}()

	var overheatCount uint32

	OverheatLock = func() {
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 0, 1); !swapped {
				atomic.AddUint32(&overheatCount, 1)
				fmt.Println("OverheatLock happend")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}
	OverheatUnlock = func() {
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 1, 0); !swapped {
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}

	// Запускаем два SingleHash параллельно через fan-out:
	// один источник раздаёт данные в два канала, каждый из которых обрабатывается своим SingleHash
	in1 := make(chan interface{})
	in2 := make(chan interface{})
	out1 := make(chan interface{})
	out2 := make(chan interface{})

	start := time.Now()

	// Два параллельных SingleHash
	go func() {
		defer close(out1)
		SingleHash(in1, out1)
	}()
	go func() {
		defer close(out2)
		SingleHash(in2, out2)
	}()

	// Отправляем данные в оба канала одновременно
	go func() {
		for _, v := range []int{0, 1, 2} {
			in1 <- v
		}
		close(in1)
	}()
	go func() {
		for _, v := range []int{3, 4, 5} {
			in2 <- v
		}
		close(in2)
	}()

	// Читаем результаты
	for range out1 {
	}
	for range out2 {
	}

	elapsed := time.Since(start)

	if overheatCount > 0 {
		t.Errorf("DataSignerMd5 was called concurrently from parallel SingleHash instances, overheat happened %d time(s)", overheatCount)
	}

	// Без перегрева: 6 * 10ms md5 (последовательно) + ~2s crc32 ≈ ~2.1s
	// С перегревом: каждый конкурентный вызов добавляет +1с
	expectedTime := 3 * time.Second
	if elapsed > expectedTime {
		t.Errorf("execution too long: got %s, expected < %s (possible overheat)", elapsed, expectedTime)
	}
}
