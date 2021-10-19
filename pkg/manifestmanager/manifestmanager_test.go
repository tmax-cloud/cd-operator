package manifestmanager

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// type clusterSecret struct {
// 	secretName string
// 	// value      string
// }

type getManifestURLTestCase struct {
	repoURL        string
	path           string
	targetRevision string

	expectedErrOccur bool
	expectedErrMsg   string
	expectedResult   []string
}

// type applyManifestTestCase struct {
// 	repoURL        string
// 	path           string
// 	targetRevision string
// 	destName       string
// 	destNamespace  string

// 	isDefaultCluster bool

// 	// clusterSecret
// }

func TestGetManifestURL(t *testing.T) {
	// Set loggers
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}
	var m ManifestManager
	// https://github.com/tmax-cloud/cd-operator.git
	// api.github.com/repos/argoproj/argocd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=master

	tc := map[string]getManifestURLTestCase{
		"githubValidURLDir": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps",
			path:             "guestbook",
			targetRevision:   "main",
			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/test/guestbook-testui-deployment.yaml"},
		},
		"githubValidURLFile": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps",
			path:             "deployment.yaml",
			targetRevision:   "main",
			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml"},
		},
		"githubInvalidURL": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps-fake",
			path:             "guestbook",
			targetRevision:   "main",
			expectedErrOccur: true,
			expectedErrMsg:   "404 Not Found",
		},
		// TODO: tc for gitlab & other apiURL
		// "gitlabValidURL": {

		// },
		// "gitlabInvalidURL": {

		// },
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				Spec: cdv1.ApplicationSpec{
					Source: cdv1.ApplicationSource{
						RepoURL:        c.repoURL,
						Path:           c.path, // 아직 single yaml만 가능
						TargetRevision: c.targetRevision,
					},
				},
			}
			result, err := m.GetManifestURLList(app)
			if c.expectedErrOccur {
				require.Error(t, err)
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expectedResult, result)
			}
		})
	}
}

// func TestApplyManifest(t *testing.T) {
// 	// Set loggers
// 	if os.Getenv("CI") != "true" {
// 		logrus.SetLevel(logrus.InfoLevel)
// 		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
// 	}

// 	tc := map[string]applyManifestTestCase{
// 		"defaultCluster": {
// 			repoURL:        "https://github.com/tmax-cloud/cd-example-apps",
// 			path:           "guestbook/guestbook-ui-svc.yaml",
// 			targetRevision: "main",
// 			destName:       "",
// 			destNamespace:  "test",

