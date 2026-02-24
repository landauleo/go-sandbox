package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// обеспечивает нам конвейерную обработку функций-воркеров, которые что-то делают
func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{} //нужен именно указатель, чтобы не передалась копия счётчика (в Go по дефолту передается всё по значению)
	in := make(chan interface{})

	for _, jobItem := range jobs {
		wg.Add(1)
		//чтобы передавать результаты выполнения текущей ф-ии в следующую, а не устраивать помоймку
		out := make(chan interface{})

		// про анонимную функцию я не догадалась сама, это AI подсказал, я бы точно не дошла,
		//в моем изначальном варианте было тупо -> go jobItem(in, out)
		go func(worker job, input, output chan interface{}) {
			//для этих двух строк ниже анонимная ф-я и была нужна
			defer wg.Done()
			defer close(output)

			worker(input, output)
		}(jobItem, in, out)

		in = out //AAAAA!!! -> STDOUT одной программы передаётся как STDIN в другую программу
	}
	wg.Wait() //блокирует выполнение кода, пока счетчики не будет 0
}

// считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~), где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in chan interface{}, out chan interface{}) {
	for data := range in {
		res := DataSignerCrc32(strconv.Itoa(data.(int))) + "~" + DataSignerCrc32(DataSignerMd5(strconv.Itoa(data.(int))))
		out <- res
	}
}

// считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки), где th=0..5 ( т.е. 6 хешей на
// каждое входящее значение ), потом берёт конкатенацию результатов в порядке расчета (0..5), где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in chan interface{}, out chan interface{}) {
	for data := range in {
		for i := range 5 {
			arg := strconv.Itoa(i) + data.(string) //нельзя(!!!) так просто взять и сложить всё сразу в строке ниже
			res := DataSignerCrc32(arg)
			out <- res
		}
	}
}

// получает все результаты, сортирует (https://golang.org/pkg/sort/), объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in chan interface{}, out chan interface{}) {
	var resultStrings []string
	for data := range in {
		resultStrings = append(resultStrings, data.(string))
	}

	sort.Strings(resultStrings)
	out <- strings.Join(resultStrings, "_")
}
