# HumanContactStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RequestedAt** | Pointer to **NullableTime** |  | [optional] 
**RespondedAt** | Pointer to **NullableTime** |  | [optional] 
**Response** | Pointer to **NullableString** |  | [optional] 
**ResponseOptionName** | Pointer to **NullableString** |  | [optional] 
**SlackMessageTs** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewHumanContactStatus

`func NewHumanContactStatus() *HumanContactStatus`

NewHumanContactStatus instantiates a new HumanContactStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHumanContactStatusWithDefaults

`func NewHumanContactStatusWithDefaults() *HumanContactStatus`

NewHumanContactStatusWithDefaults instantiates a new HumanContactStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRequestedAt

`func (o *HumanContactStatus) GetRequestedAt() time.Time`

GetRequestedAt returns the RequestedAt field if non-nil, zero value otherwise.

### GetRequestedAtOk

`func (o *HumanContactStatus) GetRequestedAtOk() (*time.Time, bool)`

GetRequestedAtOk returns a tuple with the RequestedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequestedAt

`func (o *HumanContactStatus) SetRequestedAt(v time.Time)`

SetRequestedAt sets RequestedAt field to given value.

### HasRequestedAt

`func (o *HumanContactStatus) HasRequestedAt() bool`

HasRequestedAt returns a boolean if a field has been set.

### SetRequestedAtNil

`func (o *HumanContactStatus) SetRequestedAtNil(b bool)`

 SetRequestedAtNil sets the value for RequestedAt to be an explicit nil

### UnsetRequestedAt
`func (o *HumanContactStatus) UnsetRequestedAt()`

UnsetRequestedAt ensures that no value is present for RequestedAt, not even an explicit nil
### GetRespondedAt

`func (o *HumanContactStatus) GetRespondedAt() time.Time`

GetRespondedAt returns the RespondedAt field if non-nil, zero value otherwise.

### GetRespondedAtOk

`func (o *HumanContactStatus) GetRespondedAtOk() (*time.Time, bool)`

GetRespondedAtOk returns a tuple with the RespondedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRespondedAt

`func (o *HumanContactStatus) SetRespondedAt(v time.Time)`

SetRespondedAt sets RespondedAt field to given value.

### HasRespondedAt

`func (o *HumanContactStatus) HasRespondedAt() bool`

HasRespondedAt returns a boolean if a field has been set.

### SetRespondedAtNil

`func (o *HumanContactStatus) SetRespondedAtNil(b bool)`

 SetRespondedAtNil sets the value for RespondedAt to be an explicit nil

### UnsetRespondedAt
`func (o *HumanContactStatus) UnsetRespondedAt()`

UnsetRespondedAt ensures that no value is present for RespondedAt, not even an explicit nil
### GetResponse

`func (o *HumanContactStatus) GetResponse() string`

GetResponse returns the Response field if non-nil, zero value otherwise.

### GetResponseOk

`func (o *HumanContactStatus) GetResponseOk() (*string, bool)`

GetResponseOk returns a tuple with the Response field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResponse

`func (o *HumanContactStatus) SetResponse(v string)`

SetResponse sets Response field to given value.

### HasResponse

`func (o *HumanContactStatus) HasResponse() bool`

HasResponse returns a boolean if a field has been set.

### SetResponseNil

`func (o *HumanContactStatus) SetResponseNil(b bool)`

 SetResponseNil sets the value for Response to be an explicit nil

### UnsetResponse
`func (o *HumanContactStatus) UnsetResponse()`

UnsetResponse ensures that no value is present for Response, not even an explicit nil
### GetResponseOptionName

`func (o *HumanContactStatus) GetResponseOptionName() string`

GetResponseOptionName returns the ResponseOptionName field if non-nil, zero value otherwise.

### GetResponseOptionNameOk

`func (o *HumanContactStatus) GetResponseOptionNameOk() (*string, bool)`

GetResponseOptionNameOk returns a tuple with the ResponseOptionName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResponseOptionName

`func (o *HumanContactStatus) SetResponseOptionName(v string)`

SetResponseOptionName sets ResponseOptionName field to given value.

### HasResponseOptionName

`func (o *HumanContactStatus) HasResponseOptionName() bool`

HasResponseOptionName returns a boolean if a field has been set.

### SetResponseOptionNameNil

`func (o *HumanContactStatus) SetResponseOptionNameNil(b bool)`

 SetResponseOptionNameNil sets the value for ResponseOptionName to be an explicit nil

### UnsetResponseOptionName
`func (o *HumanContactStatus) UnsetResponseOptionName()`

UnsetResponseOptionName ensures that no value is present for ResponseOptionName, not even an explicit nil
### GetSlackMessageTs

`func (o *HumanContactStatus) GetSlackMessageTs() string`

GetSlackMessageTs returns the SlackMessageTs field if non-nil, zero value otherwise.

### GetSlackMessageTsOk

`func (o *HumanContactStatus) GetSlackMessageTsOk() (*string, bool)`

GetSlackMessageTsOk returns a tuple with the SlackMessageTs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSlackMessageTs

`func (o *HumanContactStatus) SetSlackMessageTs(v string)`

SetSlackMessageTs sets SlackMessageTs field to given value.

### HasSlackMessageTs

`func (o *HumanContactStatus) HasSlackMessageTs() bool`

HasSlackMessageTs returns a boolean if a field has been set.

### SetSlackMessageTsNil

`func (o *HumanContactStatus) SetSlackMessageTsNil(b bool)`

 SetSlackMessageTsNil sets the value for SlackMessageTs to be an explicit nil

### UnsetSlackMessageTs
`func (o *HumanContactStatus) UnsetSlackMessageTs()`

UnsetSlackMessageTs ensures that no value is present for SlackMessageTs, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


