# 🔍 GitOps Policy Check: my-app

| Timestamp | Base | Head | Environments |
-|-|-|-
2025-10-28 21:02:48 UTC | base | head | `stg`, `prod`

## 📊 Manifest Changes




### [`prod`]: `36` lines (32➕/4➖)




```diff
--- before	2025-10-28 21:02:48
+++ after	2025-10-28 21:02:48
@@ -51,7 +51,7 @@
           value: production
         - name: LOG_LEVEL
           value: info
-        image: nginx:1.21
+        image: nginx:latest
         livenessProbe:
           failureThreshold: 3
           httpGet:
@@ -73,12 +73,45 @@
           timeoutSeconds: 3
         resources:
           limits:
-            cpu: 1000m
             memory: 512Mi
           requests:
             cpu: 500m
             memory: 256Mi
 ---
+apiVersion: batch/v1
+kind: CronJob
+metadata:
+  labels:
+    environment: prod
+  name: prod-hello-world-cronjob
+  namespace: my-app-prod
+spec:
+  failedJobsHistoryLimit: 1
+  jobTemplate:
+    metadata:
+      labels:
+        environment: prod
+    spec:
+      backoffLimit: 0
+      template:
+        metadata:
+          labels:
+            environment: prod
+        spec:
+          containers:
+          - command:
+            - /bin/sh
+            - -c
+            - |
+              echo "hello world"
+              sleep 1800  # 30 minutes = 1800 seconds
+              echo "shutting down"
+            image: busybox:1.35
+            name: hello-world
+          restartPolicy: Never
+  schedule: 0 */12 * * *
+  successfulJobsHistoryLimit: 3
+---
 apiVersion: autoscaling/v2
 kind: HorizontalPodAutoscaler
 metadata:
@@ -197,7 +230,7 @@
   namespace: my-app-prod
 spec:
   rules:
-  - host: my-app-prod.example.com
+  - host: my-app.example.com
     http:
       paths:
       - backend:
@@ -209,5 +242,5 @@
         pathType: Prefix
   tls:
   - hosts:
-    - my-app-prod.example.com
+    - my-app.example.com
     secretName: my-app-prod-tls

```






### [`stg`]: `16` lines (12➕/4➖)




```diff
--- before	2025-10-28 21:02:48
+++ after	2025-10-28 21:02:48
@@ -7,6 +7,7 @@
   labels:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   name: stg-my-app-service
   namespace: my-app-stg
@@ -19,6 +20,7 @@
   selector:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   type: ClusterIP
 ---
@@ -28,6 +30,7 @@
   labels:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   name: stg-my-app
   namespace: my-app-stg
@@ -37,12 +40,14 @@
     matchLabels:
       app: my-app
       environment: stg
+      github.com/nvatuan/domains: my-app
       version: v1.0.0
   template:
     metadata:
       labels:
         app: my-app
         environment: stg
+        github.com/nvatuan/domains: my-app
         version: v1.0.0
     spec:
       containers:
@@ -50,7 +55,7 @@
         - name: ENVIRONMENT
           value: staging
         - name: LOG_LEVEL
-          value: debug
+          value: warn
         image: nginx:1.21
         livenessProbe:
           httpGet:
@@ -69,8 +74,8 @@
           periodSeconds: 5
         resources:
           limits:
-            cpu: 500m
-            memory: 256Mi
+            cpu: 800m
+            memory: 512Mi
           requests:
             cpu: 250m
             memory: 128Mi
@@ -81,6 +86,7 @@
   labels:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   name: stg-my-app-hpa
   namespace: my-app-stg
@@ -128,6 +134,7 @@
   labels:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   name: stg-my-app-keda
   namespace: my-app-stg
@@ -138,7 +145,7 @@
     replicas: 1
   idleReplicaCount: 0
   maxReplicaCount: 8
-  minReplicaCount: 1
+  minReplicaCount: 4
   pollingInterval: 15
   scaleTargetRef:
     name: my-app
@@ -167,6 +174,7 @@
   labels:
     app: my-app
     environment: stg
+    github.com/nvatuan/domains: my-app
     version: v1.0.0
   name: stg-my-app-ingress
   namespace: my-app-stg

```







## 🛡️ Policy Evaluation

| **Environments** | **Success** | **Failed** | **Omitted** |
|--------------|---------|--------|---------|
| `prod` | `1`✅ | `2`❌ | `0`⏭️ |
| `stg` | `1`✅ | `2`❌ | `0`⏭️ |


<details> <summary> Policy Evaluation Matrix: </summary>

| Policy Name | stg | prod |
|-------------|-----|------|
| Service Taggings | ✅ PASS | ❌ FAIL |
| Service High Availability | ❌ FAIL | ❌ FAIL |
| Service No CPU Limit | ❌ FAIL | ✅ PASS |

</details>

<details> <summary> Failing Policies Details: </summary>

#### 🚫 BLOCKING Policies | `prod`: `1`❌ | `stg`: `0`❌ |

##### [`stg`] environment
* None! 🙌

##### [`prod`] environment 


* Policy `Service Taggings` failed with the following messages:
  * CronJob prod-hello-world-cronjob does not have the required label 'github.com/nvatuan/domains'
  * Deployment prod-my-app does not have the required label 'github.com/nvatuan/domains'




#### ⚠️ WARNING Policies |  `prod`: `1`❌ | `stg`: `1`❌ |
##### [`stg`] environment 


* Policy `Service High Availability` failed with the following messages:
  * Deployment 'stg-my-app' must have PodAntiAffinity or PodTopologySpread for high availability



##### [`prod`] environment 


* Policy `Service High Availability` failed with the following messages:
  * Deployment 'prod-my-app' must have PodAntiAffinity or PodTopologySpread for high availability




#### 💡 RECOMMEND Policies |  `prod`: `0`❌ | `stg`: `1`❌ |
##### [`stg`] environment 


* Policy `Service No CPU Limit` failed with the following messages:
  * Deployment 'stg-my-app' container 'my-app' should not have a cpu limit, found: 800m




##### [`prod`] environment
* None! 🙌


#### ⏭️ Omitted Policies |  `prod`: `0`❌ | `stg`: `0`❌ |

##### [`stg`] environment
* None! 🙌


##### [`prod`] environment
* None! 🙌


</details>

