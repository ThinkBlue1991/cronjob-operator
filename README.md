## 使用operator-sdk 创新crd

### 安装operator-sdk
#### 下载
```shell script
RELEASE_VERSION=v0.11.0
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
```
`如果本机没有安装公钥，会报错如下：`
```shell script
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ gpg: assuming signed data in 'operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin'
$ gpg: Signature made Fri Apr  5 20:03:22 2019 CEST
$ gpg:                using RSA key <KEY_ID>
$ gpg: Can't check signature: No public key
```
``执行如下脚本：``
```shell script
gpg --recv-key "$KEY_ID"
gpg --keyserver keyserver.ubuntu.com --recv-key "$KEY_ID"
```
#### 安装
```shell script
$ chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
```

### 创建新的工程
```shell script
## 初始化工程
mkdir -p $GOPATH/example/
cd $GOPATH/example
export GO111MODULE=on
operator-sdk new cronjob-operator --repo github.com//ThinkBlue1991/cronjob-operator
cd cronjob-operator

## 添加CRD的api
operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=CronJob
```
### 定义Status和spec
- 修改pkg/apis/cache/v1alphal/cronjob_types.go
- 修改types后，针对resource进行更新和生成代码
```shell script
operator-sdk generate k8s
```

### openapi验证
```shell script
operator-sdk generate openapi
```
`查看deploy/crds/cache.example.com_cronjob_crd.yaml 其中根据spec定义了crd参数的属性`

### 增加新的Controller
```shell script
operator-sdk add controller --api-version=cache.example.com/v1alpha1 --kind=CronJob
```
`会在pkg/controller/cronjob/下出现cronjob_controller.go文件`
`修改cronjob_controller.go文件，主要关心函数func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error)`

### 编译运行operator
```shell script
kubectl create -f deploy/crds/cache.example.com_cronjob_crd.yaml
```

```shell script
operator-sdk build hub.geovis.io/zhangjx/cronjob-operator:v0.1
ocker push hub.geovis.io/zhangjx/cronjob-operator:v0.1
```
**Note:** `hub.geovis.io是本地的私有仓库`
#### 修改operator.yaml文件
```shell script
sed -i 's|REPLACE_IMAGE|hub.geovis.io/zhangjx/cronjob-operator:v0.1|g' deploy/operator.yaml
```

#### 执行其他的yaml文件
```shell script
# Setup Service Account
$ kubectl create -f deploy/service_account.yaml
# Setup RBAC 
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
# Deploy the app-operator
$ kubectl create -f deploy/operator.yam
```
#### 创建新的cronjob
```shell script
kubectl create -f deploy/crds/cronjob.yaml
```
### 查看结果
#### crd
![](https://raw.githubusercontent.com/ThinkBlue1991/Images1991/master/img/20191023195352.png)
#### operator
![](https://raw.githubusercontent.com/ThinkBlue1991/Images1991/master/img/20191023195506.png)

#### job
![](https://raw.githubusercontent.com/ThinkBlue1991/Images1991/master/img/20191023195537.png)
**Note:**`执行4次job任务，cronjob任务结束`
