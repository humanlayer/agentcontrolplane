# FunctionCallStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RequestedAt** | Pointer to **NullableTime** |  | [optional] 
**RespondedAt** | Pointer to **NullableTime** |  | [optional] 
**Approved** | Pointer to **NullableBool** |  | [optional] 
**Comment** | Pointer to **NullableString** |  | [optional] 
**UserInfo** | Pointer to **map[string]interface{}** |  | [optional] 
**SlackContext** | Pointer to **map[string]interface{}** |  | [optional] 
**RejectOptionName** | Pointer to **NullableString** |  | [optional] 
**SlackMessageTs** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewFunctionCallStatus

`func NewFunctionCallStatus() *FunctionCallStatus`

NewFunctionCallStatus instantiates a new FunctionCallStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFunctionCallStatusWithDefaults

`func NewFunctionCallStatusWithDefaults() *FunctionCallStatus`

NewFunctionCallStatusWithDefaults instantiates a new FunctionCallStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRequestedAt

`func (o *FunctionCallStatus) GetRequestedAt() time.Time`

GetRequestedAt returns the RequestedAt field if non-nil, zero value otherwise.

### GetRequestedAtOk

`func (o *FunctionCallStatus) GetRequestedAtOk() (*time.Time, bool)`

GetRequestedAtOk returns a tuple with the RequestedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequestedAt

`func (o *FunctionCallStatus) SetRequestedAt(v time.Time)`

SetRequestedAt sets RequestedAt field to given value.

### HasRequestedAt

`func (o *FunctionCallStatus) HasRequestedAt() bool`

HasRequestedAt returns a boolean if a field has been set.

### SetRequestedAtNil

`func (o *FunctionCallStatus) SetRequestedAtNil(b bool)`

 SetRequestedAtNil sets the value for RequestedAt to be an explicit nil

### UnsetRequestedAt
`func (o *FunctionCallStatus) UnsetRequestedAt()`

UnsetRequestedAt ensures that no value is present for RequestedAt, not even an explicit nil
### GetRespondedAt

`func (o *FunctionCallStatus) GetRespondedAt() time.Time`

GetRespondedAt returns the RespondedAt field if non-nil, zero value otherwise.

### GetRespondedAtOk

`func (o *FunctionCallStatus) GetRespondedAtOk() (*time.Time, bool)`

GetRespondedAtOk returns a tuple with the RespondedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRespondedAt

`func (o *FunctionCallStatus) SetRespondedAt(v time.Time)`

SetRespondedAt sets RespondedAt field to given value.

### HasRespondedAt

`func (o *FunctionCallStatus) HasRespondedAt() bool`

HasRespondedAt returns a boolean if a field has been set.

### SetRespondedAtNil

`func (o *FunctionCallStatus) SetRespondedAtNil(b bool)`

 SetRespondedAtNil sets the value for RespondedAt to be an explicit nil

### UnsetRespondedAt
`func (o *FunctionCallStatus) UnsetRespondedAt()`

UnsetRespondedAt ensures that no value is present for RespondedAt, not even an explicit nil
### GetApproved

`func (o *FunctionCallStatus) GetApproved() bool`

GetApproved returns the Approved field if non-nil, zero value otherwise.

### GetApprovedOk

`func (o *FunctionCallStatus) GetApprovedOk() (*bool, bool)`

GetApprovedOk returns a tuple with the Approved field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApproved

`func (o *FunctionCallStatus) SetApproved(v bool)`

SetApproved sets Approved field to given value.

### HasApproved

`func (o *FunctionCallStatus) HasApproved() bool`

HasApproved returns a boolean if a field has been set.

### SetApprovedNil

`func (o *FunctionCallStatus) SetApprovedNil(b bool)`

 SetApprovedNil sets the value for Approved to be an explicit nil

### UnsetApproved
`func (o *FunctionCallStatus) UnsetApproved()`

UnsetApproved ensures that no value is present for Approved, not even an explicit nil
### GetComment

`func (o *FunctionCallStatus) GetComment() string`

GetComment returns the Comment field if non-nil, zero value otherwise.

### GetCommentOk

`func (o *FunctionCallStatus) GetCommentOk() (*string, bool)`

GetCommentOk returns a tuple with the Comment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComment

`func (o *FunctionCallStatus) SetComment(v string)`

SetComment sets Comment field to given value.

### HasComment

`func (o *FunctionCallStatus) HasComment() bool`

HasComment returns a boolean if a field has been set.

### SetCommentNil

