package utils

import (
	"golang-test/models"
	"sync"
)

type Pool struct {
	Concurrency int
	TaskChan    chan models.User
	ResultsChan chan map[string]interface{}
	Wg          sync.WaitGroup
	Mu          sync.Mutex
}

func (p *Pool) Worker() {
	defer p.Wg.Done() // Move the WaitGroup.Done call here to simplify
	
	for task := range p.TaskChan {
		user := task.Serialize()
		// No need for mutex when just sending to a channel
		p.ResultsChan <- *user
	}
}

func (p *Pool) Run(users []models.User) []map[string]interface{} {
	// Initialize channels
	p.TaskChan = make(chan models.User, len(users))
	p.ResultsChan = make(chan map[string]interface{}, len(users))
	
	// Start workers
	p.Wg.Add(p.Concurrency) // Add only for the number of workers, not tasks
	for i := 0; i < p.Concurrency; i++ {
		go p.Worker()
	}
	
	// Send tasks to workers
	for _, user := range users {
		p.TaskChan <- user
	}
	close(p.TaskChan) // Signal workers that no more tasks are coming
	
	// Create a separate goroutine to close the results channel after all workers finish
	go func() {
		p.Wg.Wait()
		close(p.ResultsChan)
	}()
	
	// Collect results
	var serializedUsers []map[string]interface{}
	for serializedUser := range p.ResultsChan {
		serializedUsers = append(serializedUsers, serializedUser)
	}
	
	return serializedUsers
}
