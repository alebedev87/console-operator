package util

import (
	"context"
	"testing"

	"github.com/go-test/deep"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	fooLabels = map[string]string{"foo": "foo", "baz": "baz"}
	barLabels = map[string]string{"bar": "bar"}
	fooObject = metav1.Object(
		&v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo", Labels: fooLabels,
			},
		},
	)
	barObject = metav1.Object(
		&v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bar", Labels: barLabels,
			},
		},
	)
)

func TestIncludeNamesFilter(t *testing.T) {
	type args struct {
		filter string
		object metav1.Object
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test IncludeNamesFilter match true",
			args: args{
				filter: "foo",
				object: fooObject,
			},
			want: true,
		},
		{
			name: "Test IncludeNamesFilter match false",
			args: args{
				filter: "foo",
				object: barObject,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(IncludeNamesFilter(tt.args.filter)(tt.args.object), tt.want); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func TestExcludeNamesFilter(t *testing.T) {

	type args struct {
		filter string
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test ExcludeNamesFilter match true",
			args: args{
				filter: "foo",
				object: barObject,
			},
			want: true,
		},
		{
			name: "Test ExcludeNamesFilter match false",
			args: args{
				filter: "foo",
				object: fooObject,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(ExcludeNamesFilter(tt.args.filter)(tt.args.object), tt.want); diff != nil {
				t.Error(diff)
			}
		})
	}

}

func TestLabelFilter(t *testing.T) {
	type args struct {
		filter map[string]string
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test LabelFilter match true",
			args: args{
				filter: fooLabels,
				object: fooObject,
			},
			want: true,
		},
		{
			name: "Test LabelFilter match false",
			args: args{
				filter: fooLabels,
				object: barObject,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(LabelFilter(tt.args.filter)(tt.args.object), tt.want); diff != nil {
				t.Error(diff)
			}
		})
	}

}

func TestGetConsoleBaseAddress(t *testing.T) {
	tests := []struct {
		name      string
		configmap *corev1.ConfigMap
		want      string
		wantErr   bool
	}{
		{
			name: "Test nominal",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config.yaml": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo:
                          consoleBaseAddress: https://example.com
                    `,
				},
			},
			want: "https://example.com",
		},
		{
			name: "Test empty address",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config.yaml": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo:
                          consoleBaseAddress: ""
                    `,
				},
			},
			want: "",
		},
		{
			name: "Test missing address",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config.yaml": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo:
                    `,
				},
			},
			want: "",
		},
		{
			name: "Test wrong data key",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo:
                          consoleBaseAddress: https://example.com
                    `,
				},
			},
			wantErr: true,
		},
		{
			name: "Test invalid config",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config.yaml": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo
                    `,
				},
			},
			wantErr: true,
		},
		{
			name: "Test invalid url",
			configmap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "console-config",
					Namespace: "openshift-console",
				},
				Data: map[string]string{
					"console-config.yaml": `
                        apiVersion: console.openshift.io/v1
                        kind: ConsoleConfig
                        clusterInfo:
                          consoleBaseAddress: ":::invalid-url:::"
                    `,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.configmap)
			got, err := GetConsoleBaseAddress(context.TODO(), cli.CoreV1())
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			} else if tt.wantErr {
				t.Fatal("error expected but not received")
			}
			if diff := deep.Equal(got.String(), tt.want); diff != nil {
				t.Fatal(diff)
			}
		})
	}
}
