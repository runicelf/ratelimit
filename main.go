package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func main() {
	rate := flag.Int("rate", 1, "максимальное количество запусков программы в секунду")
	inflight := flag.Int("inflight", 1, "максимальное количество параллельно запущенных команд")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("не передана команда")
		return
	}

	command := strings.Join(flag.Args(), " ")

	args := make(chan string, *inflight)
	go readStdIn(args, *rate)

	wg := sync.WaitGroup{}
	for i := 0; i < *inflight; i++ {
		wg.Add(1)
		go execCommandWorker(command, args, &wg)
	}

	wg.Wait()
}

func readStdIn(args chan string, rate int) {
	defer close(args)

	reader := bufio.NewReader(os.Stdin)
	var stdInBuffer []string
	for {
		text, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		checkErr(err)
		stdInBuffer = append(stdInBuffer, text[:len(text)-1])
	}

	rateCounter := rate
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for _, arg := range stdInBuffer {
		select {
		case <-ticker.C:
			rateCounter = rate
			rateCounter--
			args <- arg
		default:
			if rateCounter == 0 {
				<-ticker.C
				rateCounter = rate
			}

			rateCounter--
			args <- arg
		}
	}
}

func execCommandWorker(command string, args chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for arg := range args {
		wholeCommand := strings.Split(strings.Replace(command, "{}", arg, 1), " ")

		cmd := exec.Command(wholeCommand[0], wholeCommand[1:]...)
		bytes, err := cmd.Output()
		checkErr(err)

		fmt.Print(string(bytes))
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
