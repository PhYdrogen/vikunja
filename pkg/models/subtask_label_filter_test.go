// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"strconv"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubtaskLabelFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	testUser := &user.User{
		ID:       1,
		Username: "user1",
	}

	project := &Project{
		Title:       "Test Project for Subtask Label Filter",
		Description: "Project to test subtask_label filter",
		OwnerID:     testUser.ID,
	}
	err := project.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, project.ID)

	labelClient := &Label{
		Title:       "client",
		CreatedByID: testUser.ID,
	}
	err = labelClient.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, labelClient.ID)

	labelAtt := &Label{
		Title:       "att",
		CreatedByID: testUser.ID,
	}
	err = labelAtt.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, labelAtt.ID)

	labelDone := &Label{
		Title:       "done",
		CreatedByID: testUser.ID,
	}
	err = labelDone.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, labelDone.ID)

	parentTask := &Task{
		Title:       "Parent Task - Client",
		Description: "This is the parent task with client label",
		ProjectID:   project.ID,
		CreatedByID: testUser.ID,
	}
	err = parentTask.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, parentTask.ID)

	labelTask := &LabelTask{
		LabelID: labelClient.ID,
		TaskID:  parentTask.ID,
	}
	_, err = s.Insert(labelTask)
	require.NoError(t, err)

	childTaskAtt := &Task{
		Title:       "Child Task - Att",
		Description: "This is a child task with att label",
		ProjectID:   project.ID,
		CreatedByID: testUser.ID,
	}
	err = childTaskAtt.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, childTaskAtt.ID)

	labelTaskAtt := &LabelTask{
		LabelID: labelAtt.ID,
		TaskID:  childTaskAtt.ID,
	}
	_, err = s.Insert(labelTaskAtt)
	require.NoError(t, err)

	childTaskDone := &Task{
		Title:       "Child Task - Done",
		Description: "This is a child task with done label",
		ProjectID:   project.ID,
		CreatedByID: testUser.ID,
	}
	err = childTaskDone.Create(s, testUser)
	require.NoError(t, err)
	require.NotZero(t, childTaskDone.ID)

	labelTaskDone := &LabelTask{
		LabelID: labelDone.ID,
		TaskID:  childTaskDone.ID,
	}
	_, err = s.Insert(labelTaskDone)
	require.NoError(t, err)

	relation1 := &TaskRelation{
		TaskID:       parentTask.ID,
		OtherTaskID:  childTaskAtt.ID,
		RelationKind: RelationKindSubtask,
		CreatedByID:  testUser.ID,
	}
	err = relation1.Create(s, testUser)
	require.NoError(t, err)

	// relation2 := &TaskRelation{
	// 	TaskID:       childTaskAtt.ID,
	// 	OtherTaskID:  parentTask.ID,
	// 	RelationKind: RelationKindParenttask,
	// 	CreatedByID:  testUser.ID,
	// }
	// err = relation2.Create(s, testUser)
	// require.NoError(t, err)

	relation3 := &TaskRelation{
		TaskID:       parentTask.ID,
		OtherTaskID:  childTaskDone.ID,
		RelationKind: RelationKindSubtask,
		CreatedByID:  testUser.ID,
	}
	err = relation3.Create(s, testUser)
	require.NoError(t, err)

	// relation4 := &TaskRelation{
	// 	TaskID:       childTaskDone.ID,
	// 	OtherTaskID:  parentTask.ID,
	// 	RelationKind: RelationKindParenttask,
	// 	CreatedByID:  testUser.ID,
	// }
	// err = relation4.Create(s, testUser)
	// require.NoError(t, err)

	err = s.Commit()
	require.NoError(t, err)

	t.Run("filter with labels = client && subtask_label = att", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		taskCollection := &TaskCollection{
			Filter: "labels = " + strconv.FormatInt(labelClient.ID, 10) + " && subtask_label = att",
		}

		res, _, _, err := taskCollection.ReadAll(s, testUser, "", 1, 50)
		require.NoError(t, err)

		tasks, ok := res.([]*Task)
		require.True(t, ok, "Result should be of type []*Task")

		// The filter should return only the parent task (that has label "client" AND has a subtask with label "att")
		// It should NOT return the subtasks themselves in the main results
		assert.Len(t, tasks, 1, "Expected 1 task to be returned (the parent)")

		// Verify it's the parent task
		assert.Equal(t, parentTask.ID, tasks[0].ID, "The returned task should be the parent task")

		// Verify the parent has the client label
		parentLabelIDs := make([]int64, len(tasks[0].Labels))
		for i, label := range tasks[0].Labels {
			parentLabelIDs[i] = label.ID
		}
		assert.Contains(t, parentLabelIDs, labelClient.ID, "Parent task should have 'client' label")

		// Verify that the parent has subtasks related
		// IMPORTANT: When subtask_label filter is applied, only matching subtasks should be in RelatedTasks
		subtasks, hasSubtasks := tasks[0].RelatedTasks[RelationKindSubtask]
		assert.True(t, hasSubtasks, "Parent should have subtasks")

		// THIS IS THE KEY TEST: Only 1 subtask should be returned (the one with "att" label)
		// The subtask with "done" label should NOT be in the results
		assert.Len(t, subtasks, 1, "Should have exactly 1 subtask (filtered by subtask_label = att)")

		// Verify it's the correct subtask (the one with "att" label)
		assert.Equal(t, childTaskAtt.ID, subtasks[0].ID, "The only subtask should be the one with 'att' label")
		assert.Equal(t, "Child Task - Att", subtasks[0].Title, "Should be the 'att' labeled task")

		// Verify the other subtask (with "done" label) is NOT in the results
		for _, subtask := range subtasks {
			assert.NotEqual(t, childTaskDone.ID, subtask.ID, "The subtask with 'done' label should NOT be in results")
		}
	})

	t.Run("verify parent has client label", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		task := &Task{ID: parentTask.ID}
		err := task.ReadOne(s, testUser)
		require.NoError(t, err)

		labelIDs := make([]int64, len(task.Labels))
		for i, label := range task.Labels {
			labelIDs[i] = label.ID
		}

		assert.Contains(t, labelIDs, labelClient.ID, "Parent task should have 'client' label")
	})

	t.Run("verify child tasks have correct labels", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		// Check child with "att" label
		taskAtt := &Task{ID: childTaskAtt.ID}
		err := taskAtt.ReadOne(s, testUser)
		require.NoError(t, err)

		labelIDsAtt := make([]int64, len(taskAtt.Labels))
		for i, label := range taskAtt.Labels {
			labelIDsAtt[i] = label.ID
		}

		assert.Contains(t, labelIDsAtt, labelAtt.ID, "Child task should have 'att' label")

		// Check child with "done" label
		taskDone := &Task{ID: childTaskDone.ID}
		err = taskDone.ReadOne(s, testUser)
		require.NoError(t, err)

		labelIDsDone := make([]int64, len(taskDone.Labels))
		for i, label := range taskDone.Labels {
			labelIDsDone[i] = label.ID
		}

		assert.Contains(t, labelIDsDone, labelDone.ID, "Child task should have 'done' label")
	})

	t.Run("verify parent-child relationships", func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()

		// Check parent task has subtasks
		parentTaskWithRelations := &Task{ID: parentTask.ID}
		err := parentTaskWithRelations.ReadOne(s, testUser)
		require.NoError(t, err)

		subtasks, ok := parentTaskWithRelations.RelatedTasks[RelationKindSubtask]
		assert.True(t, ok, "Parent should have subtasks")
		assert.Len(t, subtasks, 2, "Parent should have 2 subtasks")

		subtaskIDs := make([]int64, len(subtasks))
		for i, subtask := range subtasks {
			subtaskIDs[i] = subtask.ID
		}

		assert.Contains(t, subtaskIDs, childTaskAtt.ID, "Parent should have child with 'att' as subtask")
		assert.Contains(t, subtaskIDs, childTaskDone.ID, "Parent should have child with 'done' as subtask")
	})
}
