# SlackContactChannelOutput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ChannelOrUserId** | **string** |  | 
**ContextAboutChannelOrUser** | Pointer to **NullableString** |  | [optional] 
**AllowedResponderIds** | Pointer to **[]string** |  | [optional] 
**ExperimentalSlackBlocks** | Pointer to **NullableBool** |  | [optional] 
**ThreadTs** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewSlackContactChannelOutput

`func NewSlackContactChannelOutput(channelOrUserId string, ) *SlackContactChannelOutput`

NewSlackContactChannelOutput instantiates a new SlackContactChannelOutput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSlackContactChannelOutputWithDefaults

`func NewSlackContactChannelOutputWithDefaults() *SlackContactChannelOutput`

NewSlackContactChannelOutputWithDefaults instantiates a new SlackContactChannelOutput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetChannelOrUserId

`func (o *SlackContactChannelOutput) GetChannelOrUserId() string`

GetChannelOrUserId returns the ChannelOrUserId field if non-nil, zero value otherwise.

### GetChannelOrUserIdOk

`func (o *SlackContactChannelOutput) GetChannelOrUserIdOk() (*string, bool)`

GetChannelOrUserIdOk returns a tuple with the ChannelOrUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannelOrUserId

`func (o *SlackContactChannelOutput) SetChannelOrUserId(v string)`

SetChannelOrUserId sets ChannelOrUserId field to given value.


### GetContextAboutChannelOrUser

`func (o *SlackContactChannelOutput) GetContextAboutChannelOrUser() string`

GetContextAboutChannelOrUser returns the ContextAboutChannelOrUser field if non-nil, zero value otherwise.

### GetContextAboutChannelOrUserOk

`func (o *SlackContactChannelOutput) GetContextAboutChannelOrUserOk() (*string, bool)`

GetContextAboutChannelOrUserOk returns a tuple with the ContextAboutChannelOrUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContextAboutChannelOrUser

`func (o *SlackContactChannelOutput) SetContextAboutChannelOrUser(v string)`

SetContextAboutChannelOrUser sets ContextAboutChannelOrUser field to given value.

### HasContextAboutChannelOrUser

`func (o *SlackContactChannelOutput) HasContextAboutChannelOrUser() bool`

HasContextAboutChannelOrUser returns a boolean if a field has been set.

### SetContextAboutChannelOrUserNil

`func (o *SlackContactChannelOutput) SetContextAboutChannelOrUserNil(b bool)`

 SetContextAboutChannelOrUserNil sets the value for ContextAboutChannelOrUser to be an explicit nil

### UnsetContextAboutChannelOrUser
`func (o *SlackContactChannelOutput) UnsetContextAboutChannelOrUser()`

UnsetContextAboutChannelOrUser ensures that no value is present for ContextAboutChannelOrUser, not even an explicit nil
### GetAllowedResponderIds

`func (o *SlackContactChannelOutput) GetAllowedResponderIds() []string`

GetAllowedResponderIds returns the AllowedResponderIds field if non-nil, zero value otherwise.

### GetAllowedResponderIdsOk

`func (o *SlackContactChannelOutput) GetAllowedResponderIdsOk() (*[]string, bool)`

GetAllowedResponderIdsOk returns a tuple with the AllowedResponderIds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedResponderIds

`func (o *SlackContactChannelOutput) SetAllowedResponderIds(v []string)`

SetAllowedResponderIds sets AllowedResponderIds field to given value.

### HasAllowedResponderIds

`func (o *SlackContactChannelOutput) HasAllowedResponderIds() bool`

HasAllowedResponderIds returns a boolean if a field has been set.

### SetAllowedResponderIdsNil

`func (o *SlackContactChannelOutput) SetAllowedResponderIdsNil(b bool)`

 SetAllowedResponderIdsNil sets the value for AllowedResponderIds to be an explicit nil

### UnsetAllowedResponderIds
`func (o *SlackContactChannelOutput) UnsetAllowedResponderIds()`

UnsetAllowedResponderIds ensures that no value is present for AllowedResponderIds, not even an explicit nil
### GetExperimentalSlackBlocks

`func (o *SlackContactChannelOutput) GetExperimentalSlackBlocks() bool`

GetExperimentalSlackBlocks returns the ExperimentalSlackBlocks field if non-nil, zero value otherwise.

### GetExperimentalSlackBlocksOk

`func (o *SlackContactChannelOutput) GetExperimentalSlackBlocksOk() (*bool, bool)`

GetExperimentalSlackBlocksOk returns a tuple with the ExperimentalSlackBlocks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExperimentalSlackBlocks

`func (o *SlackContactChannelOutput) SetExperimentalSlackBlocks(v bool)`

SetExperimentalSlackBlocks sets ExperimentalSlackBlocks field to given value.

### HasExperimentalSlackBlocks

`func (o *SlackContactChannelOutput) HasExperimentalSlackBlocks() bool`

HasExperimentalSlackBlocks returns a boolean if a field has been set.

### SetExperimentalSlackBlocksNil

`func (o *SlackContactChannelOutput) SetExperimentalSlackBlocksNil(b bool)`

 SetExperimentalSlackBlocksNil sets the value for ExperimentalSlackBlocks to be an explicit nil

### UnsetExperimentalSlackBlocks
`func (o *SlackContactChannelOutput) UnsetExperimentalSlackBlocks()`

UnsetExperimentalSlackBlocks ensures that no value is present for ExperimentalSlackBlocks, not even an explicit nil
### GetThreadTs

`func (o *SlackContactChannelOutput) GetThreadTs() string`

GetThreadTs returns the ThreadTs field if non-nil, zero value otherwise.

### GetThreadTsOk

`func (o *SlackContactChannelOutput) GetThreadTsOk() (*string, bool)`

GetThreadTsOk returns a tuple with the ThreadTs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThreadTs

`func (o *SlackContactChannelOutput) SetThreadTs(v string)`

SetThreadTs sets ThreadTs field to given value.

### HasThreadTs

`func (o *SlackContactChannelOutput) HasThreadTs() bool`

HasThreadTs returns a boolean if a field has been set.

### SetThreadTsNil

`func (o *SlackContactChannelOutput) SetThreadTsNil(b bool)`

 SetThreadTsNil sets the value for ThreadTs to be an explicit nil

### UnsetThreadTs
`func (o *SlackContactChannelOutput) UnsetThreadTs()`

UnsetThreadTs ensures that no value is present for ThreadTs, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


