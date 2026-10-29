package subagent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// JobState is a background job's lifecycle state.
type JobState string

const (
	JobQueued  JobState = "queued"
	JobRunning JobState = "running"
	JobDone    JobState = "done"
	JobFailed  JobState = "failed"
	JobKilled  JobState = "killed"
)

// Job is one background unit of work: a task spawn, or a single subagent run.
//
// Upstream: omp src/async/job-manager.ts AsyncJob.
type Job struct {
	ID    string
	Type  string
	Label string
	// OwnerID is the agent that spawned the job, empty for the session itself.
	OwnerID string

	State     JobState
	Queued    bool
	StartedAt time.Time
	EndedAt   time.Time
	// Error is the failure message for a failed or killed job.
	Error string

	// Result is the delivery payload: the subagent's final text.
	Result string
	// Delivered records that the parent consumed this job's result.
	Delivered bool

	cancel context.CancelFunc
}

// Elapsed is the job's wall-clock span.
func (j *Job) Elapsed(now time.Time) time.Duration {
	if j.StartedAt.IsZero() {
		return 0
	}
	end := j.EndedAt
	if end.IsZero() {
		end = now
	}
	if end.Before(j.StartedAt) {
		return 0
	}
	return end.Sub(j.StartedAt)
}

// Settlement is one completed job handed to a waiter.
type Settlement struct {
	Job    Job
	Result string
	Err    error
}

// ErrNoJobs reports that nothing is running or awaiting delivery.
var ErrNoJobs = errors.New("no background jobs to wait for")

// Jobs tracks background jobs for one session tree.
//
// Upstream: omp src/async/job-manager.ts AsyncJobManager. Delivery is
// consume-once: a waiter takes a settlement and the parent's next turn does
// not replay it.
type Jobs struct {
	mu       sync.Mutex
	jobs     map[string]*Job
	order    []string
	pending  []Settlement
	waiters  map[uint64]chan Settlement
	nextWait uint64
	nextID   uint64
	now      func() time.Time
	maxJobs  int
	onUpdate func(Job)
}

// NewJobs returns a job manager. maxJobs of 0 means unbounded.
func NewJobs(maxJobs int) *Jobs {
	return &Jobs{
		jobs:    map[string]*Job{},
		waiters: map[uint64]chan Settlement{},
		now:     time.Now,
		maxJobs: maxJobs,
	}
}

// SetClock replaces the time source, for tests.
func (j *Jobs) SetClock(now func() time.Time) {
	j.mu.Lock()
	j.now = now
	j.mu.Unlock()
}

// SetObserver installs a callback invoked after every job state change. It is
// called without the manager's lock held.
func (j *Jobs) SetObserver(fn func(Job)) {
	j.mu.Lock()
	j.onUpdate = fn
	j.mu.Unlock()
}

// Admit reserves capacity for one more running job. It reports false when the
// cap is reached, so a caller can fall back to a synchronous run instead of
// failing the call.
func (j *Jobs) Admit() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.maxJobs <= 0 {
		return true
	}
	running := 0
	for _, job := range j.jobs {
		if job.State == JobRunning {
			running++
		}
	}
	return running < j.maxJobs
}

// Register adds a job and returns it with a cancel handle bound to
// parentContext. The body runs on its own goroutine; the job settles when body
// returns.
func (j *Jobs) Register(parentContext context.Context, id, jobType, label, ownerID string, body func(context.Context) (string, error)) (*Job, error) {
	j.mu.Lock()
	if id == "" {
		j.nextID++
		id = fmt.Sprintf("%s-%d", jobType, j.nextID)
	}
	if _, exists := j.jobs[id]; exists {
		j.mu.Unlock()
		return nil, fmt.Errorf("subagent: job %q already registered", id)
	}
	ctx, cancel := context.WithCancel(parentContext)
	job := &Job{
		ID:      id,
		Type:    jobType,
		Label:   label,
		OwnerID: ownerID,
		State:   JobQueued,
		Queued:  true,
		cancel:  cancel,
	}
	j.jobs[id] = job
	j.order = append(j.order, id)
	snapshot := *job
	j.mu.Unlock()
	j.notify(snapshot)

	go func() {
		j.markRunning(id)
		result, err := body(ctx)
		j.settle(id, result, err)
	}()
	return job, nil
}

// markRunning promotes a queued job once its body starts.
func (j *Jobs) markRunning(id string) {
	j.mu.Lock()
	job, ok := j.jobs[id]
	if !ok {
		j.mu.Unlock()
		return
	}
	job.State = JobRunning
	job.Queued = false
	job.StartedAt = j.now()
	snapshot := *job
	j.mu.Unlock()
	j.notify(snapshot)
}

