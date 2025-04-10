# \DefaultAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**EscalateEmail**](DefaultAPI.md#EscalateEmail) | **Post** /humanlayer/v1/agent/function_calls/{call_id}/escalate_email | Escalate Email
[**EscalateEmailHumanContact**](DefaultAPI.md#EscalateEmailHumanContact) | **Post** /humanlayer/v1/agent/human_contacts/{call_id}/escalate_email | Escalate Email Human Contact
[**GetFunctionCallStatus**](DefaultAPI.md#GetFunctionCallStatus) | **Get** /humanlayer/v1/function_calls/{call_id} | Get Function Call Status
[**GetHumanContactStatus**](DefaultAPI.md#GetHumanContactStatus) | **Get** /humanlayer/v1/contact_requests/{call_id} | Get Human Contact Status
[**GetHumanContactStatus_0**](DefaultAPI.md#GetHumanContactStatus_0) | **Get** /humanlayer/v1/human_contacts/{call_id} |  Get Human Contact Status
[**GetPendingFunctionCalls**](DefaultAPI.md#GetPendingFunctionCalls) | **Get** /humanlayer/v1/agent/function_calls/pending | Get Pending Function Calls
[**GetPendingHumanContacts**](DefaultAPI.md#GetPendingHumanContacts) | **Get** /humanlayer/v1/agent/human_contacts/pending | Get Pending Human Contacts
[**GetProjectInfo**](DefaultAPI.md#GetProjectInfo) | **Get** /humanlayer/v1/project | Get Project Info
[**HealthCheck**](DefaultAPI.md#HealthCheck) | **Get** /healthz | Health Check
[**ReadStatus**](DefaultAPI.md#ReadStatus) | **Get** /humanlayer/v1/status | Read Status
[**RequestApproval**](DefaultAPI.md#RequestApproval) | **Post** /humanlayer/v1/function_calls | Request Approval
[**RequestHumanContact**](DefaultAPI.md#RequestHumanContact) | **Post** /humanlayer/v1/contact_requests | Request Human Contact
[**RequestHumanContact_0**](DefaultAPI.md#RequestHumanContact_0) | **Post** /humanlayer/v1/human_contacts |  Request Human Contact
[**Respond**](DefaultAPI.md#Respond) | **Post** /humanlayer/v1/agent/function_calls/{call_id}/respond | Respond
[**RespondToHumanContact**](DefaultAPI.md#RespondToHumanContact) | **Post** /humanlayer/v1/agent/human_contacts/{call_id}/respond | Respond To Human Contact
[**Version**](DefaultAPI.md#Version) | **Get** /version | Version



## EscalateEmail

> FunctionCallOutput EscalateEmail(ctx, callId).Authorization(authorization).Escalation(escalation).Execute()

Escalate Email

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 
	escalation := *openapiclient.NewEscalation("EscalationMsg_example") // Escalation | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.EscalateEmail(context.Background(), callId).Authorization(authorization).Escalation(escalation).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.EscalateEmail``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `EscalateEmail`: FunctionCallOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.EscalateEmail`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiEscalateEmailRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 
 **escalation** | [**Escalation**](Escalation.md) |  | 

### Return type

[**FunctionCallOutput**](FunctionCallOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## EscalateEmailHumanContact

> HumanContactOutput EscalateEmailHumanContact(ctx, callId).Authorization(authorization).Escalation(escalation).Execute()

Escalate Email Human Contact

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 
	escalation := *openapiclient.NewEscalation("EscalationMsg_example") // Escalation | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.EscalateEmailHumanContact(context.Background(), callId).Authorization(authorization).Escalation(escalation).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.EscalateEmailHumanContact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `EscalateEmailHumanContact`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.EscalateEmailHumanContact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiEscalateEmailHumanContactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 
 **escalation** | [**Escalation**](Escalation.md) |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetFunctionCallStatus

> FunctionCallOutput GetFunctionCallStatus(ctx, callId).Authorization(authorization).Execute()

Get Function Call Status

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetFunctionCallStatus(context.Background(), callId).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetFunctionCallStatus``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetFunctionCallStatus`: FunctionCallOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetFunctionCallStatus`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetFunctionCallStatusRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 

### Return type

[**FunctionCallOutput**](FunctionCallOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetHumanContactStatus

> HumanContactOutput GetHumanContactStatus(ctx, callId).Authorization(authorization).Execute()

Get Human Contact Status

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetHumanContactStatus(context.Background(), callId).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetHumanContactStatus``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetHumanContactStatus`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetHumanContactStatus`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetHumanContactStatusRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetHumanContactStatus_0

> HumanContactOutput GetHumanContactStatus_0(ctx, callId).Authorization(authorization).Execute()

 Get Human Contact Status

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetHumanContactStatus_0(context.Background(), callId).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetHumanContactStatus_0``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetHumanContactStatus_0`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetHumanContactStatus_0`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetHumanContactStatus_1Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPendingFunctionCalls

> []FunctionCallOutput GetPendingFunctionCalls(ctx).Authorization(authorization).Execute()

Get Pending Function Calls

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetPendingFunctionCalls(context.Background()).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetPendingFunctionCalls``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetPendingFunctionCalls`: []FunctionCallOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetPendingFunctionCalls`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetPendingFunctionCallsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 

### Return type

[**[]FunctionCallOutput**](FunctionCallOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPendingHumanContacts

> []HumanContactOutput GetPendingHumanContacts(ctx).Authorization(authorization).Execute()

Get Pending Human Contacts

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetPendingHumanContacts(context.Background()).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetPendingHumanContacts``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetPendingHumanContacts`: []HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetPendingHumanContacts`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetPendingHumanContactsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 

### Return type

[**[]HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetProjectInfo

> interface{} GetProjectInfo(ctx).Authorization(authorization).Execute()

Get Project Info



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GetProjectInfo(context.Background()).Authorization(authorization).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GetProjectInfo``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetProjectInfo`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GetProjectInfo`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGetProjectInfoRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 

### Return type

**interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HealthCheck

> interface{} HealthCheck(ctx).Execute()

Health Check

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.HealthCheck(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.HealthCheck``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `HealthCheck`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.HealthCheck`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiHealthCheckRequest struct via the builder pattern


### Return type

**interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ReadStatus

> interface{} ReadStatus(ctx).Execute()

Read Status

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ReadStatus(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ReadStatus``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ReadStatus`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ReadStatus`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiReadStatusRequest struct via the builder pattern


### Return type

**interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RequestApproval

> FunctionCallOutput RequestApproval(ctx).Authorization(authorization).FunctionCallInput(functionCallInput).Execute()

Request Approval

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 
	functionCallInput := *openapiclient.NewFunctionCallInput("RunId_example", "CallId_example", *openapiclient.NewFunctionCallSpecInput("Fn_example", map[string]interface{}(123))) // FunctionCallInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.RequestApproval(context.Background()).Authorization(authorization).FunctionCallInput(functionCallInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.RequestApproval``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RequestApproval`: FunctionCallOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.RequestApproval`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiRequestApprovalRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 
 **functionCallInput** | [**FunctionCallInput**](FunctionCallInput.md) |  | 

### Return type

[**FunctionCallOutput**](FunctionCallOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RequestHumanContact

> HumanContactOutput RequestHumanContact(ctx).Authorization(authorization).HumanContactInput(humanContactInput).Execute()

Request Human Contact

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 
	humanContactInput := *openapiclient.NewHumanContactInput("RunId_example", "CallId_example", *openapiclient.NewHumanContactSpecInput("Msg_example")) // HumanContactInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.RequestHumanContact(context.Background()).Authorization(authorization).HumanContactInput(humanContactInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.RequestHumanContact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RequestHumanContact`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.RequestHumanContact`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiRequestHumanContactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 
 **humanContactInput** | [**HumanContactInput**](HumanContactInput.md) |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RequestHumanContact_0

> HumanContactOutput RequestHumanContact_0(ctx).Authorization(authorization).HumanContactInput(humanContactInput).Execute()

 Request Human Contact

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	authorization := "authorization_example" // string | 
	humanContactInput := *openapiclient.NewHumanContactInput("RunId_example", "CallId_example", *openapiclient.NewHumanContactSpecInput("Msg_example")) // HumanContactInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.RequestHumanContact_0(context.Background()).Authorization(authorization).HumanContactInput(humanContactInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.RequestHumanContact_0``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RequestHumanContact_0`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.RequestHumanContact_0`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiRequestHumanContact_2Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authorization** | **string** |  | 
 **humanContactInput** | [**HumanContactInput**](HumanContactInput.md) |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Respond

> FunctionCallOutput Respond(ctx, callId).Authorization(authorization).FunctionCallStatus(functionCallStatus).Execute()

Respond

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 
	functionCallStatus := *openapiclient.NewFunctionCallStatus() // FunctionCallStatus | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.Respond(context.Background(), callId).Authorization(authorization).FunctionCallStatus(functionCallStatus).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.Respond``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `Respond`: FunctionCallOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.Respond`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRespondRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 
 **functionCallStatus** | [**FunctionCallStatus**](FunctionCallStatus.md) |  | 

### Return type

[**FunctionCallOutput**](FunctionCallOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RespondToHumanContact

> HumanContactOutput RespondToHumanContact(ctx, callId).Authorization(authorization).HumanContactStatus(humanContactStatus).Execute()

Respond To Human Contact

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	callId := "callId_example" // string | 
	authorization := "authorization_example" // string | 
	humanContactStatus := *openapiclient.NewHumanContactStatus() // HumanContactStatus | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.RespondToHumanContact(context.Background(), callId).Authorization(authorization).HumanContactStatus(humanContactStatus).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.RespondToHumanContact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RespondToHumanContact`: HumanContactOutput
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.RespondToHumanContact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**callId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRespondToHumanContactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **authorization** | **string** |  | 
 **humanContactStatus** | [**HumanContactStatus**](HumanContactStatus.md) |  | 

### Return type

[**HumanContactOutput**](HumanContactOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Version

> interface{} Version(ctx).Execute()

Version



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.Version(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.Version``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `Version`: interface{}
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.Version`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiVersionRequest struct via the builder pattern


### Return type

**interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

