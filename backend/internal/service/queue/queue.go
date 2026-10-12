// Package queue 为 RabbitMQ 可选项提供无需外部服务的进程内队列替代实现。
package queue

import (
	"context"
	"errors"
	"sync"
)

type Job struct {
	Name string
	Data any
}

type Handler func(context.Context, Job)

type Queue struct {
	jobs   chan Job
	handle Handler
	closed chan struct{}
	once   sync.Once
}

var defaultQueue struct {
	sync.RWMutex
	queue *Queue
}

// New 创建有界队列；队列满时 Publish 返回错误，调用方可据此重试或降级。
func New(size int, handle Handler) *Queue {
	if size < 1 {
		size = 1
	}
	return &Queue{jobs: make(chan Job, size), handle: handle, closed: make(chan struct{})}
}

// StartDefault 初始化进程级队列，业务服务可通过 PublishDefault 投递任务。
func StartDefault(ctx context.Context, size int, handle Handler) {
	defaultQueue.Lock()
	defer defaultQueue.Unlock()
	if defaultQueue.queue != nil {
		return
	}
	defaultQueue.queue = New(size, handle)
	defaultQueue.queue.Start(ctx)
}

// PublishDefault 向启动时创建的进程级队列投递任务。
func PublishDefault(job Job) error {
	defaultQueue.RLock()
	current := defaultQueue.queue
	defaultQueue.RUnlock()
	if current == nil {
		return errors.New("queue is not started")
	}
	return current.Publish(job)
}

// StopDefault 停止进程级队列。
func StopDefault() {
	defaultQueue.Lock()
	defer defaultQueue.Unlock()
	if defaultQueue.queue != nil {
		defaultQueue.queue.Close()
		defaultQueue.queue = nil
	}
}

// Start 启动唯一消费者。进程内队列不持久化，重启后未处理任务会丢失。
func (q *Queue) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-q.closed:
				return
			case job := <-q.jobs:
				if q.handle != nil {
					q.handle(ctx, job)
				}
			}
		}
	}()
}

// Publish 向本地队列提交任务，不会在队列满时静默丢弃。
func (q *Queue) Publish(job Job) error {
	select {
	case <-q.closed:
		return errors.New("queue is closed")
	default:
	}
	select {
	case q.jobs <- job:
		return nil
	default:
		return errors.New("queue is full")
	}
}

// Close 停止消费者。
func (q *Queue) Close() {
	q.once.Do(func() { close(q.closed) })
}