// settle finalizes a job, records its delivery, and wakes one waiter.
func (j *Jobs) settle(id, result string, err error) {
	j.mu.Lock()
	job, ok := j.jobs[id]
	if !ok {
		j.mu.Unlock()
		return
	}
	job.EndedAt = j.now()
	job.Result = result
	if job.StartedAt.IsZero() {
		job.StartedAt = job.EndedAt
	}
	switch {
	case err == nil:
		job.State = JobDone
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		job.State = JobKilled
		job.Error = err.Error()
	default:
		job.State = JobFailed
		job.Error = err.Error()
	}
	job.cancel = nil
	settlement := Settlement{Job: *job, Result: result, Err: err}
	snapshot := *job
	if waiter := j.popWaiterLocked(); waiter != nil {
		job.Delivered = true
		settlement.Job = *job
		waiter <- settlement
	} else {
		j.pending = append(j.pending, settlement)
	}
	j.mu.Unlock()
	j.notify(snapshot)
}

// popWaiterLocked removes and returns any waiting channel.
func (j *Jobs) popWaiterLocked() chan Settlement {
	if len(j.waiters) == 0 {
		return nil
	}
	var lowest uint64
	first := true
	for id := range j.waiters {
		if first || id < lowest {
			lowest = id
			first = false
		}
	}
	ch := j.waiters[lowest]
	delete(j.waiters, lowest)
	return ch
}

// notify calls the observer without holding the lock.
func (j *Jobs) notify(job Job) {
	j.mu.Lock()
	fn := j.onUpdate
	j.mu.Unlock()
	if fn != nil {
		fn(job)
	}
}

// Wait blocks until a job settles or ctx ends. Results are delivered once: a
// settled job consumed by one waiter is never returned again.
//
// Upstream: omp tools/wait (30 minute cap applied by the caller).
func (j *Jobs) Wait(ctx context.Context) (Settlement, error) {
	j.mu.Lock()
	if len(j.pending) > 0 {
		settlement := j.pending[0]
		j.pending = j.pending[1:]
		if job, ok := j.jobs[settlement.Job.ID]; ok {
			job.Delivered = true
		}
		j.mu.Unlock()
		return settlement, nil
	}
	var anyOutstanding bool
	for _, job := range j.jobs {
		if job.State == JobRunning || job.State == JobQueued || !job.Delivered {
			anyOutstanding = true
			break
		}
	}
	if !anyOutstanding {
		j.mu.Unlock()
		return Settlement{}, ErrNoJobs
	}
	id := j.nextWait
	j.nextWait++
	ch := make(chan Settlement, 1)
	j.waiters[id] = ch
	j.mu.Unlock()

	select {
	case settlement := <-ch:
		return settlement, nil
	case <-ctx.Done():
		j.mu.Lock()
		delete(j.waiters, id)
		j.mu.Unlock()
		return Settlement{}, ctx.Err()
	}
}

// Kill cancels one job.
func (j *Jobs) Kill(id string) bool {
	j.mu.Lock()
	job, ok := j.jobs[id]
	if !ok {
		j.mu.Unlock()
		return false
	}
	cancel := job.cancel
	j.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return true
}

// KillAll cancels every running job.
func (j *Jobs) KillAll() {
	j.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(j.jobs))
	for _, job := range j.jobs {
		if job.cancel != nil {
			cancels = append(cancels, job.cancel)
		}
	}
	j.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

// Snapshot returns a copy of one job.
func (j *Jobs) Snapshot(id string) (Job, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	job, ok := j.jobs[id]
	if !ok {
		return Job{}, false
	}
	return *job, true
}

// List returns job snapshots in registration order.
func (j *Jobs) List() []Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]Job, 0, len(j.order))
	for _, id := range j.order {
		if job, ok := j.jobs[id]; ok {
			out = append(out, *job)
		}
	}
	return out
}

// Active returns running and queued job snapshots, newest first.
func (j *Jobs) Active() []Job {
	all := j.List()
	out := make([]Job, 0, len(all))
	for _, job := range all {
		if job.State == JobRunning || job.State == JobQueued {
			out = append(out, job)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].StartedAt.After(out[b].StartedAt) })
	return out
}

// Forget drops settlement bookkeeping for one job, keeping its record.
func (j *Jobs) Forget(id string) {
	j.mu.Lock()
	kept := j.pending[:0]
	for _, settlement := range j.pending {
		if settlement.Job.ID != id {
			kept = append(kept, settlement)
		}
	}
	j.pending = kept
	j.mu.Unlock()
}
