# HumanContactSpecInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Msg** | **string** |  | 
**Subject** | Pointer to **NullableString** |  | [optional] 
**Channel** | Pointer to [**NullableContactChannelInput**](ContactChannelInput.md) |  | [optional] 
**ResponseOptions** | Pointer to [**[]ResponseOption**](ResponseOption.md) |  | [optional] 
**State** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewHumanContactSpecInput

`func NewHumanContactSpecInput(msg string, ) *HumanContactSpecInput`

NewHumanContactSpecInput instantiates a new HumanContactSpecInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHumanContactSpecInputWithDefaults

`func NewHumanContactSpecInputWithDefaults() *HumanContactSpecInput`

NewHumanContactSpecInputWithDefaults instantiates a new HumanContactSpecInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMsg

`func (o *HumanContactSpecInput) GetMsg() string`

GetMsg returns the Msg field if non-nil, zero value otherwise.

### GetMsgOk

`func (o *HumanContactSpecInput) GetMsgOk() (*string, bool)`

GetMsgOk returns a tuple with the Msg field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMsg

`func (o *HumanContactSpecInput) SetMsg(v string)`

SetMsg sets Msg field to given value.


### GetSubject

`func (o *HumanContactSpecInput) GetSubject() string`

GetSubject returns the Subject field if non-nil, zero value otherwise.

### GetSubjectOk

`func (o *HumanContactSpecInput) GetSubjectOk() (*string, bool)`

GetSubjectOk returns a tuple with the Subject field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubject

`func (o *HumanContactSpecInput) SetSubject(v string)`

SetSubject sets Subject field to given value.

### HasSubject

`func (o *HumanContactSpecInput) HasSubject() bool`

HasSubject returns a boolean if a field has been set.

### SetSubjectNil

`func (o *HumanContactSpecInput) SetSubjectNil(b bool)`

 SetSubjectNil sets the value for Subject to be an explicit nil

### UnsetSubject
`func (o *HumanContactSpecInput) UnsetSubject()`

UnsetSubject ensures that no value is present for Subject, not even an explicit nil
### GetChannel

`func (o *HumanContactSpecInput) GetChannel() ContactChannelInput`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *HumanContactSpecInput) GetChannelOk() (*ContactChannelInput, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *HumanContactSpecInput) SetChannel(v ContactChannelInput)`

SetChannel sets Channel field to given value.

### HasChannel

`func (o *HumanContactSpecInput) HasChannel() bool`

HasChannel returns a boolean if a field has been set.

### SetChannelNil

`func (o *HumanContactSpecInput) SetChannelNil(b bool)`

 SetChannelNil sets the value for Channel to be an explicit nil

### UnsetChannel
`func (o *HumanContactSpecInput) UnsetChannel()`

UnsetChannel ensures that no value is present for Channel, not even an explicit nil
### GetResponseOptions

`func (o *HumanContactSpecInput) GetResponseOptions() []ResponseOption`

GetResponseOptions returns the ResponseOptions field if non-nil, zero value otherwise.

### GetResponseOptionsOk

`func (o *HumanContactSpecInput) GetResponseOptionsOk() (*[]ResponseOption, bool)`

GetResponseOptionsOk returns a tuple with the ResponseOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResponseOptions

`func (o *HumanContactSpecInput) SetResponseOptions(v []ResponseOption)`

SetResponseOptions sets ResponseOptions field to given value.

### HasResponseOptions

`func (o *HumanContactSpecInput) HasResponseOptions() bool`

HasResponseOptions returns a boolean if a field has been set.

### SetResponseOptionsNil

`func (o *HumanContactSpecInput) SetResponseOptionsNil(b bool)`

 SetResponseOptionsNil sets the value for ResponseOptions to be an explicit nil

### UnsetResponseOptions
`func (o *HumanContactSpecInput) UnsetResponseOptions()`

UnsetResponseOptions ensures that no value is present for ResponseOptions, not even an explicit nil
### GetState

`func (o *HumanContactSpecInput) GetState() map[string]interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *HumanContactSpecInput) GetStateOk() (*map[string]interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *HumanContactSpecInput) SetState(v map[string]interface{})`

SetState sets State field to given value.

### HasState

`func (o *HumanContactSpecInput) HasState() bool`

HasState returns a boolean if a field has been set.

### SetStateNil

`func (o *HumanContactSpecInput) SetStateNil(b bool)`

 SetStateNil sets the value for State to be an explicit nil

### UnsetState
`func (o *HumanContactSpecInput) UnsetState()`

UnsetState ensures that no value is present for State, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


