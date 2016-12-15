// +-------------------------------------------------------------------------
// | Copyright (C) 2016 Yunify, Inc.
// +-------------------------------------------------------------------------
// | Licensed under the Apache License, Version 2.0 (the "License");
// | you may not use this work except in compliance with the License.
// | You may obtain a copy of the License in the LICENSE file, or at:
// |
// | http://www.apache.org/licenses/LICENSE-2.0
// |
// | Unless required by applicable law or agreed to in writing, software
// | distributed under the License is distributed on an "AS IS" BASIS,
// | WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// | See the License for the specific language governing permissions and
// | limitations under the License.
// +-------------------------------------------------------------------------

package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/yunify/qingcloud-sdk-go/request/errs"
	"github.com/yunify/qingcloud-sdk-go/service"
)

// QingCloudServiceFeatureContext provides feature context for QingCloudService.
func QingCloudServiceFeatureContext(s *godog.Suite) {
	s.Step(`^initialize QingCloud service$`, initializeQingCloudService)
	s.Step(`^the QingCloud service is initialized$`, theQingCloudServiceIsInitialized)

	s.Step(`^initialize instance service$`, initializeInstanceService)
	s.Step(`^the instance service is initialized$`, theInstanceServiceIsInitialized)

	s.Step(`^initialize job service$`, initializeJobService)
	s.Step(`^the job service is initialized$`, theJobServiceIsInitialized)

	s.Step(`^describe zones$`, describeZones)
	s.Step(`^describe zones should get (\d+) zone at least$`, describeZonesShouldGetZoneAtLeast)
	s.Step(`^describe zones should have the zone I\'m using$`, describeZonesShouldHaveTheZoneIamUsing)

	s.Step(`^instance configuration:$`, instanceConfiguration)
	s.Step(`^run instances$`, runInstances)
	s.Step(`^run instances should get a job ID$`, runInstancesShouldGetAJobID)
	s.Step(`^run instances will be finished$`, runInstancesWillBeFinished)

	s.Step(`^terminate instances$`, terminateInstances)
	s.Step(`^terminate instances should get a job ID$`, terminateInstancesShouldGetAJobID)
	s.Step(`^terminate instances will be finished$`, terminateInstancesWillBeFinished)

	s.Step(`^describe jobs$`, describeJobs)
	s.Step(`^describe jobs should get (\d+) job at least$`, describeJobsShouldGetJobAtLeast)
	s.Step(`^describe jobs should have the jobs I just created$`, describeJobsShouldHaveTheJobsIJustCreated)
}

// --------------------------------------------------------------------------

var instanceService *service.InstanceService
var jobService *service.JobService

func initializeQingCloudService() error {
	return nil
}

func theQingCloudServiceIsInitialized() error {
	if qingcloudService == nil {
		return errors.New("QingCloud service is not initialized")
	}
	return nil
}

func initializeInstanceService() error {
	instanceService, err = qingcloudService.Instance(tc.Zone)
	return err
}

func theInstanceServiceIsInitialized() error {
	if instanceService == nil {
		return errors.New("Instance sub service is not initialized")
	}
	return nil
}

func initializeJobService() error {
	jobService, err = qingcloudService.Job(tc.Zone)
	return err
}

func theJobServiceIsInitialized() error {
	if jobService == nil {
		return errors.New("Job sub service is not initialized")
	}
	return nil
}

// --------------------------------------------------------------------------

var describeZonesOutput *service.DescribeZonesOutput

func describeZones() error {
	describeZonesOutput, err = qingcloudService.DescribeZones(nil)
	return err
}

func describeZonesShouldGetZoneAtLeast(count int) error {
	if len(describeZonesOutput.ZoneSet) > count {
		return nil
	}
	return fmt.Errorf("DescribeZones only got \"%d\" zone(s)", count)
}

func describeZonesShouldHaveTheZoneIamUsing() error {
	for _, zone := range describeZonesOutput.ZoneSet {
		if zone.ZoneID == tc.Zone {
			return nil
		}
	}

	return fmt.Errorf("DescribeZones dosen't have zone \"%s\"", tc.Zone)
}

