### Secure random strings

we need to replace all usage of random uuids or hex strings with a more k8s-native SRS (secure random string) approach, just generate a string of 6-8 characters similar to how k8s names jobs and pods. this method may already exist in places, just need to make sure its used everywhere.

### Add humanlayer project ID to contactChannel status

when bootstrapping and validating contact channels, use the api key to fetch the humanlayer projet id

you can check the project with GET https://api.humanlayer.dev/humanlayer/v1/project with the API key

add the project slug and org slug to the contactChannel status.

### add contactChannel to task spec

when creating a new task, allow an optional contactChannel reference to be passed in.

if a contact channel is set, check that it exists during validation of the task, and that it is Ready


### add channelApiKey and Id to the contactChannel Spec

allow a channel to be specified with a channelApiKey secretKeyRef and channelId

on validation, if channelApiKey is set, check that apiKey reference is unset, and that channelId is set

on validation, ping the humanlayer api endpoint (to be written) on /humanlayer/v1/contact_channel/{channelId} with the channelApiKey to check that the channel exists and is ready. Set the org, project on the contactChannel status

update humanLayer client usage to use the channelApiKey and channelId when creating a new contactChannel


### support for v1Beta3 events as inbound to a specific server route 

support for v1Beta3 events as inbound to a specific server route with a contact channel id and an api key

for these events, set the contact channel id and the api key in a new contactChannel
create a new task with a reference to the contactChannel that was created

when the task is completed (final answer with no tool calls), instead of appending that message to the contextWindow,
append a "repond_to_human" tool call, with the Content as the argument

Then create a ToolCall as if the model had emitted a tool call, and let it poll until completion.

the v1 beta3 events can be used like so (typescript example, you will need to translate to go)


```
type V1Beta3ConversationCreated = {
    is_test: boolean;
    type: "conversation.created";
    // an api key that can be used to send messages back on the same channel, e.g. same email thread, same slack thread, etc.
    channel_api_key: string; 
    event: {
        user_message: string;
        // a contact channel id that fully identifies the channel stored in humanlayer for usage during requests
        contact_channel_id: number;
        agent_name: string;
    }
}

type V1Beta3Event = V1Beta3ConversationCreated;

const outerLoop = async (req: Request, res: Response) => {
    console.log("outerLoop", req.body);
    const body = req.body as V1Beta3Event;
    const hl = humanlayer({
        runId: process.env.HUMANLAYER_RUN_ID || `12fa-agent`,
        // use the humanlayer api key for the channel
        apiKey: body.channel_api_key,
        contactChannel: {
            // use the humanlayer contact channel id for the channel when sending messages back
            channel_id: body.event.contact_channel_id,
        } as ContactChannel // todo export this type flavor
    });

    // ... snip ...

        switch (lastEvent.data.intent) {
            case "request_more_information": // NOTE - in go, these are both just Content with no Tool Calls
                // create a human contact with the model's question for the user
            case "done_for_now": // NOTE - in go, these are both just Content with no Tool Calls
                // create a human contact with the model's final answer
                hl.createHumanContact({
                    spec: {
                        msg: lastEvent.data.message,
                        state: {
                            thread_id: threadId
                        }
                    }
                });
                console.log(`created human contact "${lastEvent.data.message}"`);
                break;
            case "divide":
                const intent = lastEvent.data.intent;
                // remove intent from kwargs payload
                const { intent: _, ...kwargs } = lastEvent.data;
                // example of requesting approval - will use hl instance with the embedded api key and contact channel id
                hl.createFunctionCall({
                    spec: {
                        fn: intent,
                        kwargs: kwargs,
                        state: {
                            thread_id: threadId
                        }
                    }
                });
                console.log("created function call", {intent, kwargs});
                break;
        }
    });
    res.json({ status: "ok" });
}
```

when model emits final answer (no tool calls), replace with human contact

when the model emits a final answer (no tool calls), instead of appending that message to the contextWindow,
instead create a human contact with the message using the client 

### multiple llm calls can happen in parallel

This one is weird. You must try to reproduce it. A task like "fetch the data at https://lotrapi.co./api/v1/characters and then fetch data about two of the related locations" with a fetch mcp, will cause a many-turn conversation with multiple tool calls and parallel cool calls. This causes bugs. We send invalid payloads to llms, etc.

This should not be hard to reproduce. ask me if you have trouble

here are some examples from kubectl output

0s          Normal    AllToolCallsCompleted       task/manager-task                                                       All tool calls completed
0s          Normal    SendingContextWindowToLLM   task/manager-task                                                       Sending context window to LLM
0s          Normal    LLMFinalAnswer              task/manager-task                                                       LLM response received successfully
0s          Normal    SendingContextWindowToLLM   task/manager-task                                                       Sending context window to LLM
0s          Normal    LLMFinalAnswer              task/manager-task                                                       LLM response received successfully

and 

0s          Normal    SendingContextWindowToLLM   task/delegate-manager-task-00293a0-tc-01-web-search                     Sending context window to LLM
0s          Normal    LLMFinalAnswer              task/delegate-manager-task-00293a0-tc-01-web-search                     LLM response received successfully
0s          Normal    SendingContextWindowToLLM   task/delegate-manager-task-00293a0-tc-01-web-search                     Sending context window to LLM
0s          Normal    LLMFinalAnswer              task/delegate-manager-task-00293a0-tc-01-web-search                     LLM response received successfully

feel like we hit weird race conditions


You may also notice things like ValidationSucceeded *multiple times* dude to similar same-controller-reconciling-twice.