`func (o *FunctionCallStatus) SetCommentNil(b bool)`

 SetCommentNil sets the value for Comment to be an explicit nil

### UnsetComment
`func (o *FunctionCallStatus) UnsetComment()`

UnsetComment ensures that no value is present for Comment, not even an explicit nil
### GetUserInfo

`func (o *FunctionCallStatus) GetUserInfo() map[string]interface{}`

GetUserInfo returns the UserInfo field if non-nil, zero value otherwise.

### GetUserInfoOk

`func (o *FunctionCallStatus) GetUserInfoOk() (*map[string]interface{}, bool)`

GetUserInfoOk returns a tuple with the UserInfo field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserInfo

`func (o *FunctionCallStatus) SetUserInfo(v map[string]interface{})`

SetUserInfo sets UserInfo field to given value.

### HasUserInfo

`func (o *FunctionCallStatus) HasUserInfo() bool`

HasUserInfo returns a boolean if a field has been set.

### SetUserInfoNil

`func (o *FunctionCallStatus) SetUserInfoNil(b bool)`

 SetUserInfoNil sets the value for UserInfo to be an explicit nil

### UnsetUserInfo
`func (o *FunctionCallStatus) UnsetUserInfo()`

UnsetUserInfo ensures that no value is present for UserInfo, not even an explicit nil
### GetSlackContext

`func (o *FunctionCallStatus) GetSlackContext() map[string]interface{}`

GetSlackContext returns the SlackContext field if non-nil, zero value otherwise.

### GetSlackContextOk

`func (o *FunctionCallStatus) GetSlackContextOk() (*map[string]interface{}, bool)`

GetSlackContextOk returns a tuple with the SlackContext field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSlackContext

`func (o *FunctionCallStatus) SetSlackContext(v map[string]interface{})`

SetSlackContext sets SlackContext field to given value.

### HasSlackContext

`func (o *FunctionCallStatus) HasSlackContext() bool`

HasSlackContext returns a boolean if a field has been set.

### SetSlackContextNil

`func (o *FunctionCallStatus) SetSlackContextNil(b bool)`

 SetSlackContextNil sets the value for SlackContext to be an explicit nil

### UnsetSlackContext
`func (o *FunctionCallStatus) UnsetSlackContext()`

UnsetSlackContext ensures that no value is present for SlackContext, not even an explicit nil
### GetRejectOptionName

`func (o *FunctionCallStatus) GetRejectOptionName() string`

GetRejectOptionName returns the RejectOptionName field if non-nil, zero value otherwise.

### GetRejectOptionNameOk

`func (o *FunctionCallStatus) GetRejectOptionNameOk() (*string, bool)`

GetRejectOptionNameOk returns a tuple with the RejectOptionName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRejectOptionName

`func (o *FunctionCallStatus) SetRejectOptionName(v string)`

SetRejectOptionName sets RejectOptionName field to given value.

### HasRejectOptionName

`func (o *FunctionCallStatus) HasRejectOptionName() bool`

HasRejectOptionName returns a boolean if a field has been set.

### SetRejectOptionNameNil

`func (o *FunctionCallStatus) SetRejectOptionNameNil(b bool)`

 SetRejectOptionNameNil sets the value for RejectOptionName to be an explicit nil

### UnsetRejectOptionName
`func (o *FunctionCallStatus) UnsetRejectOptionName()`

UnsetRejectOptionName ensures that no value is present for RejectOptionName, not even an explicit nil
### GetSlackMessageTs

`func (o *FunctionCallStatus) GetSlackMessageTs() string`

GetSlackMessageTs returns the SlackMessageTs field if non-nil, zero value otherwise.

### GetSlackMessageTsOk

`func (o *FunctionCallStatus) GetSlackMessageTsOk() (*string, bool)`

GetSlackMessageTsOk returns a tuple with the SlackMessageTs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSlackMessageTs

`func (o *FunctionCallStatus) SetSlackMessageTs(v string)`

SetSlackMessageTs sets SlackMessageTs field to given value.

### HasSlackMessageTs

`func (o *FunctionCallStatus) HasSlackMessageTs() bool`

HasSlackMessageTs returns a boolean if a field has been set.

### SetSlackMessageTsNil

`func (o *FunctionCallStatus) SetSlackMessageTsNil(b bool)`

 SetSlackMessageTsNil sets the value for SlackMessageTs to be an explicit nil

### UnsetSlackMessageTs
`func (o *FunctionCallStatus) UnsetSlackMessageTs()`

UnsetSlackMessageTs ensures that no value is present for SlackMessageTs, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


