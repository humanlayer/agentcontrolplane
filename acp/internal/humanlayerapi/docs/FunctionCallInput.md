# FunctionCallInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RunId** | **string** |  | 
**CallId** | **string** |  | 
**Spec** | [**FunctionCallSpecInput**](FunctionCallSpecInput.md) |  | 
**Status** | Pointer to [**NullableFunctionCallStatus**](FunctionCallStatus.md) |  | [optional] 

## Methods

### NewFunctionCallInput

`func NewFunctionCallInput(runId string, callId string, spec FunctionCallSpecInput, ) *FunctionCallInput`

NewFunctionCallInput instantiates a new FunctionCallInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFunctionCallInputWithDefaults

`func NewFunctionCallInputWithDefaults() *FunctionCallInput`

NewFunctionCallInputWithDefaults instantiates a new FunctionCallInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRunId

`func (o *FunctionCallInput) GetRunId() string`

GetRunId returns the RunId field if non-nil, zero value otherwise.

### GetRunIdOk

`func (o *FunctionCallInput) GetRunIdOk() (*string, bool)`

GetRunIdOk returns a tuple with the RunId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunId

`func (o *FunctionCallInput) SetRunId(v string)`

SetRunId sets RunId field to given value.


### GetCallId

`func (o *FunctionCallInput) GetCallId() string`

GetCallId returns the CallId field if non-nil, zero value otherwise.

### GetCallIdOk

`func (o *FunctionCallInput) GetCallIdOk() (*string, bool)`

GetCallIdOk returns a tuple with the CallId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCallId

`func (o *FunctionCallInput) SetCallId(v string)`

SetCallId sets CallId field to given value.


### GetSpec

`func (o *FunctionCallInput) GetSpec() FunctionCallSpecInput`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *FunctionCallInput) GetSpecOk() (*FunctionCallSpecInput, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *FunctionCallInput) SetSpec(v FunctionCallSpecInput)`

SetSpec sets Spec field to given value.


### GetStatus

`func (o *FunctionCallInput) GetStatus() FunctionCallStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *FunctionCallInput) GetStatusOk() (*FunctionCallStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *FunctionCallInput) SetStatus(v FunctionCallStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *FunctionCallInput) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### SetStatusNil

`func (o *FunctionCallInput) SetStatusNil(b bool)`

 SetStatusNil sets the value for Status to be an explicit nil

### UnsetStatus
`func (o *FunctionCallInput) UnsetStatus()`

UnsetStatus ensures that no value is present for Status, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


