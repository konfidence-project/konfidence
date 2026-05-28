package activation

import (
	"context"
	"errors"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/activation/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("activation task execution tests", func() {
	var (
		ctx              context.Context
		mockCtrl         *gomock.Controller
		clientMock       *MockClient
		namespace        string
		registration     star.ActivationTaskRegistration
		vectorActivation *star.VectorActivation
		scheme           *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		namespace = "default"
		scheme = runtime.NewScheme()
		_ = star.AddToScheme(scheme)

		registration = star.ActivationTaskRegistration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "registration-1",
				Namespace: namespace,
			},
			Spec: star.ActivationTaskRegistrationSpec{
				Type: "test-type",
			},
		}

		vectorActivation = &star.VectorActivation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "activation-1",
				Namespace: namespace,
				UID:       "test-uid",
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "star.konfidence.io/v1alpha1",
				Kind:       "VectorActivation",
			},
			Spec: star.VectorActivationSpec{
				StageVersion: "stage-version-1",
			},
		}
	})

	Context("ListExecutionsForRegistration", func() {
		It("should return execution list for registration and activation", func() {
			expectedLabels := client.MatchingLabels{
				"registration": registration.Name,
				"activation":   vectorActivation.Name,
			}

			clientMock.EXPECT().List(ctx, gomock.Any(), client.InNamespace(namespace), expectedLabels).
				DoAndReturn(func(_ context.Context, list interface{}, _ ...interface{}) error {
					execList := list.(*star.ActivationTaskExecutionList)
					execList.Items = append(execList.Items, star.ActivationTaskExecution{
						ObjectMeta: metav1.ObjectMeta{Name: "execution-1"},
					})
					return nil
				})

			result, err := ListExecutionsForRegistration(ctx, clientMock, namespace, registration, vectorActivation)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Items).To(HaveLen(1))
		})

		It("should return error when ListExecutionsForRegistration fails", func() {
			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				Return(errors.New("list error"))

			result, err := ListExecutionsForRegistration(ctx, clientMock, namespace, registration, vectorActivation)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("CreateExecution", func() {
		It("should create execution successfully", func() {
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).
				DoAndReturn(func(_ context.Context, obj interface{}, _ ...interface{}) error {
					execution := obj.(*star.ActivationTaskExecution)
					Expect(execution.GenerateName).To(HavePrefix(registration.Name + "-"))
					Expect(execution.Spec.Type).To(Equal(registration.Spec.Type))
					Expect(execution.Spec.VectorActivation).To(Equal(vectorActivation.Name))
					Expect(execution.Labels["registration"]).To(Equal(registration.Name))
					Expect(execution.Labels["activation"]).To(Equal(vectorActivation.Name))
					Expect(execution.OwnerReferences).To(HaveLen(1))
					Expect(execution.OwnerReferences[0].Name).To(Equal(vectorActivation.Name))
					return nil
				})

			result, err := CreateExecution(ctx, clientMock, namespace, vectorActivation, registration)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})

		It("should return error when CreateExecution fails", func() {
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("create error"))

			result, err := CreateExecution(ctx, clientMock, namespace, vectorActivation, registration)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("EnsureExecutionsForRegistrations", func() {
		It("should create executions for all registrations without existing executions", func() {
			registrationList := &star.ActivationTaskRegistrationList{
				Items: []star.ActivationTaskRegistration{
					registration,
					{
						ObjectMeta: metav1.ObjectMeta{Name: "registration-2", Namespace: namespace},
						Spec:       star.ActivationTaskRegistrationSpec{Type: "test-type-2"},
					},
				},
			}

			// First registration has no executions
			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, list interface{}, _ ...interface{}) error {
					return nil
				})
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			// Second registration has no executions
			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, list interface{}, _ ...interface{}) error {
					return nil
				})
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			executionList, err := EnsureExecutionsForRegistrations(ctx, clientMock, namespace, registrationList, vectorActivation)
			Expect(err).ToNot(HaveOccurred())
			Expect(executionList.Items).ToNot(BeEmpty())
		})

		It("should skip creating execution if one already exists", func() {
			registrationList := &star.ActivationTaskRegistrationList{
				Items: []star.ActivationTaskRegistration{registration},
			}

			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, list interface{}, _ ...interface{}) error {
					execList := list.(*star.ActivationTaskExecutionList)
					execList.Items = append(execList.Items, star.ActivationTaskExecution{})
					return nil
				})

			_, err := EnsureExecutionsForRegistrations(ctx, clientMock, namespace, registrationList, vectorActivation)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error when EnsureExecutionsForRegistrations fails to list", func() {
			registrationList := &star.ActivationTaskRegistrationList{
				Items: []star.ActivationTaskRegistration{registration},
			}

			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				Return(errors.New("list error"))

			_, err := EnsureExecutionsForRegistrations(ctx, clientMock, namespace, registrationList, vectorActivation)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when EnsureExecutionsForRegistrations fails to create", func() {
			registrationList := &star.ActivationTaskRegistrationList{
				Items: []star.ActivationTaskRegistration{registration},
			}
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().List(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("create error"))

			_, err := EnsureExecutionsForRegistrations(ctx, clientMock, namespace, registrationList, vectorActivation)
			Expect(err).To(HaveOccurred())
		})
	})
})