// 			isDefaultCluster: true,
// 		},
// 		// 		"externalCluster": {
// 		// 			repoURL:        "https://github.com/tmax-cloud/cd-example-apps",
// 		// 			path:           "guestbook/guestbook-ui-svc.yaml",
// 		// 			targetRevision: "main",
// 		// 			destName:       "testDestination",
// 		// 			destNamespace:  "test",
// 		// 			clusterSecret: clusterSecret{
// 		// 				secretName: "testDestination",
// 		// 				value: `apiVersion: v1
// 		// clusters:
// 		// - cluster:
// 		//     certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJeE1Ea3dNVEE1TWpJeU5sb1hEVE14TURnek1EQTVNakl5Tmxvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTWhkCkNPRmNLVXdVejExZzVtUGZUVFl6Y0syUi9mdjRzOVRMSzBqaURxQ0VJVncrRG0rOWdEcDVtOFRTQ2k2Y3V3aE0KMFBOQW5KbFh3VVkybWVnRm1FcGZLeHhHR2hwbmk4dWN0eXAwZk50UDhCYmsySlZRdk1keTkwMzRKNnpFS0hScgp4YW54SWFYNE9YTzIyQUg0RzlqSVhvTUFxNHQyWFphNFAyRUxoejkrOW1yN2JTbWZzSFBzazRBYU1TUUlTUjVVClhESjg0dG1vaHhXRFUxbXplci85V2gweFgrbFI0Um54bUhMTmdTZEgzVlZrcFE3Z2F2UDR6MHhXTEtweDFsTkoKRk1DS1FISlpIeFBRaGFZamUrdERJUldzTlAyMjU2blhkVTNsVjRCbGF1M0JPdkJDSmpBOUpxa0pCdVNGNzlPZgpqYmd5amNFZDJudmtoTHBwb2s4Q0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZOWGhJZ2t3ZS92MlZ3NFJpWHN0c1UyZHNpVCtNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFER2xkM0JkODB4eUQ3MDJsLzFrV21zWHMxRWlUZXgycDJLU1VFMHRobzlXL3FrWDBQdwo1ZTNOcFc4QmZpdFA2VXFMMTd4WmVuMThDa0p6NXdyOUtsRW10eE5KQWdCSXJGS0M0REJpdmxrblU1empRbk9YCmNTbUFTelpPMktIeFJMcU5WRFNQTU5VTHp6NVZEUzUwRklmeWtwaGduYTk2M0Fyc2Q2cFVjRDljMzdlV1RJYjgKR2ZSQ1BCdlJ2dEZ6Qk1nMHpVWWRrWW1iMXNzM3U2WEc4QVNHYVVNUnZoK0ord0tFNDZydFNTbE5YOFRwVHpFbgptWDFrYXl5TjFFMUxsc1RLZCtsaE5ORERxODM5VGsxaVp6MEd1cE10b0JiY2luOVZocTRzYmdvV1h1MHVDVkw5CkJaQkI5eU5MaG1jR0tqRWREQ3RsZnF3Q1RKL240QnA1aGJPMAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
// 		//     server: https://172.22.11.2:6443
// 		//   name: kubernetes
// 		// contexts:
// 		// - context:
// 		//     cluster: kubernetes
// 		//     user: kubernetes-admin
// 		//   name: kubernetes-admin@kubernetes
// 		// current-context: kubernetes-admin@kubernetes
// 		// kind: Config
// 		// preferences: {}
// 		// users:
// 		// - name: kubernetes-admin
// 		//   user:
// 		//     client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lJVFd5amF6Tm05OVl3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TVRBNU1ERXdPVEl5TWpaYUZ3MHlNakE1TURFd09USXlNamhhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXBHTnZXelVQTS8rdmdGam4KWTlPQklISm52c2dSVWZLcDkxbGc0MHo4TnZza1ZFbHhqeEo0cWo5UzZyYzM2ZGdYdDgrcTFROTdib3Z0YU1IdgpBVkdMb0pDUlErcXlGVlVESkRnbWl2WEZmeVNMYW1OTFZqekdoRnVsNTdHdzlyNWRCbXRnQzVvR1VobVJLc24xCmg0dXFwWUFIMjhxOFV4NTBXaWw3Z1ozZjU3d0ZGZVJYSHl4bWo1bDhKd3NwQ3FPMWRtVlZtSlFLdnNWSDVYQW0KOGtTNUhFSnVMNmpNTlRqZWc3QjlZZ1VmeDk0VDFTcHJvNEtGMXdVeTdSczlhTThMTEpVS1g2UzYzSXR3bU01dwpVVTN0NnFZSXFGeGZCb2Q4Z3ByVVJsRmh4MHZCd3dNc2VSZlVnY2RtUEhwV3ZoakdhMFZFM1BIdDd4U0ZnU2N6CithTjNDd0lEQVFBQm8wZ3dSakFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0h3WURWUjBqQkJnd0ZvQVUxZUVpQ1RCNysvWlhEaEdKZXkyeFRaMnlKUDR3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFBTDNlNUVjUGowMGhTR0ozbmt2K2lQZnNWdGpDZFhQaVAzeDNISld2amZsbUlyY3Q5SFovKzNqClVid2diV0syYStUSE16QjMyMGdIdDFFUlk0VEV4dWVobXdGY0o0Ym5VWXZ5OHdiZnVSWHh6a2JXcnJIbWRzN1cKRTZ4bllDZVZKSXRyRGZCUWdldDFEWnY5aGdUaEhBdDV1c2FnaTVHNmZ5MzhBYXFTTk5EVTk4VXFSa1pVcUJRawphOG9VSTRxUU5YenFmZWptVm0wcWdxNjI3QWJ6SmlOZnNjMGJqOVlaT05VdkdTdGlUOUZ5eTQrWEpkNzc3RkhYCjZVdGdKWlJFL040OUEyT2JIWkdQWWFrSDl0L1BrUWhlcTduVVhiWGFHL3NZQ1A0YnRwY3F4bzgrZUJMSHpUTlMKTUxITE5aNlh3ZnVIRHk0T2hmYk1WUCs4Wk1iNzI0VT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
// 		//     client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBcEdOdld6VVBNLyt2Z0Zqblk5T0JJSEpudnNnUlVmS3A5MWxnNDB6OE52c2tWRWx4Cmp4SjRxajlTNnJjMzZkZ1h0OCtxMVE5N2JvdnRhTUh2QVZHTG9KQ1JRK3F5RlZVREpEZ21pdlhGZnlTTGFtTkwKVmp6R2hGdWw1N0d3OXI1ZEJtdGdDNW9HVWhtUktzbjFoNHVxcFlBSDI4cThVeDUwV2lsN2daM2Y1N3dGRmVSWApIeXhtajVsOEp3c3BDcU8xZG1WVm1KUUt2c1ZINVhBbThrUzVIRUp1TDZqTU5UamVnN0I5WWdVZng5NFQxU3ByCm80S0Yxd1V5N1JzOWFNOExMSlVLWDZTNjNJdHdtTTV3VVUzdDZxWUlxRnhmQm9kOGdwclVSbEZoeDB2Qnd3TXMKZVJmVWdjZG1QSHBXdmhqR2EwVkUzUEh0N3hTRmdTY3orYU4zQ3dJREFRQUJBb0lCQUVVQkVGOWkyR3psYVZBaApBWkJmMmhZNnI5M2ZzWldLblZvZEJKU2xYa0hlRGhQcmVHV3NSVWFCcWxhb2Jpb1U4Vy9SRms2MVh3UzZhLy9MCldINWZNcE5GM0JSOFVpQ3VQTkZaV0tTQUlsVUtqQk11ZHhOT0U2Ni9vZGF1T2pCNUhDZHpyeTl2aWpPd1U4VjQKWFQ1Mm5EMDRqeFB0K0R1VHp4ZUJ6anhNZnc2UXEvdTRMcWNDT0tsdVp3b2xIWkQvUGo3QmZxQmxSQXpPYVNLNQppUTI3RjNDVkdZZ1NtaHd4ZG9idnFWTitBQkxMbkI2VUdmbVFIbEMwZXJUVW5CK25NQlBaeERHWGcwMVpKS3dECkx4RWJqczJub2c3REtybWhTZEJndDNsdXp6UU1XZHJJMCswU2ZxOGlBeVBMVEExRVRNdW4rUDlrRGRreGdQMGUKZlhmY2o2RUNnWUVBeWp1dkVqTlhBRXhTbUFoRjI1R3FOUzFuMjA3cHJZRDlyZE1QTnh3K3prWWhwb3dONWI4egp5bzhkeE9xYXpYbThSMW1CbStYNVk1Q25jV0VrTlNEMXBnS1ZERWdNaUVYV1ZGM1hpY2UzZE8ra2hTWm1yalJzClhuRDlBOGhBMlpsRGo5Nm9KU0pqajkzY3FUd0ZYVmwyNFNrNDMrMURUMWJYTDdTN0h4QW5mK01DZ1lFQTBCZjYKVXN0VWRuQjVZU1dHZEY2T3QwTmorZnRQKzZrNEZ2c2xOd05OcDBld201ZnovY3JXQlljZUhoeDhlTCs5TWcvNAp3czcrQ0ljU0d5cklVOUpxNXRhM0lPSUQxblJGcnYwN1V4MGRXYTcwb3lTMGhLK0dtb0lCZjUvY09sdUZ2NWE0CmxPWGxmMkdpbmRSWUJDbnloUVRQU1hQLzNZdWxTL3dJK0c3dGhMa0NnWUJ5SjlLaFVYMncyMlJjRVg5dGZBSTYKVmxFanlKMjdwTzZOcW5BU1NjMWlIdEJyOU83N1d6emZBSDVyWTRyU3BmOFR2NENjQWVzT3V0N3A3MDNDOThIeQpYYzdJeWZyWkNhTDhxS1E4VUJKTTNlRmVqOWl5U1VGSzVqak1ZOFBIa081RVRnbFlQTnM2b0tBb240cmZzTnFjCkt1ckI3R3BzWkxhL1pTT2pXemtReFFLQmdRQ3crVE55NW1uV3NLRUo5WmY3cjg5QUhKZ1NLYUZFTGczOXZXbFEKK0FZNmxjV2xEZjM3Z1YyekpjNS9YVXFlaHJLb3VOeWZFTnNLOVpSNGRsSVl0NE1pL3NpUHRxZjg0clhBdEt5WAphdE5qU2wvVHY0dW1ySUNWTnF0L2xyejlCSWtpLzFQTGpoazMxQmt3a1Q2cGkrTXRMWUg4dmlLRWtCYnNJRlNnCnMvWmNRUUtCZ0FoeGtndjh5ZHlCWEYzMzJtTUpBVXdUTTBGdEhxbFpobHhYS29yakNBT1lVODBSVnN1Qlo5eW0KOTlvNWY2NzhDWEFMZ0NtM2lmRHVaMUk3TlhIaTZjZ3VSbTN6TVVPd2RiMFpURDQxRE5DdmJZMUY0WnppdnY0aApJMTBSd3ZGakNnc0VUYVN2U2ZHRUZuWXJyVTYyL1Z6Uis3ZE5nOTBMaHlaY1dzR2FzMGs1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
// 		// `,
// 		// 			},
// 		// },
// 	}

