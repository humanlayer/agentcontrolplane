# SMSContactChannel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PhoneNumber** | **string** |  | 
**ContextAboutUser** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewSMSContactChannel

`func NewSMSContactChannel(phoneNumber string, ) *SMSContactChannel`

NewSMSContactChannel instantiates a new SMSContactChannel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSMSContactChannelWithDefaults

`func NewSMSContactChannelWithDefaults() *SMSContactChannel`

NewSMSContactChannelWithDefaults instantiates a new SMSContactChannel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPhoneNumber

`func (o *SMSContactChannel) GetPhoneNumber() string`

GetPhoneNumber returns the PhoneNumber field if non-nil, zero value otherwise.

### GetPhoneNumberOk

`func (o *SMSContactChannel) GetPhoneNumberOk() (*string, bool)`

GetPhoneNumberOk returns a tuple with the PhoneNumber field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhoneNumber

`func (o *SMSContactChannel) SetPhoneNumber(v string)`

SetPhoneNumber sets PhoneNumber field to given value.


### GetContextAboutUser

`func (o *SMSContactChannel) GetContextAboutUser() string`

GetContextAboutUser returns the ContextAboutUser field if non-nil, zero value otherwise.

### GetContextAboutUserOk

`func (o *SMSContactChannel) GetContextAboutUserOk() (*string, bool)`

GetContextAboutUserOk returns a tuple with the ContextAboutUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContextAboutUser

`func (o *SMSContactChannel) SetContextAboutUser(v string)`

SetContextAboutUser sets ContextAboutUser field to given value.

### HasContextAboutUser

`func (o *SMSContactChannel) HasContextAboutUser() bool`

HasContextAboutUser returns a boolean if a field has been set.

### SetContextAboutUserNil

`func (o *SMSContactChannel) SetContextAboutUserNil(b bool)`

 SetContextAboutUserNil sets the value for ContextAboutUser to be an explicit nil

### UnsetContextAboutUser
`func (o *SMSContactChannel) UnsetContextAboutUser()`

UnsetContextAboutUser ensures that no value is present for ContextAboutUser, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


