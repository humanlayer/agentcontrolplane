# SlackContactChannelInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ChannelOrUserId** | **string** |  | 
**ContextAboutChannelOrUser** | Pointer to **NullableString** |  | [optional] 
**BotToken** | Pointer to **NullableString** |  | [optional] 
**AllowedResponderIds** | Pointer to **[]string** |  | [optional] 
**ExperimentalSlackBlocks** | Pointer to **NullableBool** |  | [optional] 
**ThreadTs** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewSlackContactChannelInput

`func NewSlackContactChannelInput(channelOrUserId string, ) *SlackContactChannelInput`

NewSlackContactChannelInput instantiates a new SlackContactChannelInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSlackContactChannelInputWithDefaults

`func NewSlackContactChannelInputWithDefaults() *SlackContactChannelInput`

NewSlackContactChannelInputWithDefaults instantiates a new SlackContactChannelInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetChannelOrUserId

`func (o *SlackContactChannelInput) GetChannelOrUserId() string`

GetChannelOrUserId returns the ChannelOrUserId field if non-nil, zero value otherwise.

### GetChannelOrUserIdOk

`func (o *SlackContactChannelInput) GetChannelOrUserIdOk() (*string, bool)`

GetChannelOrUserIdOk returns a tuple with the ChannelOrUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannelOrUserId

`func (o *SlackContactChannelInput) SetChannelOrUserId(v string)`

SetChannelOrUserId sets ChannelOrUserId field to given value.


### GetContextAboutChannelOrUser

`func (o *SlackContactChannelInput) GetContextAboutChannelOrUser() string`

GetContextAboutChannelOrUser returns the ContextAboutChannelOrUser field if non-nil, zero value otherwise.

### GetContextAboutChannelOrUserOk

`func (o *SlackContactChannelInput) GetContextAboutChannelOrUserOk() (*string, bool)`

GetContextAboutChannelOrUserOk returns a tuple with the ContextAboutChannelOrUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContextAboutChannelOrUser

`func (o *SlackContactChannelInput) SetContextAboutChannelOrUser(v string)`

SetContextAboutChannelOrUser sets ContextAboutChannelOrUser field to given value.

### HasContextAboutChannelOrUser

`func (o *SlackContactChannelInput) HasContextAboutChannelOrUser() bool`

HasContextAboutChannelOrUser returns a boolean if a field has been set.

### SetContextAboutChannelOrUserNil

`func (o *SlackContactChannelInput) SetContextAboutChannelOrUserNil(b bool)`

 SetContextAboutChannelOrUserNil sets the value for ContextAboutChannelOrUser to be an explicit nil

### UnsetContextAboutChannelOrUser
`func (o *SlackContactChannelInput) UnsetContextAboutChannelOrUser()`

UnsetContextAboutChannelOrUser ensures that no value is present for ContextAboutChannelOrUser, not even an explicit nil
### GetBotToken

`func (o *SlackContactChannelInput) GetBotToken() string`

GetBotToken returns the BotToken field if non-nil, zero value otherwise.

### GetBotTokenOk

`func (o *SlackContactChannelInput) GetBotTokenOk() (*string, bool)`

GetBotTokenOk returns a tuple with the BotToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBotToken

`func (o *SlackContactChannelInput) SetBotToken(v string)`

SetBotToken sets BotToken field to given value.

### HasBotToken

`func (o *SlackContactChannelInput) HasBotToken() bool`

HasBotToken returns a boolean if a field has been set.

### SetBotTokenNil

`func (o *SlackContactChannelInput) SetBotTokenNil(b bool)`

 SetBotTokenNil sets the value for BotToken to be an explicit nil

### UnsetBotToken
`func (o *SlackContactChannelInput) UnsetBotToken()`

UnsetBotToken ensures that no value is present for BotToken, not even an explicit nil
### GetAllowedResponderIds

`func (o *SlackContactChannelInput) GetAllowedResponderIds() []string`

GetAllowedResponderIds returns the AllowedResponderIds field if non-nil, zero value otherwise.

### GetAllowedResponderIdsOk

`func (o *SlackContactChannelInput) GetAllowedResponderIdsOk() (*[]string, bool)`

GetAllowedResponderIdsOk returns a tuple with the AllowedResponderIds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedResponderIds

`func (o *SlackContactChannelInput) SetAllowedResponderIds(v []string)`

SetAllowedResponderIds sets AllowedResponderIds field to given value.

### HasAllowedResponderIds

`func (o *SlackContactChannelInput) HasAllowedResponderIds() bool`

HasAllowedResponderIds returns a boolean if a field has been set.

### SetAllowedResponderIdsNil

`func (o *SlackContactChannelInput) SetAllowedResponderIdsNil(b bool)`

 SetAllowedResponderIdsNil sets the value for AllowedResponderIds to be an explicit nil

### UnsetAllowedResponderIds
`func (o *SlackContactChannelInput) UnsetAllowedResponderIds()`

UnsetAllowedResponderIds ensures that no value is present for AllowedResponderIds, not even an explicit nil
### GetExperimentalSlackBlocks

`func (o *SlackContactChannelInput) GetExperimentalSlackBlocks() bool`

GetExperimentalSlackBlocks returns the ExperimentalSlackBlocks field if non-nil, zero value otherwise.

### GetExperimentalSlackBlocksOk

`func (o *SlackContactChannelInput) GetExperimentalSlackBlocksOk() (*bool, bool)`

GetExperimentalSlackBlocksOk returns a tuple with the ExperimentalSlackBlocks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExperimentalSlackBlocks

`func (o *SlackContactChannelInput) SetExperimentalSlackBlocks(v bool)`

SetExperimentalSlackBlocks sets ExperimentalSlackBlocks field to given value.

### HasExperimentalSlackBlocks

`func (o *SlackContactChannelInput) HasExperimentalSlackBlocks() bool`

HasExperimentalSlackBlocks returns a boolean if a field has been set.

### SetExperimentalSlackBlocksNil

`func (o *SlackContactChannelInput) SetExperimentalSlackBlocksNil(b bool)`

 SetExperimentalSlackBlocksNil sets the value for ExperimentalSlackBlocks to be an explicit nil

### UnsetExperimentalSlackBlocks
`func (o *SlackContactChannelInput) UnsetExperimentalSlackBlocks()`

UnsetExperimentalSlackBlocks ensures that no value is present for ExperimentalSlackBlocks, not even an explicit nil
### GetThreadTs

`func (o *SlackContactChannelInput) GetThreadTs() string`

GetThreadTs returns the ThreadTs field if non-nil, zero value otherwise.

### GetThreadTsOk

`func (o *SlackContactChannelInput) GetThreadTsOk() (*string, bool)`

GetThreadTsOk returns a tuple with the ThreadTs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThreadTs

`func (o *SlackContactChannelInput) SetThreadTs(v string)`

SetThreadTs sets ThreadTs field to given value.

### HasThreadTs

`func (o *SlackContactChannelInput) HasThreadTs() bool`

HasThreadTs returns a boolean if a field has been set.

### SetThreadTsNil

`func (o *SlackContactChannelInput) SetThreadTsNil(b bool)`

 SetThreadTsNil sets the value for ThreadTs to be an explicit nil

### UnsetThreadTs
`func (o *SlackContactChannelInput) UnsetThreadTs()`

UnsetThreadTs ensures that no value is present for ThreadTs, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


