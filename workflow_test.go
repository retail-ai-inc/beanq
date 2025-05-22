package beanq

import (
	"testing"
)

func TestWorkflow_initSteps(t *testing.T) {
	w := &Workflow{
		progresses: []TransBranch{
			{TaskID: "c1", Status: StatusPrepared}, // Step0
			{TaskID: "a1", Status: StatusPrepared}, // Step1
			{TaskID: "c2", Status: StatusPrepared}, // Step2
			{TaskID: "a2", Status: StatusPrepared}, // Step3
		},
	}

	actions, compensates := w.initSteps()

	actionsList := actions()
	if len(actionsList) != 1 || actionsList[0].TaskID != "a1" {
		t.Errorf("Expected only a1 to be executable, got: %+v", actionsList)
	}

	actionsList[0].Status = StatusSucceed

	actionsList = actions()
	if len(actionsList) != 1 || actionsList[0].TaskID != "a2" {
		t.Errorf("Expected only a2 to be executable, got: %+v", actionsList)
	}

	actionsList[0].Status = StatusSucceed

	if len(actions()) != 0 {
		t.Errorf("Expected no actions left to run, got: %+v", actions())
	}

	w.progresses[3].Status = StatusAborting
	compensateList := compensates()
	if len(compensateList) != 1 || compensateList[0].TaskID != "c2" {
		t.Errorf("Expected c2 to be rollback target, got: %+v", compensateList[0].TaskID)
	}
}
