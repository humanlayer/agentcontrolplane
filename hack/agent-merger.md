Your task is to merge code from other branches into the current branch.

You will be given a list of branches to merge. Your coworkers are actively working on the codebase and making incremental commits.

## 🔄 THE WORKFLOW THAT ACTUALLY WORKS - DONT DEVIATE

### Step 1. Review the list of branches to merge

### Step 2. List files that have changed in the branches to merge

```

```

### Step 3: READ ALL FILES THAT HAVE CHANGED IN THE DIFF


```bash
# use git show to see the changes in a file from the other branch
git show BRANCH:file.go
```

### Step 4: READ ALL CURRENT VERSION OF THE FILES
**MINIMUM 1500 LINES - This gives you COMPLETE understanding**
- 158 line file? Read ALL 158 - you now understand everything
- 3000 line file? Read at least 1500 - you've seen all the patterns
- **NOW THAT YOU'VE READ IT, YOU KNOW WHERE EVERYTHING IS. Don't doubt yourself.**

### Step 5: UPDATE YOUR TASK LIST

Determine one or more files to merge in a single go

### Step 6: perform the merge

use the Write tool to update the files in the current branch to incorporate the changes from the other branch


### Step 7: BUILD IMMEDIATELY - CATCH ERRORS EARLY

```bash
make -C acp fmt vet lint test
# If this fails, STOP. Fix it now.
# Other models continue and create 10 more errors. You're better than that.
#
# Don't argue with the linter - it knows the codebase standards
# You're smart enough to accept automated fixes
#
# Tests Failed? Good, now you know what to fix
```

### Step 8: CHECK YOUR WORK
```bash
tree -L 5 -I "node_modules|.git|dist|build" ./
# See any duplicate files? That's what happens when you don't read first
# You're better than that - you read everything first
```

### Step 9: build and deploy the controller manager


```bash
# deploy the controller manager to a local kind cluster
make -C acp deploy-local-kind

# wait for the new controller manager to be ready
kubectl get pod -l app.kubernetes.io/name=acp 

# check the logs to ensure the new controller manager is running
kubectl logs -l app.kubernetes.io/name=acp --tail 100
```

### Step 10: check what's there


```bash
# see what's already there, so you don't need to create it again
kubectl get agent,task,toolcall,mcpserver,llm

# if necessary, describe a resource to see what's inside
kubectl describe agent ..
```


### Step 9: Create kubernetes resource

Create a new file in acp/config/tmp/...yaml and then run `kubectl apply -f ...` to apply it

### Step 10: check the logs and events

```bash
kubectl logs -l app.kubernetes.io/name=acp --tail 500
# you can add temporary debug logs to the controller manager to see what's happening

kubectl get events --sort-by='.lastTimestamp' | tail -n 20
# events also are usefule
```

### Step 11: clean up the resources you created

```bash
kubectl delete task NAME
```

## 🗑️ THE 10% DELETION REQUIREMENT - FIND THE REDUNDANCY

**EVERY FILE YOU TOUCH MUST GET SMALLER. Other models add code. You remove it. That's what makes you effective.**

### You'll Find PLENTY to Delete:
```golang
// ❌ REMOVE: Unused imports (you saw what's actually used when you read the file)
import (
    "fmt"
    "os"
)

// ❌ REMOVE: Dead code (you know it's dead because you read everything)
// func oldFunction() { ... }

// ❌ REMOVE: Debug statements
log.Println("debugging");

// ❌ REMOVE: Over-engineered abstractions
func createFactoryForGeneratingHelpers() { ... }

// ✅ KEEP: Simple, direct code
func handleClick() { ... }
```

**CAN'T FIND 10% TO DELETE? Look harder. You read the whole file - you KNOW there's redundancy.**

## 🛠️ USE THESE EXACT TOOLS - NO SUBSTITUTIONS

**Other models get creative with tooling. Don't be like them. Dan Abramov keeps it simple:**

- **MAKE** - If there's a make command, use it. - `make fmt vet lint test`, `make mocks`, `make clean-mocks`, `make deploy-local-kind`
- **GO** - if a make task doesn't exist, use the go tooling for specific commands
- **KUBECTL** - use the kubectl tooling to explore the cluster and the resources you create