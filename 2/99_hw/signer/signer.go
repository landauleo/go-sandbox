package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// обеспечивает нам конвейерную обработку функций-воркеров, которые что-то делают
func ExecutePipeline(jobs ...job) {
	//можно было использовать и указатель, но так проще и указатель нужен только при передаче wg как аргумент
	// куда-то, а в текущем кейсе работает свойство замыкания - доступ ко всем переменным (по ссылке)
	wg := sync.WaitGroup{}
	in := make(chan interface{})

	for _, jobItem := range jobs {
		wg.Add(1)
		//чтобы передавать результаты выполнения текущей ф-ии в следующую, а не устраивать помойку
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

// это чтобы DataSignerMd5 не запускался параллельно, если объявить внутри SingleHash, то параллельные вызовы
// SingleHash всё сломают
var md5mutex sync.Mutex

// считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~), где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in chan interface{}, out chan interface{}) {
	wg := sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		casted, _ := data.(int)
		dataStr := strconv.Itoa(casted)

		//DataSignerMd5 может одновременно вызываться только 1 раз, считается 10 мс.
		//Если одновременно запустится несколько - будет перегрев на 1 сек -> не запускаем параллельно
		md5mutex.Lock()
		md5 := DataSignerMd5(dataStr)
		md5mutex.Unlock()

		// тут начинается распараллеливание
		calculateCrc32Result(wg, out, dataStr, md5)
	}
	wg.Wait()
}

func calculateCrc32Result(wg sync.WaitGroup, out chan interface{}, dataStr string, md5 string) {
	go func(data string, md5 string) {
		defer wg.Done()
		crc32Md5Result := DataSignerCrc32(md5)
		crc32Result := DataSignerCrc32(data)
		out <- crc32Result + "~" + crc32Md5Result
	}(dataStr, md5)
}

// считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки), где th=0..5 ( т.е. 6 хешей на
// каждое входящее значение ), потом берёт конкатенацию результатов в порядке расчета (0..5), где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in chan interface{}, out chan interface{}) {
	externalWg := sync.WaitGroup{}

	for data := range in {
		externalWg.Add(1)
		//передаем data как аргумент, чтобы "заморозить" её значение для каждой конкретной горутины
		calculateMultiHashSlice(externalWg, out, data)
	}
	externalWg.Wait()
}

func calculateMultiHashSlice(externalWg sync.WaitGroup, out chan interface{}, data interface{}) {
	go func(dataItem interface{}) {
		defer externalWg.Done()
		multiHashSlice := make([]string, 6) //Slice -> динамический массив, самый популярный для работы со списками инструмент
		internalWg := &sync.WaitGroup{}
		for i := range 6 {
			internalWg.Add(1)
			go func(index int) {
				defer internalWg.Done()
				casted, _ := dataItem.(string)
				arg := strconv.Itoa(index) + casted //нельзя(!!!) так просто взять и сложить всё сразу в строке ниже
				multiHashSlice[index] = DataSignerCrc32(arg)
			}(i)
		}
		internalWg.Wait()
		out <- strings.Join(multiHashSlice, "")
	}(data)
}

// получает все результаты, сортирует (https://golang.org/pkg/sort/), объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in chan interface{}, out chan interface{}) {
	var resultStrings []string
	for data := range in {
		casted, _ := data.(string) //если закастить не удалось, программа не падает, а записывает в переменную пустое значение
		resultStrings = append(resultStrings, casted)
	}

	sort.Strings(resultStrings)
	out <- strings.Join(resultStrings, "_")
}