// 	s := runtime.NewScheme()
// 	utilruntime.Must(cdv1.AddToScheme(s))
// 	utilruntime.Must(v1.AddToScheme(s))
// 	server := newTestServer()

// 	for name, c := range tc {
// 		t.Run(name, func(t *testing.T) {
// 			err := createFakeCert()
// 			require.NoError(t, err)

// 			defer func() {
// 				err = removeFakeCert()
// 				require.NoError(t, err)
// 			}()
// 			app := &cdv1.Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "test",
// 					Namespace: "test",
// 				},
// 				Spec: cdv1.ApplicationSpec{
// 					Source: cdv1.ApplicationSource{
// 						RepoURL:        c.repoURL,
// 						Path:           c.path,
// 						TargetRevision: c.targetRevision,
// 					},
// 					Destination: cdv1.ApplicationDestination{
// 						Name:      c.destName,
// 						Namespace: c.destNamespace,
// 					},
// 				},
// 			}

// 			var fakeCli client.Client
// 			if !c.isDefaultCluster {
// 				sec := &v1.Secret{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:      c.destName + "-kubeconfig",
// 						Namespace: app.Namespace,
// 					},
// 					StringData: map[string]string{
// 						// "value": c.value,
// 					},
// 				}
// 				fakeCli = fake.NewFakeClientWithScheme(s, app, sec)
// 			} else {
// 				fakeCli = fake.NewFakeClientWithScheme(s, app)
// 			}
// 			m := ManifestManager{Client: fakeCli}
// 			err = m.ApplyManifest(server.URL, app)
// 			assert.Equal(t, err, nil)
// 		})
// 	}
// 	//TODO : 아웃풋인 DeployResource을 활용해서 Test 짜기
// 	//TODO : resource 삭제하는 로직 필요
// 	//TODO : team 환경 이용하여 테스트 함. 수정 필요
// }