// --------------------------------------------------------------------------

var runInstanceInput *service.RunInstancesInput
var runInstanceOutput *service.RunInstancesOutput

func instanceConfiguration(configuration *gherkin.DataTable) error {
	count, err := strconv.Atoi(configuration.Rows[1].Cells[2].Value)
	if err != nil {
		return err
	}

	runInstanceInput = &service.RunInstancesInput{
		ImageID:      configuration.Rows[1].Cells[0].Value,
		InstanceType: configuration.Rows[1].Cells[1].Value,
		Count:        count,
		LoginMode:    configuration.Rows[1].Cells[3].Value,
		LoginPasswd:  configuration.Rows[1].Cells[4].Value,
	}
	return nil
}

func runInstances() error {
	runInstanceOutput, err = instanceService.RunInstances(runInstanceInput)
	return err
}

func runInstancesShouldGetAJobID() error {
	if runInstanceOutput.JobID != "" {
		return nil
	}
	return errors.New("RunInstances don't get a job ID")
}

func runInstancesWillBeFinished() error {
	retries := 0
	for retries < tc.MaxRetries {
		describeJobOutput, err := jobService.DescribeJobs(
			&service.DescribeJobsInput{
				Jobs: []string{runInstanceOutput.JobID},
			},
		)
		if err != nil {
			return err
		}
		if describeJobOutput.JobSet[0].Status == "successful" {
			return nil
		}
		retries++
		time.Sleep(time.Second * time.Duration(tc.RetryWaitTime))
	}
	return nil
}

// --------------------------------------------------------------------------

var terminateInstanceOutput *service.TerminateInstancesOutput

func terminateInstances() error {
	retries := 0
	for retries < tc.MaxRetries {
		terminateInstanceOutput, err = instanceService.TerminateInstances(
			&service.TerminateInstancesInput{
				Instances: runInstanceOutput.Instances,
			},
		)
		if err != nil {
			fmt.Println(err)
			switch e := err.(type) {
			case *errs.QingCloudError:
				if e.RetCode != 1400 {
					return e
				}
			default:
				return err
			}
		} else {
			return nil
		}
		retries++
		time.Sleep(time.Second * time.Duration(tc.RetryWaitTime))
	}
	return nil
}

func terminateInstancesShouldGetAJobID() error {
	if terminateInstanceOutput.JobID != "" {
		return nil
	}
	return errors.New("TerminateInstances doesn't get a job ID")
}

func terminateInstancesWillBeFinished() error {
	retries := 0
	for retries < tc.MaxRetries {
		describeJobOutput, err := jobService.DescribeJobs(
			&service.DescribeJobsInput{
				Jobs: []string{terminateInstanceOutput.JobID},
			},
		)
		if err != nil {
			return err
		}
		if describeJobOutput.JobSet[0].Status == "successful" {
			return nil
		}
		retries++
		time.Sleep(time.Second * time.Duration(tc.RetryWaitTime))
	}
	return nil
}

// --------------------------------------------------------------------------

var describeJobOutput *service.DescribeJobsOutput

func describeJobs() error {
	describeJobOutput, err = jobService.DescribeJobs(nil)
	return err
}

func describeJobsShouldGetJobAtLeast(count int) error {
	if len(describeJobOutput.JobSet) > count {
		return nil
	}
	return fmt.Errorf("DescribeJobs doesn't get \"%d\" job(s)", count)
}

func describeJobsShouldHaveTheJobsIJustCreated() error {
	okCount := 0
	for _, job := range describeJobOutput.JobSet {
		if job.JobID == runInstanceOutput.JobID {
			okCount++
		}
		if job.JobID == terminateInstanceOutput.JobID {
			okCount++
		}
	}

	if okCount == 2 {
		return nil
	}
	return errors.New("DescribeJobs doesn't get the jobs I just created")
}
