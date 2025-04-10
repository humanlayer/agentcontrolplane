# EmailContactChannel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** |  | 
**ContextAboutUser** | Pointer to **NullableString** |  | [optional] 
**AdditionalRecipients** | Pointer to [**[]EmailRecipient**](EmailRecipient.md) |  | [optional] 
**ExperimentalSubjectLine** | Pointer to **NullableString** |  | [optional] 
**ExperimentalReferencesMessageId** | Pointer to **NullableString** |  | [optional] 
**ExperimentalInReplyToMessageId** | Pointer to **NullableString** |  | [optional] 
**Template** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewEmailContactChannel

`func NewEmailContactChannel(address string, ) *EmailContactChannel`

NewEmailContactChannel instantiates a new EmailContactChannel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEmailContactChannelWithDefaults

`func NewEmailContactChannelWithDefaults() *EmailContactChannel`

NewEmailContactChannelWithDefaults instantiates a new EmailContactChannel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *EmailContactChannel) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *EmailContactChannel) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *EmailContactChannel) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetContextAboutUser

`func (o *EmailContactChannel) GetContextAboutUser() string`

GetContextAboutUser returns the ContextAboutUser field if non-nil, zero value otherwise.

### GetContextAboutUserOk

`func (o *EmailContactChannel) GetContextAboutUserOk() (*string, bool)`

GetContextAboutUserOk returns a tuple with the ContextAboutUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContextAboutUser

`func (o *EmailContactChannel) SetContextAboutUser(v string)`

SetContextAboutUser sets ContextAboutUser field to given value.

### HasContextAboutUser

`func (o *EmailContactChannel) HasContextAboutUser() bool`

HasContextAboutUser returns a boolean if a field has been set.

### SetContextAboutUserNil

`func (o *EmailContactChannel) SetContextAboutUserNil(b bool)`

 SetContextAboutUserNil sets the value for ContextAboutUser to be an explicit nil

### UnsetContextAboutUser
`func (o *EmailContactChannel) UnsetContextAboutUser()`

UnsetContextAboutUser ensures that no value is present for ContextAboutUser, not even an explicit nil
### GetAdditionalRecipients

`func (o *EmailContactChannel) GetAdditionalRecipients() []EmailRecipient`

GetAdditionalRecipients returns the AdditionalRecipients field if non-nil, zero value otherwise.

### GetAdditionalRecipientsOk

`func (o *EmailContactChannel) GetAdditionalRecipientsOk() (*[]EmailRecipient, bool)`

GetAdditionalRecipientsOk returns a tuple with the AdditionalRecipients field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalRecipients

`func (o *EmailContactChannel) SetAdditionalRecipients(v []EmailRecipient)`

SetAdditionalRecipients sets AdditionalRecipients field to given value.

### HasAdditionalRecipients

`func (o *EmailContactChannel) HasAdditionalRecipients() bool`

HasAdditionalRecipients returns a boolean if a field has been set.

### SetAdditionalRecipientsNil

`func (o *EmailContactChannel) SetAdditionalRecipientsNil(b bool)`

 SetAdditionalRecipientsNil sets the value for AdditionalRecipients to be an explicit nil

### UnsetAdditionalRecipients
`func (o *EmailContactChannel) UnsetAdditionalRecipients()`

UnsetAdditionalRecipients ensures that no value is present for AdditionalRecipients, not even an explicit nil
### GetExperimentalSubjectLine

`func (o *EmailContactChannel) GetExperimentalSubjectLine() string`

GetExperimentalSubjectLine returns the ExperimentalSubjectLine field if non-nil, zero value otherwise.

### GetExperimentalSubjectLineOk

`func (o *EmailContactChannel) GetExperimentalSubjectLineOk() (*string, bool)`

GetExperimentalSubjectLineOk returns a tuple with the ExperimentalSubjectLine field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExperimentalSubjectLine

`func (o *EmailContactChannel) SetExperimentalSubjectLine(v string)`

SetExperimentalSubjectLine sets ExperimentalSubjectLine field to given value.

### HasExperimentalSubjectLine

`func (o *EmailContactChannel) HasExperimentalSubjectLine() bool`

HasExperimentalSubjectLine returns a boolean if a field has been set.

### SetExperimentalSubjectLineNil

