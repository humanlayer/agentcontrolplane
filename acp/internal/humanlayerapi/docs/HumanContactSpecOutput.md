# HumanContactSpecOutput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Msg** | **string** |  | 
**Subject** | Pointer to **NullableString** |  | [optional] 
**Channel** | Pointer to [**NullableContactChannelOutput**](ContactChannelOutput.md) |  | [optional] 
**ResponseOptions** | Pointer to [**[]ResponseOption**](ResponseOption.md) |  | [optional] 
**State** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewHumanContactSpecOutput

`func NewHumanContactSpecOutput(msg string, ) *HumanContactSpecOutput`

NewHumanContactSpecOutput instantiates a new HumanContactSpecOutput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHumanContactSpecOutputWithDefaults

`func NewHumanContactSpecOutputWithDefaults() *HumanContactSpecOutput`

NewHumanContactSpecOutputWithDefaults instantiates a new HumanContactSpecOutput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMsg

`func (o *HumanContactSpecOutput) GetMsg() string`

GetMsg returns the Msg field if non-nil, zero value otherwise.

### GetMsgOk

`func (o *HumanContactSpecOutput) GetMsgOk() (*string, bool)`

GetMsgOk returns a tuple with the Msg field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMsg

`func (o *HumanContactSpecOutput) SetMsg(v string)`

SetMsg sets Msg field to given value.


### GetSubject

`func (o *HumanContactSpecOutput) GetSubject() string`

GetSubject returns the Subject field if non-nil, zero value otherwise.

### GetSubjectOk

`func (o *HumanContactSpecOutput) GetSubjectOk() (*string, bool)`

GetSubjectOk returns a tuple with the Subject field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubject

`func (o *HumanContactSpecOutput) SetSubject(v string)`

SetSubject sets Subject field to given value.

### HasSubject

`func (o *HumanContactSpecOutput) HasSubject() bool`

HasSubject returns a boolean if a field has been set.

### SetSubjectNil

`func (o *HumanContactSpecOutput) SetSubjectNil(b bool)`

 SetSubjectNil sets the value for Subject to be an explicit nil

### UnsetSubject
`func (o *HumanContactSpecOutput) UnsetSubject()`

UnsetSubject ensures that no value is present for Subject, not even an explicit nil
### GetChannel

`func (o *HumanContactSpecOutput) GetChannel() ContactChannelOutput`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *HumanContactSpecOutput) GetChannelOk() (*ContactChannelOutput, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *HumanContactSpecOutput) SetChannel(v ContactChannelOutput)`

SetChannel sets Channel field to given value.

### HasChannel

`func (o *HumanContactSpecOutput) HasChannel() bool`

HasChannel returns a boolean if a field has been set.

### SetChannelNil

`func (o *HumanContactSpecOutput) SetChannelNil(b bool)`

 SetChannelNil sets the value for Channel to be an explicit nil

### UnsetChannel
`func (o *HumanContactSpecOutput) UnsetChannel()`

UnsetChannel ensures that no value is present for Channel, not even an explicit nil
### GetResponseOptions

`func (o *HumanContactSpecOutput) GetResponseOptions() []ResponseOption`

GetResponseOptions returns the ResponseOptions field if non-nil, zero value otherwise.

### GetResponseOptionsOk

`func (o *HumanContactSpecOutput) GetResponseOptionsOk() (*[]ResponseOption, bool)`

GetResponseOptionsOk returns a tuple with the ResponseOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResponseOptions

`func (o *HumanContactSpecOutput) SetResponseOptions(v []ResponseOption)`

SetResponseOptions sets ResponseOptions field to given value.

### HasResponseOptions

`func (o *HumanContactSpecOutput) HasResponseOptions() bool`

HasResponseOptions returns a boolean if a field has been set.

### SetResponseOptionsNil

`func (o *HumanContactSpecOutput) SetResponseOptionsNil(b bool)`

 SetResponseOptionsNil sets the value for ResponseOptions to be an explicit nil

### UnsetResponseOptions
`func (o *HumanContactSpecOutput) UnsetResponseOptions()`

UnsetResponseOptions ensures that no value is present for ResponseOptions, not even an explicit nil
### GetState

`func (o *HumanContactSpecOutput) GetState() map[string]interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *HumanContactSpecOutput) GetStateOk() (*map[string]interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *HumanContactSpecOutput) SetState(v map[string]interface{})`

SetState sets State field to given value.

### HasState

`func (o *HumanContactSpecOutput) HasState() bool`

HasState returns a boolean if a field has been set.

### SetStateNil

`func (o *HumanContactSpecOutput) SetStateNil(b bool)`

 SetStateNil sets the value for State to be an explicit nil

### UnsetState
`func (o *HumanContactSpecOutput) UnsetState()`

UnsetState ensures that no value is present for State, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


