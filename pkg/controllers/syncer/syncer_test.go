package syncer

import (
	"context"
	"testing"

	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestReconcile(t *testing.T) {

	type testCase struct {
		Name  string
		Focus bool

		// Syncer func(ctx *synccontext.RegisterContext) (Object, error)

		InitialPhysicalState []runtime.Object
		InitialVirtualState  []runtime.Object

		ExpectedPhysicalState map[schema.GroupVersionKind][]runtime.Object
		ExpectedVirtualState  map[schema.GroupVersionKind][]runtime.Object

		Compare generictesting.Compare

		shouldErr bool
		errMsg    string
	}

	for _, tc := range []testCase{
		{
			Name: "Secret used by ingress..tls.secretName, but not existing in vCluster",
			// Syncer: secrets.New,

			InitialPhysicalState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: "default",
						UID:       "123",
					},
				},
			},
			InitialVirtualState: []runtime.Object{
				// ingress referencing secret that translates to the same name as existing secret
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: "default",
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{
							{
								SecretName: "a",
							},
						},
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: "default",
							UID:       "123",
						},
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				// ingress should remain
				networkingv1.SchemeGroupVersion.WithKind("Secret"): {
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: "default",
						},
						Spec: networkingv1.IngressSpec{
							TLS: []networkingv1.IngressTLS{
								{
									SecretName: "a",
								},
							},
						},
					},
				},
			},

			shouldErr: false,
		},
	} {

		// testing scenario:
		// virt object queued (existing, nonexisting)
		// corresponding physical object (nil, not-nil)

		// setup mocks
		options := &Options{}
		scheme := testingutil.NewScheme()
		ctx := context.Background()
		pClient := testingutil.NewFakeClient(scheme, tc.InitialPhysicalState...)
		vClient := testingutil.NewFakeClient(scheme, tc.InitialVirtualState...)

		fakeContext := generictesting.NewFakeRegisterContext(pClient, vClient)

		syncer, err := secrets.New(fakeContext)
		assert.NilError(t, err)

		controller := &syncerController{
			syncer:         syncer,
			log:            loghelper.New(syncer.Name()),
			vEventRecorder: &testingutil.FakeEventRecorder{},
			physicalClient: vClient,

			currentNamespace:       fakeContext.CurrentNamespace,
			currentNamespaceClient: fakeContext.CurrentNamespaceClient,

			virtualClient: vClient,
			options:       options,
		}

		// execute
		_, err = controller.Reconcile(ctx, ctrl.Request{})
		if tc.shouldErr {
			assert.ErrorContains(t, err, tc.errMsg)
		} else {
			assert.NilError(t, err)
		}

		// assert expected result
		// Compare states
		if tc.ExpectedPhysicalState != nil {
			for gvk, objs := range tc.ExpectedPhysicalState {
				err := generictesting.CompareObjs(ctx, t, tc.Name+" physical state", pClient, gvk, scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Physical State mismatch: %v", tc.Name, err)
				}
			}
		}
		if tc.ExpectedVirtualState != nil {
			for gvk, objs := range tc.ExpectedVirtualState {
				err := generictesting.CompareObjs(ctx, t, tc.Name+" virtual state", vClient, gvk, scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Virtual State mismatch: %v", tc.Name, err)
				}
			}
		}
	}
}
