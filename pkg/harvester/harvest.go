package harvester

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/becoded/go-harvest/harvest"
	"golang.org/x/oauth2"
)

type HarvestClient struct {
	*harvest.HarvestClient
}

type harvestTask struct {
	harvest.UserProjectAssignment
	timer *harvest.TimeEntry
}

type harvestTasks []*harvestTask

func (t harvestTasks) getByKey(key string) (*harvestTask, error) {
	for _, task := range t {
		if *task.Project.Code == key {
			return task, nil
		}
	}

	return nil, fmt.Errorf("no task with key %s found", key)
}

func (h *harvester) getNewHarvestClient() error {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: h.Settings.Harvest.Pass,
		},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	service := harvest.NewHarvestClient(tc)
	service.AccountId = h.Settings.Harvest.User

	client := &HarvestClient{
		HarvestClient: service,
	}

	h.harvestClient = client
	return nil
}

func (c *HarvestClient) getCompany() (*harvest.Company, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	company, _, err := c.Company.Get(ctx)
	if err != nil {
		return nil, err
	}

	return company, nil
}

func (c *HarvestClient) getUserProjects() (harvestTasks, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	asignments, _, err := c.Project.GetMyProjectAssignments(ctx, nil)
	if err != nil {
		return nil, err
	}

	tasks := make(harvestTasks, 0)
	for _, a := range asignments.UserAssignments {
		if *a.IsProjectManager == false {
			continue
		}

		task := &harvestTask{
			UserProjectAssignment: *a,
		}

		tasks = append(tasks, task)
	}

	timers, err := c.getTimers()
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		for _, timer := range timers {
			if *timer.Project.Id == *task.Project.Id {
				task.timer = timer
				break
			}
		}
	}
	return tasks, nil
}

func (c *HarvestClient) getTimers() ([]*harvest.TimeEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isRunning := true
	times, _, err := c.Timesheet.List(ctx, &harvest.TimeEntryListOptions{
		IsRunning: &isRunning,
	})
	if err != nil {
		return nil, err
	}

	return times.TimeEntries, nil
}

func (t *harvestTask) startTimer(client *HarvestClient) error {
	if t.timer != nil {
		return nil
	}

	// Get the coding task
	var codingTask *harvest.ProjectTaskAssignment
	for _, t := range *t.TaskAssignments {
		if *t.Task.Name == "Coding" {
			codingTask = &t
			break
		}
	}
	if codingTask == nil {
		return errors.New("unable to find coding task")
	}

	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()

	entry, _, err := client.Timesheet.CreateTimeEntryViaDuration(ctx, &harvest.TimeEntryCreateViaDuration{
		ProjectId: t.Project.Id,
		TaskId:    codingTask.Task.Id,
		SpentDate: &harvest.Date{Time: time.Now()},
	})
	if err != nil {
		return err
	}

	t.timer = entry
	return nil
}

func (t *harvestTask) stopTimer(client *HarvestClient) error {
	if t.timer == nil {
		return nil
	}

	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()

	_, _, err := client.Timesheet.StopTimeEntry(ctx, *t.timer.Id)

	t.timer = nil
	return err
}