// func newTestServer() *httptest.Server {
// 	router := mux.NewRouter()

// 	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
// 		defer func() {
// 			_ = req.Body.Close()
// 		}()
// 		// yaml은 tab 말고 space로만 구문 가능
// 		data := `apiVersion: v1
// kind: Service
// metadata:
//   name: guestbook-ui-test
// spec:
//   ports:
//   - port: 80
//     targetPort: 80
//   selector:
//     app: guestbook-ui

// `
// 		_, err := io.WriteString(w, data)
// 		if err != nil {
// 			return
// 		}
// 	})

// 	return httptest.NewServer(router)
// }

// TODO: 따로 패키지로 빼기
// func createFakeCert() error {
// 	if err := os.Mkdir("./test_cert/", os.ModePerm); err != nil {
// 		return err
// 	}
// 	if err := ioutil.WriteFile("./test_cert/ca.crt", []byte("test"), 0644); err != nil {
// 		return err
// 	}
// 	if err := ioutil.WriteFile("./test_cert/tls.key", []byte("test"), 0644); err != nil {
// 		return err
// 	}
// 	if err := ioutil.WriteFile("./test_cert/tls.crt", []byte("test"), 0644); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func removeFakeCert() error {
// 	if err := os.RemoveAll("./test_cert"); err != nil {
// 		return err
// 	}
// 	return nil
// }
