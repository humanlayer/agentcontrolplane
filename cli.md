### ACP CLI

(this is a spec for a to-be-built CLI tool)

## Install

```bash
brew install humanlayer/tap/acp
```

## Usage

```bash
acp --help
```

## Commands


```
$ acp version
ACP CLI version 0.1.10
ACP Operator version 0.1.10
```

```
# convenience command - uses a `kind` binary on your path to create a local cluster
acp kind-dev
```


```
# installs the operator into the cluster
acp install
```

```
# 
acp create agent my-agent --system-prompt "You are a helpful assistant." --mcpservers=fetch,memory,linear
```

```
# creates a task and runs it to completion, streaming model output, etc to the console
acp create task my-task --message "What is on the front page of planetscale.com?" --agent my-agent --stream
```

e.g. 

```
$ acp create task my-task --message "What is on the front page of planetscale.com?" --agent my-agent --stream
Creating task 'my-task'...
Task created successfully
Streaming output from taskrun 'my-task-1'...

Assistant: Let me fetch the front page of planetscale.com to answer your question.

[Tool Call] fetch
URL: https://planetscale.com
Max Length: 5000
Raw: false

Assistant: Based on the fetched content, here's what's on the front page of planetscale.com:

PlanetScale is a MySQL-compatible serverless database platform. The front page highlights:
- Serverless MySQL database platform
- Built for developers
- Features like branching, automatic backups, and high availability
- Free tier available
- Integration with popular frameworks and tools

Would you like me to provide more specific details about any of these aspects?
```



Could use bubbletea or something to make it extra dope