# FunctionCallOutput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RunId** | **string** |  | 
**CallId** | **string** |  | 
**Spec** | [**FunctionCallSpecOutput**](FunctionCallSpecOutput.md) |  | 
**Status** | Pointer to [**NullableFunctionCallStatus**](FunctionCallStatus.md) |  | [optional] 

## Methods

### NewFunctionCallOutput

`func NewFunctionCallOutput(runId string, callId string, spec FunctionCallSpecOutput, ) *FunctionCallOutput`

NewFunctionCallOutput instantiates a new FunctionCallOutput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFunctionCallOutputWithDefaults

`func NewFunctionCallOutputWithDefaults() *FunctionCallOutput`

NewFunctionCallOutputWithDefaults instantiates a new FunctionCallOutput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRunId

`func (o *FunctionCallOutput) GetRunId() string`

GetRunId returns the RunId field if non-nil, zero value otherwise.

### GetRunIdOk

`func (o *FunctionCallOutput) GetRunIdOk() (*string, bool)`

GetRunIdOk returns a tuple with the RunId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunId

`func (o *FunctionCallOutput) SetRunId(v string)`

SetRunId sets RunId field to given value.


### GetCallId

`func (o *FunctionCallOutput) GetCallId() string`

GetCallId returns the CallId field if non-nil, zero value otherwise.

### GetCallIdOk

`func (o *FunctionCallOutput) GetCallIdOk() (*string, bool)`

GetCallIdOk returns a tuple with the CallId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCallId

`func (o *FunctionCallOutput) SetCallId(v string)`

SetCallId sets CallId field to given value.


### GetSpec

`func (o *FunctionCallOutput) GetSpec() FunctionCallSpecOutput`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *FunctionCallOutput) GetSpecOk() (*FunctionCallSpecOutput, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *FunctionCallOutput) SetSpec(v FunctionCallSpecOutput)`

SetSpec sets Spec field to given value.


### GetStatus

`func (o *FunctionCallOutput) GetStatus() FunctionCallStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *FunctionCallOutput) GetStatusOk() (*FunctionCallStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *FunctionCallOutput) SetStatus(v FunctionCallStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *FunctionCallOutput) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### SetStatusNil

`func (o *FunctionCallOutput) SetStatusNil(b bool)`

 SetStatusNil sets the value for Status to be an explicit nil

### UnsetStatus
`func (o *FunctionCallOutput) UnsetStatus()`

UnsetStatus ensures that no value is present for Status, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


