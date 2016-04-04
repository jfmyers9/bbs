package main_test

import (
	"github.com/cloudfoundry-incubator/bbs/cmd/bbs/testrunner"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/bbs/models/test/model_helpers"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Task API", func() {
	var expectedTasks []*models.Task

	BeforeEach(func() {
		bbsRunner = testrunner.New(bbsBinPath, bbsArgs)
		bbsProcess = ginkgomon.Invoke(bbsRunner)
		expectedTasks = []*models.Task{model_helpers.NewValidTask("a-guid"), model_helpers.NewValidTask("b-guid")}
		expectedTasks[1].Domain = "b-domain"
		for _, t := range expectedTasks {
			err := client.DesireTask(t.TaskGuid, t.Domain, t.TaskDefinition)
			Expect(err).NotTo(HaveOccurred())
		}
		client.StartTask(expectedTasks[1].TaskGuid, "b-cell")
	})

	AfterEach(func() {
		ginkgomon.Kill(bbsProcess)
	})

	MatchTask := func(task *models.Task) types.GomegaMatcher {
		return SatisfyAll(
			WithTransform(func(t *models.Task) string {
				return t.TaskGuid
			}, Equal(task.TaskGuid)),
			WithTransform(func(t *models.Task) string {
				return t.Domain
			}, Equal(task.Domain)),
			WithTransform(func(t *models.Task) *models.TaskDefinition {
				return t.TaskDefinition
			}, Equal(task.TaskDefinition)),
		)
	}

	MatchTasks := func(tasks []*models.Task) types.GomegaMatcher {
		matchers := []types.GomegaMatcher{}
		matchers = append(matchers, HaveLen(len(tasks)))

		for _, task := range tasks {
			matchers = append(matchers, ContainElement(MatchTask(task)))
		}

		return SatisfyAll(matchers...)
	}

	Describe("Tasks", func() {
		It("has the correct number of responses", func() {
			actualTasks, err := client.Tasks()
			Expect(err).NotTo(HaveOccurred())
			Expect(actualTasks).To(MatchTasks(expectedTasks))
		})
	})

	Describe("TasksByDomain", func() {
		It("has the correct number of responses", func() {
			domain := expectedTasks[0].Domain
			actualTasks, err := client.TasksByDomain(domain)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualTasks).To(MatchTasks([]*models.Task{expectedTasks[0]}))
		})
	})

	Describe("TasksByCellID", func() {
		It("has the correct number of responses", func() {
			actualTasks, err := client.TasksByCellID("b-cell")
			Expect(err).NotTo(HaveOccurred())
			Expect(actualTasks).To(MatchTasks([]*models.Task{expectedTasks[1]}))
		})
	})

	Describe("TaskByGuid", func() {
		It("returns the task", func() {
			task, err := client.TaskByGuid(expectedTasks[0].TaskGuid)
			Expect(err).NotTo(HaveOccurred())
			Expect(task).To(MatchTask(expectedTasks[0]))
		})
	})

	Describe("DesireTask", func() {
		It("adds the desired task", func() {
			expectedTask := model_helpers.NewValidTask("task-1")
			err := client.DesireTask(expectedTask.TaskGuid, expectedTask.Domain, expectedTask.TaskDefinition)
			Expect(err).NotTo(HaveOccurred())

			task, err := client.TaskByGuid(expectedTask.TaskGuid)
			Expect(err).NotTo(HaveOccurred())
			Expect(task).To(MatchTask(expectedTask))
		})
	})

	Describe("Task Lifecycle", func() {
		var taskDef = model_helpers.NewValidTaskDefinition()
		const taskGuid = "task-1"
		const cellId = "cell-1"

		BeforeEach(func() {
			err := client.DesireTask(taskGuid, "test", taskDef)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("StartTask", func() {
			It("changes the task state from pending to running", func() {
				task, err := client.TaskByGuid(taskGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(task.State).To(Equal(models.Task_Pending))

				_, err = client.StartTask(taskGuid, cellId)
				Expect(err).NotTo(HaveOccurred())

				task, err = client.TaskByGuid(taskGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(task.State).To(Equal(models.Task_Running))
			})

			It("shouldStart is true", func() {
				shouldStart, err := client.StartTask(taskGuid, cellId)
				Expect(err).NotTo(HaveOccurred())
				Expect(shouldStart).To(BeTrue())
			})
		})

		Describe("CancelTask", func() {
			It("cancel the desired task", func() {
				err := client.CancelTask(taskGuid)
				Expect(err).NotTo(HaveOccurred())

				task, err := client.TaskByGuid(taskGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(task.FailureReason).To(Equal("task was cancelled"))
			})
		})

		Context("task has been started", func() {
			BeforeEach(func() {
				_, err := client.StartTask(taskGuid, cellId)
				Expect(err).NotTo(HaveOccurred())
			})

			Describe("FailTask", func() {
				It("marks the task completed and sets FailureReason", func() {
					err := client.FailTask(taskGuid, "some failure happened")
					Expect(err).NotTo(HaveOccurred())

					task, err := client.TaskByGuid(taskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(task.State).To(Equal(models.Task_Completed))
					Expect(task.FailureReason).To(Equal("some failure happened"))
				})
			})

			Describe("CompleteTask", func() {
				It("changes the task state from running to completed", func() {
					task, err := client.TaskByGuid(taskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(task.State).To(Equal(models.Task_Running))

					err = client.CompleteTask(taskGuid, cellId, false, "", "result")
					Expect(err).NotTo(HaveOccurred())

					task, err = client.TaskByGuid(taskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(task.State).To(Equal(models.Task_Completed))
				})
			})

			Context("task has been completed", func() {
				BeforeEach(func() {
					err := client.CompleteTask(taskGuid, cellId, false, "", "result")
					Expect(err).NotTo(HaveOccurred())
				})

				Describe("ResolvingTask", func() {
					It("changes the task state from completed to resolving", func() {
						err := client.ResolvingTask(taskGuid)
						Expect(err).NotTo(HaveOccurred())

						task, err := client.TaskByGuid(taskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(task.State).To(Equal(models.Task_Resolving))
					})
				})

				Context("task is resolving", func() {
					BeforeEach(func() {
						err := client.ResolvingTask(taskGuid)
						Expect(err).NotTo(HaveOccurred())
					})

					Describe("DeleteTask", func() {
						It("deletes the task", func() {
							err := client.DeleteTask(taskGuid)
							Expect(err).NotTo(HaveOccurred())

							_, err = client.TaskByGuid(taskGuid)
							Expect(err).To(Equal(models.ErrResourceNotFound))
						})
					})
				})
			})
		})
	})
})
