# FunctionCallSpecInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Fn** | **string** |  | 
**Kwargs** | **map[string]interface{}** |  | 
**Channel** | Pointer to [**NullableContactChannelInput**](ContactChannelInput.md) |  | [optional] 
**RejectOptions** | Pointer to [**[]ResponseOption**](ResponseOption.md) |  | [optional] 
**State** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewFunctionCallSpecInput

`func NewFunctionCallSpecInput(fn string, kwargs map[string]interface{}, ) *FunctionCallSpecInput`

NewFunctionCallSpecInput instantiates a new FunctionCallSpecInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFunctionCallSpecInputWithDefaults

`func NewFunctionCallSpecInputWithDefaults() *FunctionCallSpecInput`

NewFunctionCallSpecInputWithDefaults instantiates a new FunctionCallSpecInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFn

`func (o *FunctionCallSpecInput) GetFn() string`

GetFn returns the Fn field if non-nil, zero value otherwise.

### GetFnOk

`func (o *FunctionCallSpecInput) GetFnOk() (*string, bool)`

GetFnOk returns a tuple with the Fn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFn

`func (o *FunctionCallSpecInput) SetFn(v string)`

SetFn sets Fn field to given value.


### GetKwargs

`func (o *FunctionCallSpecInput) GetKwargs() map[string]interface{}`

GetKwargs returns the Kwargs field if non-nil, zero value otherwise.

### GetKwargsOk

`func (o *FunctionCallSpecInput) GetKwargsOk() (*map[string]interface{}, bool)`

GetKwargsOk returns a tuple with the Kwargs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKwargs

`func (o *FunctionCallSpecInput) SetKwargs(v map[string]interface{})`

SetKwargs sets Kwargs field to given value.


### GetChannel

`func (o *FunctionCallSpecInput) GetChannel() ContactChannelInput`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *FunctionCallSpecInput) GetChannelOk() (*ContactChannelInput, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *FunctionCallSpecInput) SetChannel(v ContactChannelInput)`

SetChannel sets Channel field to given value.

### HasChannel

`func (o *FunctionCallSpecInput) HasChannel() bool`

HasChannel returns a boolean if a field has been set.

### SetChannelNil

`func (o *FunctionCallSpecInput) SetChannelNil(b bool)`

 SetChannelNil sets the value for Channel to be an explicit nil

### UnsetChannel
`func (o *FunctionCallSpecInput) UnsetChannel()`

UnsetChannel ensures that no value is present for Channel, not even an explicit nil
### GetRejectOptions

`func (o *FunctionCallSpecInput) GetRejectOptions() []ResponseOption`

GetRejectOptions returns the RejectOptions field if non-nil, zero value otherwise.

### GetRejectOptionsOk

`func (o *FunctionCallSpecInput) GetRejectOptionsOk() (*[]ResponseOption, bool)`

GetRejectOptionsOk returns a tuple with the RejectOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRejectOptions

`func (o *FunctionCallSpecInput) SetRejectOptions(v []ResponseOption)`

SetRejectOptions sets RejectOptions field to given value.

### HasRejectOptions

`func (o *FunctionCallSpecInput) HasRejectOptions() bool`

HasRejectOptions returns a boolean if a field has been set.

### SetRejectOptionsNil

`func (o *FunctionCallSpecInput) SetRejectOptionsNil(b bool)`

 SetRejectOptionsNil sets the value for RejectOptions to be an explicit nil

### UnsetRejectOptions
`func (o *FunctionCallSpecInput) UnsetRejectOptions()`

UnsetRejectOptions ensures that no value is present for RejectOptions, not even an explicit nil
### GetState

`func (o *FunctionCallSpecInput) GetState() map[string]interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *FunctionCallSpecInput) GetStateOk() (*map[string]interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *FunctionCallSpecInput) SetState(v map[string]interface{})`

SetState sets State field to given value.

### HasState

`func (o *FunctionCallSpecInput) HasState() bool`

HasState returns a boolean if a field has been set.

### SetStateNil

`func (o *FunctionCallSpecInput) SetStateNil(b bool)`

 SetStateNil sets the value for State to be an explicit nil

### UnsetState
`func (o *FunctionCallSpecInput) UnsetState()`

UnsetState ensures that no value is present for State, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


