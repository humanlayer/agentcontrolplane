# HumanContactInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RunId** | **string** |  | 
**CallId** | **string** |  | 
**Spec** | [**HumanContactSpecInput**](HumanContactSpecInput.md) |  | 
**Status** | Pointer to [**NullableHumanContactStatus**](HumanContactStatus.md) |  | [optional] 

## Methods

### NewHumanContactInput

`func NewHumanContactInput(runId string, callId string, spec HumanContactSpecInput, ) *HumanContactInput`

NewHumanContactInput instantiates a new HumanContactInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHumanContactInputWithDefaults

`func NewHumanContactInputWithDefaults() *HumanContactInput`

NewHumanContactInputWithDefaults instantiates a new HumanContactInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRunId

`func (o *HumanContactInput) GetRunId() string`

GetRunId returns the RunId field if non-nil, zero value otherwise.

### GetRunIdOk

`func (o *HumanContactInput) GetRunIdOk() (*string, bool)`

GetRunIdOk returns a tuple with the RunId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunId

`func (o *HumanContactInput) SetRunId(v string)`

SetRunId sets RunId field to given value.


### GetCallId

`func (o *HumanContactInput) GetCallId() string`

GetCallId returns the CallId field if non-nil, zero value otherwise.

### GetCallIdOk

`func (o *HumanContactInput) GetCallIdOk() (*string, bool)`

GetCallIdOk returns a tuple with the CallId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCallId

`func (o *HumanContactInput) SetCallId(v string)`

SetCallId sets CallId field to given value.


### GetSpec

`func (o *HumanContactInput) GetSpec() HumanContactSpecInput`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *HumanContactInput) GetSpecOk() (*HumanContactSpecInput, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *HumanContactInput) SetSpec(v HumanContactSpecInput)`

SetSpec sets Spec field to given value.


### GetStatus

`func (o *HumanContactInput) GetStatus() HumanContactStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *HumanContactInput) GetStatusOk() (*HumanContactStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *HumanContactInput) SetStatus(v HumanContactStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *HumanContactInput) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### SetStatusNil

`func (o *HumanContactInput) SetStatusNil(b bool)`

 SetStatusNil sets the value for Status to be an explicit nil

### UnsetStatus
`func (o *HumanContactInput) UnsetStatus()`

UnsetStatus ensures that no value is present for Status, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


