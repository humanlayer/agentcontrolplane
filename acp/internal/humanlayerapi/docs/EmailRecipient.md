# EmailRecipient

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** |  | 
**ContextAboutUser** | Pointer to **NullableString** |  | [optional] 
**Field** | Pointer to [**NullableField**](Field.md) |  | [optional] 

## Methods

### NewEmailRecipient

`func NewEmailRecipient(address string, ) *EmailRecipient`

NewEmailRecipient instantiates a new EmailRecipient object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEmailRecipientWithDefaults

`func NewEmailRecipientWithDefaults() *EmailRecipient`

NewEmailRecipientWithDefaults instantiates a new EmailRecipient object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *EmailRecipient) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *EmailRecipient) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *EmailRecipient) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetContextAboutUser

`func (o *EmailRecipient) GetContextAboutUser() string`

GetContextAboutUser returns the ContextAboutUser field if non-nil, zero value otherwise.

### GetContextAboutUserOk

`func (o *EmailRecipient) GetContextAboutUserOk() (*string, bool)`

GetContextAboutUserOk returns a tuple with the ContextAboutUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContextAboutUser

`func (o *EmailRecipient) SetContextAboutUser(v string)`

SetContextAboutUser sets ContextAboutUser field to given value.

### HasContextAboutUser

`func (o *EmailRecipient) HasContextAboutUser() bool`

HasContextAboutUser returns a boolean if a field has been set.

### SetContextAboutUserNil

`func (o *EmailRecipient) SetContextAboutUserNil(b bool)`

 SetContextAboutUserNil sets the value for ContextAboutUser to be an explicit nil

### UnsetContextAboutUser
`func (o *EmailRecipient) UnsetContextAboutUser()`

UnsetContextAboutUser ensures that no value is present for ContextAboutUser, not even an explicit nil
### GetField

`func (o *EmailRecipient) GetField() Field`

GetField returns the Field field if non-nil, zero value otherwise.

### GetFieldOk

`func (o *EmailRecipient) GetFieldOk() (*Field, bool)`

GetFieldOk returns a tuple with the Field field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetField

`func (o *EmailRecipient) SetField(v Field)`

SetField sets Field field to given value.

### HasField

`func (o *EmailRecipient) HasField() bool`

HasField returns a boolean if a field has been set.

### SetFieldNil

`func (o *EmailRecipient) SetFieldNil(b bool)`

 SetFieldNil sets the value for Field to be an explicit nil

### UnsetField
`func (o *EmailRecipient) UnsetField()`

UnsetField ensures that no value is present for Field, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