`func (o *EmailContactChannel) SetExperimentalSubjectLineNil(b bool)`

 SetExperimentalSubjectLineNil sets the value for ExperimentalSubjectLine to be an explicit nil

### UnsetExperimentalSubjectLine
`func (o *EmailContactChannel) UnsetExperimentalSubjectLine()`

UnsetExperimentalSubjectLine ensures that no value is present for ExperimentalSubjectLine, not even an explicit nil
### GetExperimentalReferencesMessageId

`func (o *EmailContactChannel) GetExperimentalReferencesMessageId() string`

GetExperimentalReferencesMessageId returns the ExperimentalReferencesMessageId field if non-nil, zero value otherwise.

### GetExperimentalReferencesMessageIdOk

`func (o *EmailContactChannel) GetExperimentalReferencesMessageIdOk() (*string, bool)`

GetExperimentalReferencesMessageIdOk returns a tuple with the ExperimentalReferencesMessageId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExperimentalReferencesMessageId

`func (o *EmailContactChannel) SetExperimentalReferencesMessageId(v string)`

SetExperimentalReferencesMessageId sets ExperimentalReferencesMessageId field to given value.

### HasExperimentalReferencesMessageId

`func (o *EmailContactChannel) HasExperimentalReferencesMessageId() bool`

HasExperimentalReferencesMessageId returns a boolean if a field has been set.

### SetExperimentalReferencesMessageIdNil

`func (o *EmailContactChannel) SetExperimentalReferencesMessageIdNil(b bool)`

 SetExperimentalReferencesMessageIdNil sets the value for ExperimentalReferencesMessageId to be an explicit nil

### UnsetExperimentalReferencesMessageId
`func (o *EmailContactChannel) UnsetExperimentalReferencesMessageId()`

UnsetExperimentalReferencesMessageId ensures that no value is present for ExperimentalReferencesMessageId, not even an explicit nil
### GetExperimentalInReplyToMessageId

`func (o *EmailContactChannel) GetExperimentalInReplyToMessageId() string`

GetExperimentalInReplyToMessageId returns the ExperimentalInReplyToMessageId field if non-nil, zero value otherwise.

### GetExperimentalInReplyToMessageIdOk

`func (o *EmailContactChannel) GetExperimentalInReplyToMessageIdOk() (*string, bool)`

GetExperimentalInReplyToMessageIdOk returns a tuple with the ExperimentalInReplyToMessageId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExperimentalInReplyToMessageId

`func (o *EmailContactChannel) SetExperimentalInReplyToMessageId(v string)`

SetExperimentalInReplyToMessageId sets ExperimentalInReplyToMessageId field to given value.

### HasExperimentalInReplyToMessageId

`func (o *EmailContactChannel) HasExperimentalInReplyToMessageId() bool`

HasExperimentalInReplyToMessageId returns a boolean if a field has been set.

### SetExperimentalInReplyToMessageIdNil

`func (o *EmailContactChannel) SetExperimentalInReplyToMessageIdNil(b bool)`

 SetExperimentalInReplyToMessageIdNil sets the value for ExperimentalInReplyToMessageId to be an explicit nil

### UnsetExperimentalInReplyToMessageId
`func (o *EmailContactChannel) UnsetExperimentalInReplyToMessageId()`

UnsetExperimentalInReplyToMessageId ensures that no value is present for ExperimentalInReplyToMessageId, not even an explicit nil
### GetTemplate

`func (o *EmailContactChannel) GetTemplate() string`

GetTemplate returns the Template field if non-nil, zero value otherwise.

### GetTemplateOk

`func (o *EmailContactChannel) GetTemplateOk() (*string, bool)`

GetTemplateOk returns a tuple with the Template field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplate

`func (o *EmailContactChannel) SetTemplate(v string)`

SetTemplate sets Template field to given value.

### HasTemplate

`func (o *EmailContactChannel) HasTemplate() bool`

HasTemplate returns a boolean if a field has been set.

### SetTemplateNil

`func (o *EmailContactChannel) SetTemplateNil(b bool)`

 SetTemplateNil sets the value for Template to be an explicit nil

### UnsetTemplate
`func (o *EmailContactChannel) UnsetTemplate()`

UnsetTemplate ensures that no value is present for Template, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


