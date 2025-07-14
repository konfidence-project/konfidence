package ensurefinalizer_test

import (
	"context"
	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.tools.sap/konfidence/pkg/pipeline/mocks"
	"github.tools.sap/konfidence/pkg/steps/ensurefinalizer"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type TestObject struct {
	metav1.ObjectMeta
}

func (t *TestObject) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (t *TestObject) DeepCopyObject() runtime.Object {
	return &TestObject{
		ObjectMeta: *t.ObjectMeta.DeepCopy(),
	}
}

func (t *TestObject) GetDeletionTimestamp() *metav1.Time {
	return t.DeletionTimestamp
}

var _ = Describe("Step", func() {
	var (
		mockClient *mocks.MockClient
		ctx        context.Context
		step       *ensurefinalizer.Step[*TestObject]
		obj        *TestObject
	)

	BeforeEach(func() {
		mockClient = mocks.NewMockClient(gomock.NewController(GinkgoT()))
		ctx = context.TODO()
		step = ensurefinalizer.New[*TestObject]("test-finalizer")
		obj = &TestObject{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-object",
			},
		}
	})

	Context("when the object does not have the finalizer and is not being deleted", func() {
		It("adds the finalizer and updates the object", func() {
			mockClient.EXPECT().Update(ctx, obj).Return(nil)

			result, err := step.Run(ctx, mockClient, obj)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(controllerutil.ContainsFinalizer(obj, "test-finalizer")).To(BeTrue())
		})
	})

	Context("when the object is being deleted", func() {
		It("does not add the finalizer", func() {
			deletionTimestamp := metav1.Now()
			obj.DeletionTimestamp = &deletionTimestamp

			result, err := step.Run(ctx, mockClient, obj)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(controllerutil.ContainsFinalizer(obj, "test-finalizer")).To(BeFalse())
		})
	})

	Context("when the object already has the finalizer", func() {
		It("does not update the object", func() {
			controllerutil.AddFinalizer(obj, "test-finalizer")

			result, err := step.Run(ctx, mockClient, obj)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})

	Context("when updating the object fails", func() {
		It("returns an error", func() {
			mockClient.EXPECT().Update(ctx, obj).Return(errors.New("update failed"))

			result, err := step.Run(ctx, mockClient, obj)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update failed"))
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
})
