package pipeline_test

import (
	"context"
	"errors"
	"time"

	"github.com/konfidence-project/pkg/pipeline"
	"github.com/konfidence-project/pkg/pipeline/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testObj struct {
	mocks.MockObject
}

type TestStep[T pipeline.Object] struct {
}

func (s *TestStep[T]) Run(ctx context.Context, c client.Client, obj T) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

var _ = Describe("Pipeline", func() {
	var (
		ctrlr      *gomock.Controller
		mockClient *mocks.MockClient
		obj        testObj
		key        client.ObjectKey
	)

	BeforeEach(func() {
		ctrlr = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(ctrlr)
		obj = testObj{}
		key = client.ObjectKey{Name: "test", Namespace: "default"}
	})

	AfterEach(func() {
		ctrlr.Finish()
	})

	It("should create a new pipeline", func() {
		p, err := pipeline.New(&obj)
		Expect(p).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail to create a pipeline with nil object", func() {
		p, err := pipeline.New[client.Object](nil)
		Expect(p).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("object is nil"))
	})

	It("should add steps to the pipeline", func() {
		p, err := pipeline.New(&obj)
		Expect(err).NotTo(HaveOccurred())

		// Add a step using interface
		p.AddStep(&TestStep[*testObj]{})

		// Add a step using function
		p.AddStepFunc(func(ctx context.Context, c client.Client, obj *testObj) (ctrl.Result, error) {
			return ctrl.Result{}, nil
		})

		Expect(p.GetSteps()).To(HaveLen(2))
	})

	Describe("Run", func() {
		var (
			p   *pipeline.Pipeline[*testObj]
			err error
		)

		BeforeEach(func() {
			ctrlr = gomock.NewController(GinkgoT())
			mockClient = mocks.NewMockClient(ctrlr)

			p, err = pipeline.New(&obj)
			Expect(p).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			ctrlr.Finish()
		})

		It("should return error if client.Get fails", func() {
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("not found"))
			p.AddStepFunc(func(ctx context.Context, c client.Client, o *testObj) (ctrl.Result, error) {
				err := c.Get(ctx, key, o)
				if err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			})
			_, err := p.Run(context.Background(), mockClient, key)
			Expect(err).To(HaveOccurred())
		})

		It("should execute steps and return on requeue", func() {
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			p.AddStepFunc(func(ctx context.Context, c client.Client, o *testObj) (ctrl.Result, error) {
				return ctrl.Result{RequeueAfter: time.Second * 5}, nil
			})
			res, err := p.Run(context.Background(), mockClient, key)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).NotTo(BeNil())
		})

		It("should stop on ErrPipelineBreak", func() {
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			p.AddStepFunc(func(ctx context.Context, c client.Client, o *testObj) (ctrl.Result, error) {
				return ctrl.Result{}, pipeline.ErrPipelineBreak
			})
			res, err := p.Run(context.Background(), mockClient, key)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{}))
		})

		It("should return error from step", func() {
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			p.AddStepFunc(func(ctx context.Context, c client.Client, o *testObj) (ctrl.Result, error) {
				return ctrl.Result{}, errors.New("step error")
			})
			res, err := p.Run(context.Background(), mockClient, key)
			Expect(err).To(MatchError("step error"))
			Expect(res).To(Equal(ctrl.Result{}))
		})
	})
})
