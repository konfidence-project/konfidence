package setcondition_test

import (
	"context"
	"errors"

	"github.com/konfidence-project/pkg/pipeline/mocks"
	"github.com/konfidence-project/pkg/steps/setcondition"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Status TestStatus
}

type TestStatus struct {
	Conditions []*metav1.Condition
}

func (t *TestObject) GetConditions() []*metav1.Condition {
	return t.Status.Conditions
}

func (t *TestObject) SetConditions(conditions []*metav1.Condition) {
	t.Status.Conditions = conditions
}

func (t *TestObject) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (t *TestObject) DeepCopyObject() runtime.Object {
	cp := &TestObject{
		TypeMeta:   t.TypeMeta,
		ObjectMeta: *t.ObjectMeta.DeepCopy(),
	}

	if t.Status.Conditions != nil {
		cp.Status.Conditions = make([]*metav1.Condition, len(t.Status.Conditions))
		for i, cond := range t.Status.Conditions {
			cp.Status.Conditions[i] = cond.DeepCopy()
		}
	}

	return cp
}

var _ = Describe("SetCondition Step", func() {
	var (
		mockCtrl              *gomock.Controller
		mockClient            *mocks.MockClient
		mockSubResourceWriter *mocks.MockSubResourceWriter
		ctx                   context.Context

		obj  *TestObject
		step *setcondition.Step[*TestObject]
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockSubResourceWriter = mocks.NewMockSubResourceWriter(mockCtrl)
		ctx = context.Background()
		obj = &TestObject{
			ObjectMeta: metav1.ObjectMeta{Name: "test-object"},
			Status: TestStatus{
				Conditions: []*metav1.Condition{},
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("sets a default condition and updates status", func() {
		// Create a new step with default values
		step = setcondition.New[*TestObject](metav1.ConditionTrue)

		// Setup expectations
		mockClient.EXPECT().Status().Return(mockSubResourceWriter)
		mockSubResourceWriter.EXPECT().Update(ctx, obj).Return(nil)

		// Run the step
		result, err := step.Run(ctx, mockClient, obj)

		// Verify results
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{}))

		// Verify condition was set correctly
		Expect(obj.Status.Conditions).To(HaveLen(1))
		cond := obj.Status.Conditions[0]
		Expect(cond.Type).To(Equal("Ready"))
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal("ResourceAvailable"))
		Expect(cond.Message).To(Equal("Resource is now available"))
	})

	It("sets a custom condition with custom type, reason, and message", func() {
		// Create a new step with custom values
		step = setcondition.New[*TestObject](
			metav1.ConditionFalse,
			setcondition.WithConditionType[*TestObject]("CustomCondition"),
			setcondition.WithReason[*TestObject]("CustomReason"),
			setcondition.WithMessage[*TestObject]("Custom message"),
		)

		// Setup expectations
		mockClient.EXPECT().Status().Return(mockSubResourceWriter)
		mockSubResourceWriter.EXPECT().Update(ctx, obj).Return(nil)

		// Run the step
		result, err := step.Run(ctx, mockClient, obj)

		// Verify results
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{}))

		// Verify condition was set correctly
		Expect(obj.Status.Conditions).To(HaveLen(1))
		cond := obj.Status.Conditions[0]
		Expect(cond.Type).To(Equal("CustomCondition"))
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal("CustomReason"))
		Expect(cond.Message).To(Equal("Custom message"))
	})

	It("returns an error if status update fails", func() {
		// Create a new step
		step = setcondition.New[*TestObject](metav1.ConditionTrue)

		// Setup expectations with an error
		updateErr := errors.New("update failed")
		mockClient.EXPECT().Status().Return(mockSubResourceWriter)
		mockSubResourceWriter.EXPECT().Update(ctx, obj).Return(updateErr)

		// Run the step
		result, err := step.Run(ctx, mockClient, obj)

		// Verify error is returned
		Expect(err).To(MatchError(updateErr))
		Expect(result).To(Equal(ctrl.Result{}))

		// Verify condition was still set on the object
		Expect(obj.Status.Conditions).To(HaveLen(1))
	})
})
