# Quick Start Guide

This guide leads you to start using CD operator, with simple examples.
The contents are as follows.

## You need...
- Git repository contains the application manifests. (e.g. https://github.com/tmax-cloud/cd-example-apps)
- k8s cluster installed cd-operator

## Create `Application`
```yaml
apiVersion: cd.tmax.io/v1
kind: Application
metadata:
  name: tutorial-application
spec:
  source:
    repoURL: https://github.com/tmax-cloud/cd-example-apps
	path: guestbook
    targetRevision: main
  destination:
    server:
    namespace:
    name:

```
위의 예시처럼, Application CRD을 cd-operator가 설치된 cluster에 apply 시킵니다.
## Register Webhook manually
아래의 방법은 수동으로 webhook을 등록할 때의 과정입니다.
### Check Webhook Secret
```bash
kubectl get applications tutorial-application
```
위 단계에서 생성된 Applcation을 조회하여 Application.Status.Secrets값을 얻습니다. (아래 단계에서 사용될 예정)
### Add Webhook
Webhook을 추가할 수 있는 셋팅 페이지로 이동하여,
(github기준 : 매니페스트가 담긴 Repo > Settings > Webhooks)
webhook의 Payload URL을 넣어주고,
Content Type은 application/json으로 
Secret은 위에서 얻은 Application.Status.Secrets 값을 넣어줍니다. 
그리고 'just the push event'을 골라주고 webhoook을 등록합니다. 

## Edit manifest file and push the commit
새로운 manifest을 작성하거나 기존의 manifest을 수정하고, 해당 repo와 targetRevision에 push을 합니다. 

## Check resource 
targetRevision의 repo 내에 있는  manifest대로 resource가 클러스터에 생성됬는지 확인합니다.  
